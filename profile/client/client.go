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
	pb "github.com/dvaumoron/puzzleprofileservice"
	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/grpcclient"
	loginservice "github.com/dvaumoron/puzzleweb/login/service"
	"github.com/dvaumoron/puzzleweb/profile/service"
	"go.uber.org/zap"
)

var _ service.AdvancedProfileService = ProfileClient{}

type ProfileClient struct {
	grpcclient.Client
	groupId     uint64
	userService loginservice.UserService
	authService adminservice.AuthService
}

func Make(serviceAddr string, logger *zap.Logger, groupId uint64, userService loginservice.UserService, authService adminservice.AuthService) ProfileClient {
	return ProfileClient{
		Client: grpcclient.Make(serviceAddr, logger), groupId: groupId,
		userService: userService, authService: authService,
	}
}

func (client ProfileClient) UpdateProfile(userId uint64, desc string, info map[string]string) error {
	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewProfileClient(conn).UpdateProfile(ctx, &pb.UserProfile{
		UserId: userId, Desc: desc, Info: info,
	})
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client ProfileClient) UpdatePicture(userId uint64, data []byte) error {
	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewProfileClient(conn).UpdatePicture(ctx, &pb.Picture{UserId: userId, Data: data})
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client ProfileClient) GetPicture(userId uint64) ([]byte, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewProfileClient(conn).GetPicture(ctx, &pb.UserId{Id: userId})
	if err != nil {
		return nil, common.LogOriginalError(client.Logger, err)
	}
	return response.Data, nil
}

func (client ProfileClient) GetProfiles(userIds []uint64) (map[uint64]service.UserProfile, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, common.LogOriginalError(client.Logger, err)
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
		return nil, common.LogOriginalError(client.Logger, err)
	}

	users, err := client.userService.GetUsers(userIds)
	if err != nil {
		return nil, err
	}

	profiles := map[uint64]service.UserProfile{}
	for _, profile := range response.List {
		userId := profile.UserId
		profiles[userId] = service.UserProfile{User: users[userId], Desc: profile.Desc, Info: profile.Info}
	}
	return profiles, err
}

// no right check
func (client ProfileClient) Delete(userId uint64) error {
	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewProfileClient(conn).Delete(ctx, &pb.UserId{Id: userId})
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client ProfileClient) ViewRight(userId uint64) error {
	return client.authService.AuthQuery(userId, client.groupId, adminservice.ActionAccess)
}
