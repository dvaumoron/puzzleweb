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
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func salt(password string) string {
	// TODO improve the security
	sha512Hasher := sha512.New()
	sha512Hasher.Write([]byte(password))
	return hex.EncodeToString(sha512Hasher.Sum(nil))
}

func VerifyOrRegister(login string, password string, register bool) (uint64, bool, error) {
	conn, err := grpc.Dial(config.LoginServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	id := uint64(0)
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
			errors.LogOriginalError(err)
			err = errors.ErrorTechnical
		}
	} else {
		errors.LogOriginalError(err)
		err = errors.ErrorTechnical
	}
	return id, success, err
}

func GetLogins(userIds []uint64) (map[uint64]string, error) {
	conn, err := grpc.Dial(config.LoginServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	var logins map[uint64]string
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		userIds = removeDuplicateId(userIds)

		var response *pb.Logins
		response, err = pb.NewLoginClient(conn).ListLogins(ctx, &pb.UserIds{Ids: userIds})

		if err == nil {
			logins = make(map[uint64]string)
			for index, value := range response.List {
				logins[userIds[index]] = value
			}
		} else {
			errors.LogOriginalError(err)
			err = errors.ErrorTechnical
		}
	} else {
		errors.LogOriginalError(err)
		err = errors.ErrorTechnical
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
				err = errors.ErrorUpdate
			}
		} else {
			errors.LogOriginalError(err)
			err = errors.ErrorTechnical
		}
	} else {
		errors.LogOriginalError(err)
		err = errors.ErrorTechnical
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
				err = errors.ErrorUpdate
			}
		} else {
			errors.LogOriginalError(err)
			err = errors.ErrorTechnical
		}
	} else {
		errors.LogOriginalError(err)
		err = errors.ErrorTechnical
	}
	return err
}

func removeDuplicateId(ids []uint64) []uint64 {
	type empty struct{}
	idSet := make(map[uint64]empty)
	for _, id := range ids {
		idSet[id] = empty{}
	}
	cleanned := make([]uint64, 0, len(ids))
	for id := range idSet {
		cleanned = append(cleanned, id)
	}
	return cleanned
}
