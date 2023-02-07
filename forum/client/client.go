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
	Creator profileclient.UserProfile
	Date    string
	Text    string
}

type deleteRequestKind func(pb.ForumClient, context.Context, *pb.IdRequest) (*pb.Response, error)

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

	conn, err := grpc.Dial(config.Shared.ForumServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewForumClient(conn).CreateThread(ctx, &pb.CreateRequest{
		ContainerId: forumId, UserId: userId, Title: title, Text: message,
	})
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func CreateCommentThread(objectId uint64, groupId uint64, userId uint64, elemTitle string) error {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionCreate)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(config.Shared.ForumServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewForumClient(conn).CreateThread(ctx, &pb.CreateRequest{
		ContainerId: objectId, UserId: userId, Text: elemTitle,
	})
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func CreateMessage(groupId uint64, userId uint64, threadId uint64, message string) error {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionUpdate)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(config.Shared.ForumServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewForumClient(conn).CreateMessage(ctx, &pb.CreateRequest{
		ContainerId: threadId, UserId: userId, Text: message,
	})
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func GetThread(forumId uint64, groupId uint64, userId uint64, threadId uint64, start uint64, end uint64) (uint64, ForumContent, []ForumContent, error) {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionAccess)
	if err != nil {
		return 0, ForumContent{}, nil, err
	}

	conn, err := grpc.Dial(config.Shared.ForumServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return 0, ForumContent{}, nil, common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client := pb.NewForumClient(conn)
	response, err := client.GetThread(ctx, &pb.IdRequest{
		ContainerId: forumId, Id: threadId,
	})
	if err != nil {
		common.LogOriginalError(err)
		return 0, ForumContent{}, nil, common.ErrTechnical
	}

	response2, err := client.GetMessages(ctx, &pb.SearchRequest{ContainerId: threadId, Start: start, End: end})
	if err != nil {
		common.LogOriginalError(err)
		return 0, ForumContent{}, nil, common.ErrTechnical
	}

	list := response2.List
	userIds := extractUserIds(list)
	threadCreatorId := response.UserId
	userIds = append(userIds, response.UserId)

	users, err := profileclient.GetProfiles(userIds)
	if err != nil {
		return 0, ForumContent{}, nil, err
	}

	thread := convertContent(response, users[threadCreatorId])
	messages := sortConvertContents(list, users)
	return response2.Total, thread, messages, nil
}

func GetThreads(forumId uint64, groupId uint64, userId uint64, start uint64, end uint64, filter string) (uint64, []ForumContent, error) {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionAccess)
	if err != nil {
		return 0, nil, err
	}

	conn, err := grpc.Dial(config.Shared.ForumServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return 0, nil, common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewForumClient(conn).GetThreads(ctx, &pb.SearchRequest{
		ContainerId: forumId, Start: start, End: end, Filter: filter,
	})
	if err != nil {
		common.LogOriginalError(err)
		return 0, nil, common.ErrTechnical
	}
	list := response.List
	if len(list) == 0 {
		return response.Total, nil, nil
	}

	users, err := profileclient.GetProfiles(extractUserIds(list))
	if err != nil {
		common.LogOriginalError(err)
		return 0, nil, common.ErrTechnical
	}
	return response.Total, sortConvertContents(list, users), err
}

func GetCommentThread(objectId uint64, groupId uint64, userId uint64, elemTitle string, start uint64, end uint64) (uint64, []ForumContent, error) {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionAccess)
	if err != nil {
		return 0, nil, err
	}

	conn, err := grpc.Dial(config.Shared.ForumServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return 0, nil, common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client := pb.NewForumClient(conn)
	response, err := client.GetThreads(ctx, &pb.SearchRequest{
		ContainerId: objectId, Start: 0, End: 1, Filter: elemTitle,
	})
	if err != nil {
		common.LogOriginalError(err)
		return 0, nil, common.ErrTechnical
	}
	if response.Total == 0 {
		common.LogOriginalError(fmt.Errorf("no comment thread found : %d, %s", objectId, elemTitle))
		return 0, nil, common.ErrTechnical
	}
	threadId := response.List[0].Id

	response2, err := client.GetMessages(ctx, &pb.SearchRequest{ContainerId: threadId, Start: start, End: end})
	if err != nil {
		common.LogOriginalError(err)
		return 0, nil, common.ErrTechnical
	}

	list := response2.List
	users, err := profileclient.GetProfiles(extractUserIds(list))
	if err != nil {
		return 0, nil, err
	}
	return response2.Total, sortConvertContents(list, users), nil
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

func deleteContent(groupId uint64, userId uint64, kind deleteRequestKind, request *pb.IdRequest) error {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionDelete)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(config.Shared.ForumServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := kind(pb.NewForumClient(conn), ctx, request)
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func deleteThread(client pb.ForumClient, ctx context.Context, request *pb.IdRequest) (*pb.Response, error) {
	return client.DeleteThread(ctx, request)
}

func deleteMessage(client pb.ForumClient, ctx context.Context, request *pb.IdRequest) (*pb.Response, error) {
	return client.DeleteMessage(ctx, request)
}

func sortConvertContents(list []*pb.Content, users map[uint64]profileclient.UserProfile) []ForumContent {
	sort.Sort(sortableContents(list))

	contents := make([]ForumContent, 0, len(list))
	for _, content := range list {
		contents = append(contents, convertContent(content, users[content.UserId]))
	}
	return contents
}

func convertContent(content *pb.Content, creator profileclient.UserProfile) ForumContent {
	createdAt := time.Unix(content.CreatedAt, 0)
	return ForumContent{
		Id: content.Id, Creator: creator,
		Date: createdAt.Format(config.Shared.DateFormat), Text: content.Text,
	}
}

// no duplicate check, there is one in GetProfiles
func extractUserIds(list []*pb.Content) []uint64 {
	userIds := make([]uint64, 0, len(list))
	for _, content := range list {
		userIds = append(userIds, content.UserId)
	}
	return userIds
}
