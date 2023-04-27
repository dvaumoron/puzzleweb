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
	"errors"
	"sort"
	"time"

	grpcclient "github.com/dvaumoron/puzzlegrpcclient"
	pb "github.com/dvaumoron/puzzleloginservice"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/login/service"
	strengthservice "github.com/dvaumoron/puzzleweb/passwordstrength/service"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"google.golang.org/grpc"
)

var errWeakPassword = errors.New("WeakPassword")

type loginClient struct {
	grpcclient.Client
	logger          *otelzap.Logger
	dateFormat      string
	saltService     service.SaltService
	strengthService strengthservice.PasswordStrengthService
}

func New(serviceAddr string, dialOptions grpc.DialOption, timeOut time.Duration, logger *otelzap.Logger, dateFormat string, saltService service.SaltService, strengthService strengthservice.PasswordStrengthService) service.FullLoginService {
	return loginClient{
		Client: grpcclient.Make(serviceAddr, dialOptions, timeOut), logger: logger, dateFormat: dateFormat,
		saltService: saltService, strengthService: strengthService,
	}
}

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

func (client loginClient) Verify(login string, password string) (bool, uint64, error) {
	salted, err := client.saltService.Salt(login, password)
	if err != nil {
		return false, 0, common.LogOriginalError(client.logger, err, "LoginClient1")
	}

	conn, err := client.Dial()
	if err != nil {
		return false, 0, common.LogOriginalError(client.logger, err, "LoginClient2")
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewLoginClient(conn).Verify(ctx, &pb.LoginRequest{Login: login, Salted: salted})
	if err != nil {
		return false, 0, common.LogOriginalError(client.logger, err, "LoginClient3")
	}
	return response.Success, response.Id, nil
}

func (client loginClient) Register(login string, password string) (bool, uint64, error) {
	strong, err := client.strengthService.Validate(password)
	if err != nil {
		return false, 0, err
	}
	if !strong {
		return false, 0, errWeakPassword
	}

	salted, err := client.saltService.Salt(login, password)
	if err != nil {
		return false, 0, common.LogOriginalError(client.logger, err, "LoginClient4")
	}

	conn, err := client.Dial()
	if err != nil {
		return false, 0, common.LogOriginalError(client.logger, err, "LoginClient5")
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewLoginClient(conn).Register(ctx, &pb.LoginRequest{Login: login, Salted: salted})
	if err != nil {
		return false, 0, common.LogOriginalError(client.logger, err, "LoginClient6")
	}
	return response.Success, response.Id, nil
}

// You should remove duplicate id in list
func (client loginClient) GetUsers(userIds []uint64) (map[uint64]service.User, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, common.LogOriginalError(client.logger, err, "LoginClient7")
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewLoginClient(conn).GetUsers(ctx, &pb.UserIds{Ids: userIds})
	if err != nil {
		return nil, common.LogOriginalError(client.logger, err, "LoginClient8")
	}

	logins := map[uint64]service.User{}
	for _, value := range response.List {
		logins[value.Id] = convertUser(value, client.dateFormat)
	}
	return logins, nil
}

func (client loginClient) ChangeLogin(userId uint64, oldLogin string, newLogin string, password string) error {
	oldSalted, err := client.saltService.Salt(oldLogin, password)
	if err != nil {
		return common.LogOriginalError(client.logger, err, "LoginClient9")
	}

	newSalted, err := client.saltService.Salt(newLogin, password)
	if err != nil {
		return common.LogOriginalError(client.logger, err, "LoginClient10")
	}

	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(client.logger, err, "LoginClient11")
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewLoginClient(conn).ChangeLogin(ctx, &pb.ChangeRequest{
		UserId: userId, NewLogin: newLogin, OldSalted: oldSalted, NewSalted: newSalted,
	})
	if err != nil {
		return common.LogOriginalError(client.logger, err, "LoginClient12")
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client loginClient) ChangePassword(userId uint64, login string, oldPassword string, newPassword string) error {
	strong, err := client.strengthService.Validate(newPassword)
	if err != nil {
		return err
	}
	if !strong {
		return errWeakPassword
	}

	oldSalted, err := client.saltService.Salt(login, oldPassword)
	if err != nil {
		return common.LogOriginalError(client.logger, err, "LoginClient13")
	}

	newSalted, err := client.saltService.Salt(login, newPassword)
	if err != nil {
		return common.LogOriginalError(client.logger, err, "LoginClient14")
	}

	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(client.logger, err, "LoginClient15")
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewLoginClient(conn).ChangePassword(ctx, &pb.ChangeRequest{
		UserId: userId, OldSalted: oldSalted, NewSalted: newSalted,
	})
	if err != nil {
		return common.LogOriginalError(client.logger, err, "LoginClient16")
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client loginClient) ListUsers(start uint64, end uint64, filter string) (uint64, []service.User, error) {
	conn, err := client.Dial()
	if err != nil {
		return 0, nil, common.LogOriginalError(client.logger, err, "LoginClient17")
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewLoginClient(conn).ListUsers(ctx, &pb.RangeRequest{
		Start: start, End: end, Filter: filter,
	})
	if err != nil {
		return 0, nil, common.LogOriginalError(client.logger, err, "LoginClient18")
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
func (client loginClient) Delete(userId uint64) error {
	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(client.logger, err, "LoginClient19")
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewLoginClient(conn).Delete(ctx, &pb.UserId{Id: userId})
	if err != nil {
		return common.LogOriginalError(client.logger, err, "LoginClient20")
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func convertUser(user *pb.User, dateFormat string) service.User {
	registredAt := time.Unix(user.RegistredAt, 0)
	return service.User{Id: user.Id, Login: user.Login, RegistredAt: registredAt.Format(dateFormat)}
}
