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
	"fmt"
	"sort"
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
	Id      uint64
	Creator *profileclient.Profile
	Date    string
	Text    string
}

type contentRequestKind func(pb.ForumClient, context.Context, *pb.SearchRequest) (*pb.Contents, error)
type deleteRequestKind func(pb.ForumClient, context.Context, *pb.IdRequest) (*pb.Confirm, error)

type sortableContents []*pb.Content

func (s sortableContents) Len() int {
	return len(s)
}

func (s sortableContents) Less(i, j int) bool {
	return s[i].CreatedAt > s[j].CreatedAt
}

func (s sortableContents) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func CreateThread(forumId uint64, groupId uint64, userId uint64, title string, message string) error {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionCreate)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(config.ForumServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrorTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client := pb.NewForumClient(conn)
	response, err := client.CreateThread(ctx, &pb.CreateRequest{
		ContainerId: forumId, UserId: userId, Text: title,
	})
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrorTechnical
	}
	if !response.Success {
		return common.ErrorUpdate
	}

	response, err = client.CreateMessage(ctx, &pb.CreateRequest{
		ContainerId: response.Id, UserId: userId, Text: message,
	})
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrorTechnical
	}
	if !response.Success {
		return common.ErrorUpdate
	}
	return nil
}

func CreateCommentThread(objectId uint64, groupId uint64, userId uint64, elemTitle string) error {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionCreate)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(config.ForumServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrorTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewForumClient(conn).CreateThread(ctx, &pb.CreateRequest{
		ContainerId: objectId, UserId: userId, Text: elemTitle,
	})
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrorTechnical
	}
	if !response.Success {
		return common.ErrorUpdate
	}
	return nil
}

func CreateMessage(groupId uint64, userId uint64, threadId uint64, message string) error {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionUpdate)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(config.ForumServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrorTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewForumClient(conn).CreateMessage(ctx, &pb.CreateRequest{
		ContainerId: threadId, UserId: userId, Text: message,
	})
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrorTechnical
	}
	if !response.Success {
		return common.ErrorUpdate
	}
	return nil
}

func GetThread(forumId uint64, groupId uint64, userId uint64, threadId uint64) (*ForumContent, error) {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionAccess)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.Dial(config.ForumServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrorTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewForumClient(conn).GetThread(ctx, &pb.IdRequest{
		ContainerId: forumId, Id: threadId,
	})
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrorTechnical
	}

	creatorId := response.UserId
	users, err := profileclient.GetProfiles([]uint64{creatorId})
	if err != nil {
		return nil, err
	}
	return convertContent(response, users[creatorId]), nil
}

func GetThreads(forumId uint64, groupId uint64, userId uint64, start uint64, end uint64, filter string) ([]*ForumContent, error) {
	return searchContent(
		groupId, userId, getThreads,
		&pb.SearchRequest{ContainerId: forumId, Start: start, End: end, Filter: filter},
	)
}

func GetCommentThreadId(objectId uint64, groupId uint64, userId uint64, elemTitle string) (uint64, error) {
	contents, err := GetThreads(objectId, groupId, userId, 0, 1, elemTitle)
	if err != nil {
		return 0, err
	}
	if len(contents) == 0 {
		common.LogOriginalError(fmt.Errorf("no comment thread found : %d, %s", objectId, elemTitle))
		return 0, common.ErrorTechnical
	}
	return contents[0].Id, nil
}

func GetMessages(groupId uint64, userId uint64, threadId uint64, start uint64, end uint64) ([]*ForumContent, error) {
	return searchContent(
		groupId, userId, getMessages,
		&pb.SearchRequest{ContainerId: threadId, Start: start, End: end},
	)
}

func DeleteThread(forumId uint64, groupId uint64, userId uint64, threadId uint64) error {
	return deleteContent(
		groupId, userId, deleteThread,
		&pb.IdRequest{ContainerId: forumId, Id: threadId},
	)
}

func DeleteMessage(groupId uint64, userId uint64, threadId uint64, messageId uint64) error {
	return deleteContent(
		groupId, userId, deleteMessage,
		&pb.IdRequest{ContainerId: threadId, Id: messageId},
	)
}

func searchContent(groupId uint64, userId uint64, kind contentRequestKind, search *pb.SearchRequest) ([]*ForumContent, error) {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionAccess)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.Dial(config.ForumServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrorTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := kind(pb.NewForumClient(conn), ctx, search)
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrorTechnical
	}
	list := response.List
	if len(list) == 0 {
		return nil, nil
	}
	return sortConvertContents(list)
}

func getThreads(client pb.ForumClient, ctx context.Context, search *pb.SearchRequest) (*pb.Contents, error) {
	return client.GetThreads(ctx, search)
}

func getMessages(client pb.ForumClient, ctx context.Context, search *pb.SearchRequest) (*pb.Contents, error) {
	return client.GetMessages(ctx, search)
}

func deleteContent(groupId uint64, userId uint64, kind deleteRequestKind, request *pb.IdRequest) error {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionDelete)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(config.ForumServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrorTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := kind(pb.NewForumClient(conn), ctx, request)
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrorTechnical
	}
	if !response.Success {
		return common.ErrorUpdate
	}
	return nil
}

func deleteThread(client pb.ForumClient, ctx context.Context, request *pb.IdRequest) (*pb.Confirm, error) {
	return client.DeleteThread(ctx, request)
}

func deleteMessage(client pb.ForumClient, ctx context.Context, request *pb.IdRequest) (*pb.Confirm, error) {
	return client.DeleteMessage(ctx, request)
}

func sortConvertContents(list []*pb.Content) ([]*ForumContent, error) {
	sort.Sort(sortableContents(list))

	size := len(list)
	// no duplicate check, there is one in GetProfiles
	userIds := make([]uint64, 0, size)
	for _, content := range list {
		userIds = append(userIds, content.UserId)
	}

	users, err := profileclient.GetProfiles(userIds)
	if err != nil {
		return nil, err
	}

	contents := make([]*ForumContent, 0, size)
	for _, content := range list {
		contents = append(contents, convertContent(content, users[content.UserId]))
	}
	return contents, nil
}

func convertContent(content *pb.Content, creator *profileclient.Profile) *ForumContent {
	createdAt := time.Unix(content.CreatedAt, 0)
	return &ForumContent{
		Id: content.Id, Creator: creator,
		Date: createdAt.Format(config.DateFormat), Text: content.Text,
	}
}
