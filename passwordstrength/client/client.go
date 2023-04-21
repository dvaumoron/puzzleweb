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
	"time"

	grpcclient "github.com/dvaumoron/puzzlegrpcclient"
	pb "github.com/dvaumoron/puzzlepassstrengthservice"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/passwordstrength/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type strengthClient struct {
	grpcclient.Client
	logger *zap.Logger
}

func New(serviceAddr string, dialOptions grpc.DialOption, timeOut time.Duration, logger *zap.Logger) service.PasswordStrengthService {
	return strengthClient{Client: grpcclient.Make(serviceAddr, dialOptions, timeOut), logger: logger}
}

func (client strengthClient) Validate(password string) (bool, error) {
	conn, err := client.Dial()
	if err != nil {
		return false, common.LogOriginalError(client.logger, err, "StrengthClient1")
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewPassstrengthClient(conn).Check(ctx, &pb.PasswordRequest{Password: password})
	if err != nil {
		return false, common.LogOriginalError(client.logger, err, "StrengthClient2")
	}
	return response.Success, nil
}

func (client strengthClient) GetRules(lang string) (string, error) {
	conn, err := client.Dial()
	if err != nil {
		return "", common.LogOriginalError(client.logger, err, "StrengthClient3")
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewPassstrengthClient(conn).GetRules(ctx, &pb.LangRequest{Lang: lang})
	if err != nil {
		return "", common.LogOriginalError(client.logger, err, "StrengthClient4")
	}
	return response.Description, nil
}
