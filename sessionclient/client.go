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
	"errors"
	"time"

	pb "github.com/dvaumoron/puzzlesessionservice"
	"github.com/dvaumoron/puzzleweb/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func createClient() (*grpc.ClientConn, pb.SessionClient, error) {
	conn, err := grpc.Dial(config.SessionServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}

	return conn, pb.NewSessionClient(conn), nil
}

func Generate() (uint64, error) {
	conn, client, err := createClient()
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	sessionId, err := client.Generate(ctx, &pb.SessionInfo{Info: map[string]string{}})
	if err != nil {
		return 0, err
	}
	return sessionId.Id, nil
}

func GetInfo(id uint64) (map[string]string, error) {
	conn, client, err := createClient()
	if err != nil {
		return map[string]string{}, err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	info, err := client.GetSessionInfo(ctx, &pb.SessionId{Id: id})
	if err != nil {
		return map[string]string{}, err
	}
	return info.Info, nil
}

func UpdateInfo(id uint64, info map[string]string) error {
	conn, client, err := createClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	strErr, err := client.UpdateSessionInfo(ctx, &pb.SessionUpdate{Id: &pb.SessionId{Id: id}, Info: &pb.SessionInfo{Info: info}})
	if err != nil {
		return err
	}
	errStr := strErr.Err
	if errStr == "" {
		return nil
	}
	return errors.New(errStr)
}
