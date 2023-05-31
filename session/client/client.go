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
	pb "github.com/dvaumoron/puzzlesessionservice"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/session/service"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"google.golang.org/grpc"
)

type sessionClient struct {
	grpcclient.Client
}

func New(serviceAddr string, dialOptions []grpc.DialOption) service.SessionService {
	return sessionClient{Client: grpcclient.Make(serviceAddr, dialOptions...)}
}

func (client sessionClient) Generate(logger otelzap.LoggerWithCtx) (uint64, error) {
	conn, err := client.Dial()
	if err != nil {
		return 0, common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewSessionClient(conn).Generate(
		logger.Context(), &pb.SessionInfo{Info: map[string]string{}},
	)
	if err != nil {
		return 0, common.LogOriginalError(logger, err)
	}
	return response.Id, nil
}

func (client sessionClient) Get(logger otelzap.LoggerWithCtx, id uint64) (map[string]string, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewSessionClient(conn).GetSessionInfo(
		logger.Context(), &pb.SessionId{Id: id},
	)
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}
	return response.Info, nil
}

func (client sessionClient) Update(logger otelzap.LoggerWithCtx, id uint64, info map[string]string) error {
	conn, err := client.Dial()
	if err != nil {
		common.LogOriginalError(logger, err)
		return common.ErrTechnical
	}
	defer conn.Close()

	response, err := pb.NewSessionClient(conn).UpdateSessionInfo(logger.Context(), &pb.SessionUpdate{Id: id, Info: info})
	if err != nil {
		common.LogOriginalError(logger, err)
		return common.ErrTechnical
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}
