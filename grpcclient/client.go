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
package grpcclient

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	serviceAddr string
	Logger      *zap.Logger
}

func Make(serviceAddr string, logger *zap.Logger) Client {
	return Client{serviceAddr: serviceAddr, Logger: logger}
}

func (client Client) Dial() (*grpc.ClientConn, error) {
	return grpc.Dial(client.serviceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func (Client) InitContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Second)
}
