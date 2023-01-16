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
	"crypto/sha512"
	"encoding/hex"
	"time"

	pb "github.com/dvaumoron/puzzleloginservice"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type User struct {
	Id          uint64
	Login       string
	RegistredAt string
}

func salt(password string) string {
	// TODO improve the security
	sha512Hasher := sha512.New()
	sha512Hasher.Write([]byte(password))
	return hex.EncodeToString(sha512Hasher.Sum(nil))
}

func VerifyOrRegister(login string, password string, register bool) (uint64, bool, error) {
	conn, err := grpc.Dial(config.LoginServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	var id uint64
	success := false
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.Response
		client := pb.NewLoginClient(conn)
		request := &pb.LoginRequest{Login: login, Salted: salt(password)}
		if register {
			response, err = client.Register(ctx, request)
		} else {
			response, err = client.Verify(ctx, request)
		}

		if err == nil {
			id = response.Id
			success = response.Success
		} else {
			common.LogOriginalError(err)
			err = common.ErrorTechnical
		}
	} else {
		common.LogOriginalError(err)
		err = common.ErrorTechnical
	}
	return id, success, err
}

// You should remove duplicate id in list
func GetLogins(userIds []uint64) (map[uint64]string, error) {
	conn, err := grpc.Dial(config.LoginServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	var logins map[uint64]string
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.Users
		response, err = pb.NewLoginClient(conn).GetUsers(ctx, &pb.UserIds{Ids: userIds})

		if err == nil {
			logins = map[uint64]string{}
			for _, value := range response.List {
				logins[value.Id] = value.Login
			}
		} else {
			common.LogOriginalError(err)
			err = common.ErrorTechnical
		}
	} else {
		common.LogOriginalError(err)
		err = common.ErrorTechnical
	}
	return logins, err
}

func ChangeLogin(userId uint64, newLogin string, password string) error {
	conn, err := grpc.Dial(config.LoginServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.Response
		response, err = pb.NewLoginClient(conn).ChangeLogin(ctx, &pb.ChangeLoginRequest{
			UserId: userId, NewLogin: newLogin, Salted: salt(password),
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

func ChangePassword(userId uint64, oldPassword string, newPassword string) error {
	conn, err := grpc.Dial(config.LoginServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.Response
		response, err = pb.NewLoginClient(conn).ChangePassword(ctx, &pb.ChangePasswordRequest{
			UserId: userId, OldSalted: salt(oldPassword), NewSalted: salt(newPassword),
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

func ListUsers(start uint64, end uint64, filter string) (uint64, []*User, error) {
	conn, err := grpc.Dial(config.LoginServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	var total uint64
	var users []*User
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.Users
		response, err = pb.NewLoginClient(conn).ListUsers(ctx, &pb.RangeRequest{
			Start: start, End: end, Filter: filter,
		})

		if err == nil {
			total = response.Total
			list := response.List
			users = make([]*User, 0, len(list))
			for _, user := range list {
				users = append(users, &User{
					Id: user.Id, Login: user.Login,
				})
			}
		} else {
			common.LogOriginalError(err)
			err = common.ErrorTechnical
		}
	} else {
		common.LogOriginalError(err)
		err = common.ErrorTechnical
	}
	return total, users, err
}

// no right check
func DeleteUser(userId uint64) error {
	conn, err := grpc.Dial(config.LoginServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.Response
		response, err = pb.NewLoginClient(conn).Delete(ctx, &pb.UserId{Id: userId})
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
