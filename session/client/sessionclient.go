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

package sessionclient

import (
	"context"

	grpcclient "github.com/dvaumoron/puzzlegrpcclient"
	pb "github.com/dvaumoron/puzzlesessionservice"
	"github.com/dvaumoron/puzzleweb/common"
	sessionservice "github.com/dvaumoron/puzzleweb/session/service"
	"google.golang.org/grpc"
)

type sessionClient struct {
	grpcclient.Client
}

func New(serviceAddr string, dialOptions []grpc.DialOption) sessionservice.SessionService {
	return sessionClient{Client: grpcclient.Make(serviceAddr, dialOptions...)}
}

func (client sessionClient) Generate(ctx context.Context) (uint64, error) {
	conn, err := client.Dial()
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	response, err := pb.NewSessionClient(conn).Generate(
		ctx, &pb.SessionInfo{Info: map[string]string{}},
	)
	return response.GetId(), err
}

func (client sessionClient) Get(ctx context.Context, id uint64) (map[string]string, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	response, err := pb.NewSessionClient(conn).GetSessionInfo(ctx, &pb.SessionId{Id: id})
	return response.GetInfo(), err
}

func (client sessionClient) Update(ctx context.Context, id uint64, info map[string]string) error {
	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	response, err := pb.NewSessionClient(conn).UpdateSessionInfo(ctx, &pb.SessionUpdate{Id: id, Info: info})
	if err != nil {
		return err
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}
