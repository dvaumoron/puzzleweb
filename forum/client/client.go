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

	pb "github.com/dvaumoron/puzzleforumservice"
	rightclient "github.com/dvaumoron/puzzleweb/admin/client"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	profileclient "github.com/dvaumoron/puzzleweb/profile/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ForumContent struct {
	Creator *profileclient.Profile
	Date    string
	Text    string
}

func CreateThread(groupId uint64, userId uint64, title string, message string) error {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionCreate)
	if err == nil {
		var conn *grpc.ClientConn
		conn, err = grpc.Dial(config.ForumServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			defer conn.Close()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			var response *pb.Confirm
			client := pb.NewForumClient(conn)
			response, err = client.CreateThread(ctx, &pb.Content{
				ContainerId: groupId, UserId: userId, Text: title,
			})
			if err == nil {
				if response.Success {
					var response2 *pb.Confirm
					response2, err = client.CreateMessage(ctx, &pb.Content{
						ContainerId: response.Id, UserId: userId, Text: message,
					})
					if err == nil {
						if !response2.Success {
							err = common.ErrorUpdate
						}
					} else {
						common.LogOriginalError(err)
						err = common.ErrorTechnical
					}
				} else {
					err = common.ErrorUpdate
				}
			} else {
				common.LogOriginalError(err)
				err = common.ErrorTechnical
			}
		} else {
			common.LogOriginalError(err)
			err = common.ErrorTechnical
		}
	}
	return err
}

func CreateMessage(groupId uint64, userId uint64, threadId uint64, message string) error {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionUpdate)
	if err == nil {
		var conn *grpc.ClientConn
		conn, err = grpc.Dial(config.ForumServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			defer conn.Close()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			var response *pb.Confirm
			response, err = pb.NewForumClient(conn).CreateMessage(ctx, &pb.Content{
				ContainerId: threadId, UserId: userId, Text: message,
			})
			if err == nil {
				if !response.Success {
					err = common.ErrorUpdate
				}
			} else {
				common.LogOriginalError(err)
				err = common.ErrorTechnical
			}
		} else {
			common.LogOriginalError(err)
			err = common.ErrorTechnical
		}
	}
	return err
}

func GetThreads(groupId uint64, userId uint64, start uint64, end uint64, filter string) error {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionAccess)
	if err == nil {
		var conn *grpc.ClientConn
		conn, err = grpc.Dial(config.ForumServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			defer conn.Close()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			var response *pb.Contents
			response, err = pb.NewForumClient(conn).GetThreads(ctx, &pb.Search{
				ContainerId: groupId, Start: start, End: end,
			})
			if err == nil {

			} else {
				common.LogOriginalError(err)
				err = common.ErrorTechnical
			}
		} else {
			common.LogOriginalError(err)
			err = common.ErrorTechnical
		}
	}
	return err
}

func GetMessages(groupId uint64, userId uint64, threadId uint64, start uint64, end uint64) error {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionAccess)
	if err == nil {
		var conn *grpc.ClientConn
		conn, err = grpc.Dial(config.ForumServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			defer conn.Close()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			var response *pb.Contents
			response, err = pb.NewForumClient(conn).GetMessages(ctx, &pb.Search{
				ContainerId: threadId, Start: start, End: end,
			})
			if err == nil {

			} else {
				common.LogOriginalError(err)
				err = common.ErrorTechnical
			}
		} else {
			common.LogOriginalError(err)
			err = common.ErrorTechnical
		}
	}
	return err
}
