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
	dateFormat      string
	saltService     service.SaltService
	strengthService strengthservice.PasswordStrengthService
}

func New(serviceAddr string, dialOptions []grpc.DialOption, dateFormat string, saltService service.SaltService, strengthService strengthservice.PasswordStrengthService) service.FullLoginService {
	return loginClient{
		Client: grpcclient.Make(serviceAddr, dialOptions...), dateFormat: dateFormat,
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

func (client loginClient) Verify(logger otelzap.LoggerWithCtx, login string, password string) (bool, uint64, error) {
	ctx := logger.Context()
	salted, err := client.saltService.Salt(ctx, login, password)
	if err != nil {
		return false, 0, common.LogOriginalError(logger, err)
	}

	conn, err := client.Dial()
	if err != nil {
		return false, 0, common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewLoginClient(conn).Verify(ctx, &pb.LoginRequest{Login: login, Salted: salted})
	if err != nil {
		return false, 0, common.LogOriginalError(logger, err)
	}
	return response.Success, response.Id, nil
}

func (client loginClient) Register(logger otelzap.LoggerWithCtx, login string, password string) (bool, uint64, error) {
	strong, err := client.strengthService.Validate(logger, password)
	if err != nil {
		return false, 0, err
	}
	if !strong {
		return false, 0, errWeakPassword
	}

	ctx := logger.Context()
	salted, err := client.saltService.Salt(ctx, login, password)
	if err != nil {
		return false, 0, common.LogOriginalError(logger, err)
	}

	conn, err := client.Dial()
	if err != nil {
		return false, 0, common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewLoginClient(conn).Register(ctx, &pb.LoginRequest{Login: login, Salted: salted})
	if err != nil {
		return false, 0, common.LogOriginalError(logger, err)
	}
	return response.Success, response.Id, nil
}

// You should remove duplicate id in list
func (client loginClient) GetUsers(logger otelzap.LoggerWithCtx, userIds []uint64) (map[uint64]service.User, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewLoginClient(conn).GetUsers(logger.Context(), &pb.UserIds{Ids: userIds})
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}

	logins := map[uint64]service.User{}
	for _, value := range response.List {
		logins[value.Id] = convertUser(value, client.dateFormat)
	}
	return logins, nil
}

func (client loginClient) ChangeLogin(logger otelzap.LoggerWithCtx, userId uint64, oldLogin string, newLogin string, password string) error {
	ctx := logger.Context()
	oldSalted, err := client.saltService.Salt(ctx, oldLogin, password)
	if err != nil {
		return common.LogOriginalError(logger, err)
	}

	newSalted, err := client.saltService.Salt(ctx, newLogin, password)
	if err != nil {
		return common.LogOriginalError(logger, err)
	}

	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewLoginClient(conn).ChangeLogin(ctx, &pb.ChangeRequest{
		UserId: userId, NewLogin: newLogin, OldSalted: oldSalted, NewSalted: newSalted,
	})
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client loginClient) ChangePassword(logger otelzap.LoggerWithCtx, userId uint64, login string, oldPassword string, newPassword string) error {
	strong, err := client.strengthService.Validate(logger, newPassword)
	if err != nil {
		return err
	}
	if !strong {
		return errWeakPassword
	}

	ctx := logger.Context()
	oldSalted, err := client.saltService.Salt(ctx, login, oldPassword)
	if err != nil {
		return common.LogOriginalError(logger, err)
	}

	newSalted, err := client.saltService.Salt(ctx, login, newPassword)
	if err != nil {
		return common.LogOriginalError(logger, err)
	}

	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewLoginClient(conn).ChangePassword(ctx, &pb.ChangeRequest{
		UserId: userId, OldSalted: oldSalted, NewSalted: newSalted,
	})
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client loginClient) ListUsers(logger otelzap.LoggerWithCtx, start uint64, end uint64, filter string) (uint64, []service.User, error) {
	conn, err := client.Dial()
	if err != nil {
		return 0, nil, common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewLoginClient(conn).ListUsers(logger.Context(), &pb.RangeRequest{
		Start: start, End: end, Filter: filter,
	})
	if err != nil {
		return 0, nil, common.LogOriginalError(logger, err)
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
func (client loginClient) Delete(logger otelzap.LoggerWithCtx, userId uint64) error {
	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewLoginClient(conn).Delete(logger.Context(), &pb.UserId{Id: userId})
	if err != nil {
		return common.LogOriginalError(logger, err)
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
