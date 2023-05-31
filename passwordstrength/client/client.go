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
	grpcclient "github.com/dvaumoron/puzzlegrpcclient"
	pb "github.com/dvaumoron/puzzlepassstrengthservice"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/passwordstrength/service"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"google.golang.org/grpc"
)

type strengthClient struct {
	grpcclient.Client
}

func New(serviceAddr string, dialOptions []grpc.DialOption) service.PasswordStrengthService {
	return strengthClient{Client: grpcclient.Make(serviceAddr, dialOptions...)}
}

func (client strengthClient) Validate(logger otelzap.LoggerWithCtx, password string) (bool, error) {
	conn, err := client.Dial()
	if err != nil {
		return false, common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewPassstrengthClient(conn).Check(logger.Context(), &pb.PasswordRequest{Password: password})
	if err != nil {
		return false, common.LogOriginalError(logger, err)
	}
	return response.Success, nil
}

func (client strengthClient) GetRules(logger otelzap.LoggerWithCtx, lang string) (string, error) {
	conn, err := client.Dial()
	if err != nil {
		return "", common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewPassstrengthClient(conn).GetRules(logger.Context(), &pb.LangRequest{Lang: lang})
	if err != nil {
		return "", common.LogOriginalError(logger, err)
	}
	return response.Description, nil
}
