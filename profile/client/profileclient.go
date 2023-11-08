/*
 *
 * Copyright 2022 puzzleweb authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package profileclient

import (
	"context"

	grpcclient "github.com/dvaumoron/puzzlegrpcclient"
	pb "github.com/dvaumoron/puzzleprofileservice"
	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/common/log"
	loginservice "github.com/dvaumoron/puzzleweb/login/service"
	profileservice "github.com/dvaumoron/puzzleweb/profile/service"
	"google.golang.org/grpc"
)

type profileClient struct {
	grpcclient.Client
	groupId        uint64
	defaultPicture []byte
	userService    loginservice.UserService
	authService    adminservice.AuthService
	loggerGetter   log.LoggerGetter
}

func New(serviceAddr string, dialOptions []grpc.DialOption, groupId uint64, defaultPicture []byte, userService loginservice.UserService, authService adminservice.AuthService, loggerGetter log.LoggerGetter) profileservice.AdvancedProfileService {
	return profileClient{
		Client: grpcclient.Make(serviceAddr, dialOptions...), groupId: groupId, defaultPicture: defaultPicture,
		userService: userService, authService: authService, loggerGetter: loggerGetter,
	}
}

func (client profileClient) UpdateProfile(ctx context.Context, userId uint64, desc string, info map[string]string) error {
	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	response, err := pb.NewProfileClient(conn).UpdateProfile(ctx, &pb.UserProfile{
		UserId: userId, Desc: desc, Info: info,
	})
	if err != nil {
		return err
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client profileClient) UpdatePicture(ctx context.Context, userId uint64, data []byte) error {
	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	response, err := pb.NewProfileClient(conn).UpdatePicture(ctx, &pb.Picture{UserId: userId, Data: data})
	if err != nil {
		return err
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client profileClient) GetPicture(ctx context.Context, userId uint64) []byte {
	logger := client.loggerGetter.Logger(ctx)
	conn, err := client.Dial()
	if err != nil {
		common.LogOriginalError(logger, err)
		return client.defaultPicture
	}
	defer conn.Close()

	response, err := pb.NewProfileClient(conn).GetPicture(ctx, &pb.UserId{Id: userId})
	if err != nil {
		common.LogOriginalError(logger, err)
		return client.defaultPicture
	}
	return response.Data
}

func (client profileClient) GetProfiles(ctx context.Context, userIds []uint64) (map[uint64]profileservice.UserProfile, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// duplicate removal
	userIds = common.MakeSet(userIds).Slice()

	response, err := pb.NewProfileClient(conn).ListProfiles(ctx, &pb.UserIds{
		Ids: userIds,
	})
	if err != nil {
		return nil, err
	}

	users, err := client.userService.GetUsers(ctx, userIds)
	if err != nil {
		return nil, err
	}

	tempProfiles := map[uint64]profileservice.UserProfile{}
	for _, profile := range response.List {
		userId := profile.UserId
		tempProfiles[userId] = profileservice.UserProfile{User: users[userId], Desc: profile.Desc, Info: profile.Info}
	}

	profiles := map[uint64]profileservice.UserProfile{}
	for userId, user := range users {
		profile, ok := tempProfiles[userId]
		if ok {
			profiles[userId] = profile
		} else {
			// user who doesn't have profile data yet
			profiles[userId] = profileservice.UserProfile{User: user}
		}
	}
	return profiles, err
}

// no right check
func (client profileClient) Delete(ctx context.Context, userId uint64) error {
	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	response, err := pb.NewProfileClient(conn).Delete(ctx, &pb.UserId{Id: userId})
	if err != nil {
		return err
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client profileClient) ViewRight(ctx context.Context, userId uint64) error {
	return client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionAccess)
}
