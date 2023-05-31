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
	grpcclient "github.com/dvaumoron/puzzlegrpcclient"
	pb "github.com/dvaumoron/puzzleprofileservice"
	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/common"
	loginservice "github.com/dvaumoron/puzzleweb/login/service"
	"github.com/dvaumoron/puzzleweb/profile/service"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"google.golang.org/grpc"
)

type profileClient struct {
	grpcclient.Client
	groupId        uint64
	userService    loginservice.UserService
	authService    adminservice.AuthService
	defaultPicture []byte
}

func New(serviceAddr string, dialOptions []grpc.DialOption, groupId uint64, userService loginservice.UserService, authService adminservice.AuthService, defaultPicture []byte) service.AdvancedProfileService {
	return profileClient{
		Client: grpcclient.Make(serviceAddr, dialOptions...), groupId: groupId,
		userService: userService, authService: authService, defaultPicture: defaultPicture,
	}
}

func (client profileClient) UpdateProfile(logger otelzap.LoggerWithCtx, userId uint64, desc string, info map[string]string) error {
	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewProfileClient(conn).UpdateProfile(logger.Context(), &pb.UserProfile{
		UserId: userId, Desc: desc, Info: info,
	})
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client profileClient) UpdatePicture(logger otelzap.LoggerWithCtx, userId uint64, data []byte) error {
	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewProfileClient(conn).UpdatePicture(logger.Context(), &pb.Picture{UserId: userId, Data: data})
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client profileClient) GetPicture(logger otelzap.LoggerWithCtx, userId uint64) []byte {
	conn, err := client.Dial()
	if err != nil {
		common.LogOriginalError(logger, err)
		return client.defaultPicture
	}
	defer conn.Close()

	response, err := pb.NewProfileClient(conn).GetPicture(logger.Context(), &pb.UserId{Id: userId})
	if err != nil {
		common.LogOriginalError(logger, err)
		return client.defaultPicture
	}
	return response.Data
}

func (client profileClient) GetProfiles(logger otelzap.LoggerWithCtx, userIds []uint64) (map[uint64]service.UserProfile, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	// duplicate removal
	userIds = common.MakeSet(userIds).Slice()

	response, err := pb.NewProfileClient(conn).ListProfiles(logger.Context(), &pb.UserIds{
		Ids: userIds,
	})
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}

	users, err := client.userService.GetUsers(logger, userIds)
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
func (client profileClient) Delete(logger otelzap.LoggerWithCtx, userId uint64) error {
	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewProfileClient(conn).Delete(logger.Context(), &pb.UserId{Id: userId})
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client profileClient) ViewRight(logger otelzap.LoggerWithCtx, userId uint64) error {
	return client.authService.AuthQuery(logger, userId, client.groupId, adminservice.ActionAccess)
}
