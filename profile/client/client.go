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
	"context"
	"time"

	pb "github.com/dvaumoron/puzzleprofileservice"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	loginclient "github.com/dvaumoron/puzzleweb/login/client"
	"github.com/dvaumoron/puzzleweb/profile/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func UpdateProfile(userId uint64, desc string, info map[string]string) error {
	conn, err := grpc.Dial(config.Shared.ProfileServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return common.LogOriginalError(nil, err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewProfileClient(conn).UpdateProfile(ctx, &pb.UserProfile{
		UserId: userId, Desc: desc, Info: info,
	})
	if err != nil {
		return common.LogOriginalError(nil, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func UpdatePicture(userId uint64, data []byte) error {
	conn, err := grpc.Dial(config.Shared.ProfileServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return common.LogOriginalError(nil, err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewProfileClient(conn).UpdatePicture(ctx, &pb.Picture{UserId: userId, Data: data})
	if err != nil {
		return common.LogOriginalError(nil, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func GetPicture(userId uint64) ([]byte, error) {
	conn, err := grpc.Dial(config.Shared.ProfileServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, common.LogOriginalError(nil, err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewProfileClient(conn).GetPicture(ctx, &pb.UserId{Id: userId})
	if err != nil {
		return nil, common.LogOriginalError(nil, err)
	}
	return response.Data, nil
}

func GetProfiles(userIds []uint64) (map[uint64]service.UserProfile, error) {
	conn, err := grpc.Dial(config.Shared.ProfileServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, common.LogOriginalError(nil, err)
	}
	defer conn.Close()

	// duplicate removal
	userIds = common.MakeSet(userIds).Slice()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewProfileClient(conn).ListProfiles(ctx, &pb.UserIds{
		Ids: userIds,
	})
	if err != nil {
		return nil, common.LogOriginalError(nil, err)
	}

	users, err := loginclient.GetUsers(userIds)
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
func Delete(userId uint64) error {
	conn, err := grpc.Dial(config.Shared.ProfileServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return common.LogOriginalError(nil, err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewProfileClient(conn).Delete(ctx, &pb.UserId{Id: userId})
	if err != nil {
		return common.LogOriginalError(nil, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}
