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
	"time"

	pb "github.com/dvaumoron/puzzlesessionservice"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Generate() (uint64, error) {
	conn, err := grpc.Dial(config.SessionServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return 0, common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewSessionClient(conn).Generate(
		ctx, &pb.SessionInfo{Info: map[string]string{}},
	)
	if err != nil {
		common.LogOriginalError(err)
		return 0, common.ErrTechnical
	}
	return response.Id, nil
}

func GetSession(id uint64) (map[string]string, error) {
	return get(config.SessionServiceAddr, id)
}

func GetSettings(id uint64) (map[string]string, error) {
	return get(config.SettingsServiceAddr, id)
}

func get(addr string, id uint64) (map[string]string, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrTechnical

	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewSessionClient(conn).GetSessionInfo(
		ctx, &pb.SessionId{Id: id},
	)
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrTechnical

	}
	return response.Info, nil
}

func UpdateSession(id uint64, session map[string]string) error {
	return update(config.SessionServiceAddr, id, session)
}

func UpdateSettings(id uint64, settings map[string]string) error {
	return update(config.SettingsServiceAddr, id, settings)
}

func update(addr string, id uint64, info map[string]string) error {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client := pb.NewSessionClient(conn)
	response, err := client.UpdateSessionInfo(ctx, &pb.SessionUpdate{Id: id, Info: info})
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}
