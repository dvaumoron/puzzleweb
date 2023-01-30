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
	"encoding/base64"
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
	return base64.StdEncoding.EncodeToString(sha512Hasher.Sum(nil))
}

func VerifyOrRegister(login string, password string, register bool) (bool, uint64, error) {
	conn, err := grpc.Dial(config.LoginServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return false, 0, common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client := pb.NewLoginClient(conn)
	request := &pb.LoginRequest{Login: login, Salted: salt(password)}
	var response *pb.Response
	if register {
		response, err = client.Register(ctx, request)
	} else {
		response, err = client.Verify(ctx, request)
	}

	if err != nil {
		common.LogOriginalError(err)
		return false, 0, common.ErrTechnical
	}
	return response.Success, response.Id, nil
}

// You should remove duplicate id in list
func GetUsers(userIds []uint64) (map[uint64]User, error) {
	conn, err := grpc.Dial(config.LoginServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewLoginClient(conn).GetUsers(ctx, &pb.UserIds{Ids: userIds})
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrTechnical
	}

	logins := map[uint64]User{}
	for _, value := range response.List {
		logins[value.Id] = convertUser(value)
	}
	return logins, nil
}

func ChangeLogin(userId uint64, newLogin string, password string) error {
	conn, err := grpc.Dial(config.LoginServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewLoginClient(conn).ChangeLogin(ctx, &pb.ChangeLoginRequest{
		UserId: userId, NewLogin: newLogin, Salted: salt(password),
	})
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func ChangePassword(userId uint64, oldPassword string, newPassword string) error {
	conn, err := grpc.Dial(config.LoginServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewLoginClient(conn).ChangePassword(ctx, &pb.ChangePasswordRequest{
		UserId: userId, OldSalted: salt(oldPassword), NewSalted: salt(newPassword),
	})
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func ListUsers(start uint64, end uint64, filter string) (uint64, []User, error) {
	conn, err := grpc.Dial(config.LoginServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return 0, nil, common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewLoginClient(conn).ListUsers(ctx, &pb.RangeRequest{
		Start: start, End: end, Filter: filter,
	})
	if err != nil {
		common.LogOriginalError(err)
		return 0, nil, common.ErrTechnical
	}

	list := response.List
	users := make([]User, 0, len(list))
	for _, user := range list {
		users = append(users, convertUser(user))
	}
	return response.Total, users, nil
}

// no right check
func DeleteUser(userId uint64) error {
	conn, err := grpc.Dial(config.LoginServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewLoginClient(conn).Delete(ctx, &pb.UserId{Id: userId})
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func convertUser(user *pb.User) User {
	registredAt := time.Unix(user.RegistredAt, 0)
	return User{Id: user.Id, Login: user.Login, RegistredAt: registredAt.Format(config.DateFormat)}
}
