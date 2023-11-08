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

package loginclient

import (
	"context"
	"errors"
	"sort"
	"time"

	grpcclient "github.com/dvaumoron/puzzlegrpcclient"
	pb "github.com/dvaumoron/puzzleloginservice"
	"github.com/dvaumoron/puzzleweb/common"
	loginservice "github.com/dvaumoron/puzzleweb/login/service"
	strengthservice "github.com/dvaumoron/puzzleweb/passwordstrength/service"
	"google.golang.org/grpc"
)

var errWeakPassword = errors.New("WeakPassword")

type loginClient struct {
	grpcclient.Client
	dateFormat      string
	saltService     loginservice.SaltService
	strengthService strengthservice.PasswordStrengthService
}

func New(serviceAddr string, dialOptions []grpc.DialOption, dateFormat string, saltService loginservice.SaltService, strengthService strengthservice.PasswordStrengthService) loginservice.FullLoginService {
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

func (client loginClient) Verify(ctx context.Context, login string, password string) (bool, uint64, error) {
	salted, err := client.saltService.Salt(ctx, login, password)
	if err != nil {
		return false, 0, err
	}

	conn, err := client.Dial()
	if err != nil {
		return false, 0, err
	}
	defer conn.Close()

	response, err := pb.NewLoginClient(conn).Verify(ctx, &pb.LoginRequest{Login: login, Salted: salted})
	if err != nil {
		return false, 0, err
	}
	return response.Success, response.Id, nil
}

func (client loginClient) Register(ctx context.Context, login string, password string) (bool, uint64, error) {
	strong, err := client.strengthService.Validate(ctx, password)
	if err != nil {
		return false, 0, err
	}
	if !strong {
		return false, 0, errWeakPassword
	}

	salted, err := client.saltService.Salt(ctx, login, password)
	if err != nil {
		return false, 0, err
	}

	conn, err := client.Dial()
	if err != nil {
		return false, 0, err
	}
	defer conn.Close()

	response, err := pb.NewLoginClient(conn).Register(ctx, &pb.LoginRequest{Login: login, Salted: salted})
	if err != nil {
		return false, 0, err
	}
	return response.Success, response.Id, nil
}

// You should remove duplicate id in list
func (client loginClient) GetUsers(ctx context.Context, userIds []uint64) (map[uint64]loginservice.User, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	response, err := pb.NewLoginClient(conn).GetUsers(ctx, &pb.UserIds{Ids: userIds})
	if err != nil {
		return nil, err
	}

	logins := map[uint64]loginservice.User{}
	for _, value := range response.List {
		logins[value.Id] = convertUser(value, client.dateFormat)
	}
	return logins, nil
}

func (client loginClient) ChangeLogin(ctx context.Context, userId uint64, oldLogin string, newLogin string, password string) error {
	oldSalted, err := client.saltService.Salt(ctx, oldLogin, password)
	if err != nil {
		return err
	}

	newSalted, err := client.saltService.Salt(ctx, newLogin, password)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	response, err := pb.NewLoginClient(conn).ChangeLogin(ctx, &pb.ChangeRequest{
		UserId: userId, NewLogin: newLogin, OldSalted: oldSalted, NewSalted: newSalted,
	})
	if err != nil {
		return err
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client loginClient) ChangePassword(ctx context.Context, userId uint64, login string, oldPassword string, newPassword string) error {
	strong, err := client.strengthService.Validate(ctx, newPassword)
	if err != nil {
		return err
	}
	if !strong {
		return errWeakPassword
	}

	oldSalted, err := client.saltService.Salt(ctx, login, oldPassword)
	if err != nil {
		return err
	}

	newSalted, err := client.saltService.Salt(ctx, login, newPassword)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	response, err := pb.NewLoginClient(conn).ChangePassword(ctx, &pb.ChangeRequest{
		UserId: userId, OldSalted: oldSalted, NewSalted: newSalted,
	})
	if err != nil {
		return err
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client loginClient) ListUsers(ctx context.Context, start uint64, end uint64, filter string) (uint64, []loginservice.User, error) {
	conn, err := client.Dial()
	if err != nil {
		return 0, nil, err
	}
	defer conn.Close()

	response, err := pb.NewLoginClient(conn).ListUsers(ctx, &pb.RangeRequest{
		Start: start, End: end, Filter: filter,
	})
	if err != nil {
		return 0, nil, err
	}

	list := response.List
	sort.Sort(sortableContents(list))
	users := make([]loginservice.User, 0, len(list))
	for _, user := range list {
		users = append(users, convertUser(user, client.dateFormat))
	}
	return response.Total, users, nil
}

// no right check
func (client loginClient) Delete(ctx context.Context, userId uint64) error {
	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	response, err := pb.NewLoginClient(conn).Delete(ctx, &pb.UserId{Id: userId})
	if err != nil {
		return err
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func convertUser(user *pb.User, dateFormat string) loginservice.User {
	registredAt := time.Unix(user.RegistredAt, 0)
	return loginservice.User{Id: user.Id, Login: user.Login, RegistredAt: registredAt.Format(dateFormat)}
}
