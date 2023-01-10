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
	"github.com/dvaumoron/puzzleweb/login/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Profile struct {
	UserId uint64
	Login  string
	Desc   string
	Info   map[string]string
}

func UpdateProfile(userId uint64, desc string, info map[string]string) error {
	conn, err := grpc.Dial(config.ProfileServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.Confirm
		response, err = pb.NewProfileClient(conn).UpdateProfile(ctx, &pb.UserProfile{
			UserId: userId, Desc: desc, Info: info,
		})
		if err == nil {
			if !response.Success {
				err = common.ErrorUpdate
			}
		} else {
			common.LogOriginalError(err)
			err = common.ErrorTechnical
		}
	} else {
		common.LogOriginalError(err)
		err = common.ErrorTechnical
	}
	return err
}

func UpdatePicture(userId uint64, data []byte) error {
	conn, err := grpc.Dial(config.ProfileServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.Confirm
		response, err = pb.NewProfileClient(conn).UpdatePicture(ctx, &pb.Picture{
			UserId: userId, Data: data,
		})
		if err == nil {
			if !response.Success {
				err = common.ErrorUpdate
			}
		} else {
			common.LogOriginalError(err)
			err = common.ErrorTechnical
		}
	} else {
		common.LogOriginalError(err)
		err = common.ErrorTechnical
	}
	return err
}

func GetPicture(userId uint64) ([]byte, error) {
	conn, err := grpc.Dial(config.ProfileServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	var data []byte
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.Picture
		response, err = pb.NewProfileClient(conn).GetPicture(ctx, &pb.UserId{Id: userId})
		if err == nil {
			data = response.Data
		} else {
			common.LogOriginalError(err)
			err = common.ErrorTechnical
		}
	} else {
		common.LogOriginalError(err)
		err = common.ErrorTechnical
	}
	return data, err
}

func GetProfiles(userIds []uint64) (map[uint64]*Profile, error) {
	conn, err := grpc.Dial(config.ProfileServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	var profiles map[uint64]*Profile
	if err == nil {
		defer conn.Close()

		userIds = common.MakeSet(userIds...).Slice()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.UserProfiles
		response, err = pb.NewProfileClient(conn).ListProfiles(ctx, &pb.UserIds{
			Ids: userIds,
		})
		if err == nil {
			var logins map[uint64]string
			logins, err = client.GetLogins(userIds)
			if err == nil {
				profiles = map[uint64]*Profile{}
				for _, profile := range response.List {
					userId := profile.UserId
					profiles[userId] = &Profile{
						UserId: userId, Login: logins[userId],
						Desc: profile.Desc, Info: profile.Info,
					}
				}
			}
		} else {
			common.LogOriginalError(err)
			err = common.ErrorTechnical
		}
	} else {
		common.LogOriginalError(err)
		err = common.ErrorTechnical
	}
	return profiles, err
}
