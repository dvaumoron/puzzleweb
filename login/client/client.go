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
	"sort"
	"time"

	pb "github.com/dvaumoron/puzzleloginservice"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/grpcclient"
	"github.com/dvaumoron/puzzleweb/login/service"
	"go.uber.org/zap"
)

// check matching with interface
var _ service.FullLoginService = LoginClient{}

type LoginClient struct {
	grpcclient.Client
	dateFormat  string
	saltService service.SaltService
}

func Make(serviceAddr string, logger *zap.Logger, dateFormat string, saltService service.SaltService) LoginClient {
	return LoginClient{
		Client: grpcclient.Make(serviceAddr, logger), dateFormat: dateFormat, saltService: saltService,
	}
}

type requestKind func(pb.LoginClient, context.Context, *pb.LoginRequest) (*pb.Response, error)

type sortableContents []*pb.User

func (s sortableContents) Len() int {
	return len(s)
}

func (s sortableContents) Less(i, j int) bool {
	return s[i].Login < s[j].Login
}

func (s sortableContents) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (client LoginClient) Verify(login string, password string) (bool, uint64, error) {
	return client.verifyOrRegister(login, password, verify)
}

func (client LoginClient) Register(login string, password string) (bool, uint64, error) {
	return client.verifyOrRegister(login, password, register)
}

// You should remove duplicate id in list
func (client LoginClient) GetUsers(userIds []uint64) (map[uint64]service.User, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewLoginClient(conn).GetUsers(ctx, &pb.UserIds{Ids: userIds})
	if err != nil {
		return nil, common.LogOriginalError(client.Logger, err)
	}

	logins := map[uint64]service.User{}
	for _, value := range response.List {
		logins[value.Id] = convertUser(value, client.dateFormat)
	}
	return logins, nil
}

func (client LoginClient) ChangeLogin(userId uint64, oldLogin string, newLogin string, password string) error {
	oldSalted, err := client.saltService.Salt(oldLogin, password)
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}

	newSalted, err := client.saltService.Salt(newLogin, password)
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}

	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewLoginClient(conn).ChangeLogin(ctx, &pb.ChangeRequest{
		UserId: userId, NewLogin: newLogin, OldSalted: oldSalted, NewSalted: newSalted,
	})
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client LoginClient) ChangePassword(userId uint64, login string, oldPassword string, newPassword string) error {
	oldSalted, err := client.saltService.Salt(login, oldPassword)
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}

	newSalted, err := client.saltService.Salt(login, newPassword)
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}

	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewLoginClient(conn).ChangePassword(ctx, &pb.ChangeRequest{
		UserId: userId, OldSalted: oldSalted, NewSalted: newSalted,
	})
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client LoginClient) ListUsers(start uint64, end uint64, filter string) (uint64, []service.User, error) {
	conn, err := client.Dial()
	if err != nil {
		return 0, nil, common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewLoginClient(conn).ListUsers(ctx, &pb.RangeRequest{
		Start: start, End: end, Filter: filter,
	})
	if err != nil {
		return 0, nil, common.LogOriginalError(client.Logger, err)
	}

	list := response.List
	sort.Sort(sortableContents(list))
	users := make([]service.User, 0, len(list))
	for _, user := range list {
		users = append(users, convertUser(user, client.dateFormat))
	}
	return response.Total, users, nil
}

// no right check
func (client LoginClient) Delete(userId uint64) error {
	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewLoginClient(conn).Delete(ctx, &pb.UserId{Id: userId})
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client LoginClient) verifyOrRegister(login string, password string, kind requestKind) (bool, uint64, error) {
	salted, err := client.saltService.Salt(login, password)
	if err != nil {
		return false, 0, common.LogOriginalError(client.Logger, err)
	}

	conn, err := client.Dial()
	if err != nil {
		return false, 0, common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := kind(pb.NewLoginClient(conn), ctx, &pb.LoginRequest{Login: login, Salted: salted})
	if err != nil {
		return false, 0, common.LogOriginalError(client.Logger, err)
	}
	return response.Success, response.Id, nil
}

func verify(loginClient pb.LoginClient, ctx context.Context, request *pb.LoginRequest) (*pb.Response, error) {
	return loginClient.Verify(ctx, request)
}

func register(loginClient pb.LoginClient, ctx context.Context, request *pb.LoginRequest) (*pb.Response, error) {
	return loginClient.Register(ctx, request)
}

func convertUser(user *pb.User, dateFormat string) service.User {
	registredAt := time.Unix(user.RegistredAt, 0)
	return service.User{Id: user.Id, Login: user.Login, RegistredAt: registredAt.Format(dateFormat)}
}
