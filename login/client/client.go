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

	pb "github.com/dvaumoron/puzzleloginservice"
	"github.com/dvaumoron/puzzleweb/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func createClient() (*grpc.ClientConn, pb.LoginClient, error) {
	conn, err := grpc.Dial(config.LoginServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}

	return conn, pb.NewLoginClient(conn), nil
}

func Validate(login, password string) (uint64, bool, error) {
	conn, client, err := createClient()
	if err != nil {
		return 0, false, err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	response, err := client.Verify(ctx, &pb.LoginRequest{Login: login, Salted: password})
	if err != nil {
		return 0, false, err
	}
	return response.Id, response.Success, nil
}
