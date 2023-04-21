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
package client

import (
	"time"

	grpcclient "github.com/dvaumoron/puzzlegrpcclient"
	pb "github.com/dvaumoron/puzzleprofileservice"
	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/common"
	loginservice "github.com/dvaumoron/puzzleweb/login/service"
	"github.com/dvaumoron/puzzleweb/profile/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type profileClient struct {
	grpcclient.Client
	logger         *zap.Logger
	groupId        uint64
	userService    loginservice.UserService
	authService    adminservice.AuthService
	defaultPicture []byte
}

func New(serviceAddr string, dialOptions grpc.DialOption, timeOut time.Duration, logger *zap.Logger, groupId uint64, userService loginservice.UserService, authService adminservice.AuthService, defaultPicture []byte) service.AdvancedProfileService {
	return profileClient{
		Client: grpcclient.Make(serviceAddr, dialOptions, timeOut), logger: logger, groupId: groupId,
		userService: userService, authService: authService, defaultPicture: defaultPicture,
	}
}

func (client profileClient) UpdateProfile(userId uint64, desc string, info map[string]string) error {
	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(client.logger, err, "ProfileClient1")
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewProfileClient(conn).UpdateProfile(ctx, &pb.UserProfile{
		UserId: userId, Desc: desc, Info: info,
	})
	if err != nil {
		return common.LogOriginalError(client.logger, err, "ProfileClient2")
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client profileClient) UpdatePicture(userId uint64, data []byte) error {
	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(client.logger, err, "ProfileClient3")
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewProfileClient(conn).UpdatePicture(ctx, &pb.Picture{UserId: userId, Data: data})
	if err != nil {
		return common.LogOriginalError(client.logger, err, "ProfileClient4")
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client profileClient) GetPicture(userId uint64) []byte {
	conn, err := client.Dial()
	if err != nil {
		common.LogOriginalError(client.logger, err, "ProfileClient5")
		return client.defaultPicture
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewProfileClient(conn).GetPicture(ctx, &pb.UserId{Id: userId})
	if err != nil {
		common.LogOriginalError(client.logger, err, "ProfileClient6")
		return client.defaultPicture
	}
	return response.Data
}

func (client profileClient) GetProfiles(userIds []uint64) (map[uint64]service.UserProfile, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, common.LogOriginalError(client.logger, err, "ProfileClient7")
	}
	defer conn.Close()

	// duplicate removal
	userIds = common.MakeSet(userIds).Slice()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewProfileClient(conn).ListProfiles(ctx, &pb.UserIds{
		Ids: userIds,
	})
	if err != nil {
		return nil, common.LogOriginalError(client.logger, err, "ProfileClient8")
	}

	users, err := client.userService.GetUsers(userIds)
	if err != nil {
		return nil, err
	}

	tempProfiles := map[uint64]service.UserProfile{}
	for _, profile := range response.List {
		userId := profile.UserId
		tempProfiles[userId] = service.UserProfile{User: users[userId], Desc: profile.Desc, Info: profile.Info}
	}

	profiles := map[uint64]service.UserProfile{}
	for userId, user := range users {
		profile, ok := tempProfiles[userId]
		if ok {
			profiles[userId] = profile
		} else {
			// user who doesn't have profile data yet
			profiles[userId] = service.UserProfile{User: user}
		}
	}
	return profiles, err
}

// no right check
func (client profileClient) Delete(userId uint64) error {
	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(client.logger, err, "ProfileClient9")
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewProfileClient(conn).Delete(ctx, &pb.UserId{Id: userId})
	if err != nil {
		return common.LogOriginalError(client.logger, err, "ProfileClient10")
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client profileClient) ViewRight(userId uint64) error {
	return client.authService.AuthQuery(userId, client.groupId, adminservice.ActionAccess)
}
