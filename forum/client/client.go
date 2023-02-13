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
	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/forum/service"
	"github.com/dvaumoron/puzzleweb/grpcclient"
	profileservice "github.com/dvaumoron/puzzleweb/profile/service"
	"go.uber.org/zap"
)

type forumClient struct {
	grpcclient.Client
	forumId        uint64
	groupId        uint64
	dateFormat     string
	authService    adminservice.AuthService
	profileService profileservice.ProfileService
}

func New(serviceAddr string, logger *zap.Logger, forumId uint64, groupId uint64, dateFormat string, authService adminservice.AuthService, profileService profileservice.ProfileService) service.FullForumService {
	return forumClient{
		Client: grpcclient.Make(serviceAddr, logger), forumId: forumId, groupId: groupId,
		dateFormat: dateFormat, authService: authService, profileService: profileService,
	}
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

func (client forumClient) CreateThread(userId uint64, title string, message string) (uint64, error) {
	err := client.authService.AuthQuery(userId, client.groupId, adminservice.ActionCreate)
	if err != nil {
		return 0, err
	}

	conn, err := client.Dial()
	if err != nil {
		return 0, common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewForumClient(conn).CreateThread(ctx, &pb.CreateRequest{
		ContainerId: client.forumId, UserId: userId, Title: title, Text: message,
	})
	if err != nil {
		return 0, common.LogOriginalError(client.Logger, err)
	}
	if !response.Success {
		return 0, common.ErrUpdate
	}
	return response.Id, nil
}

func (client forumClient) CreateCommentThread(userId uint64, elemTitle string) error {
	err := client.authService.AuthQuery(userId, client.groupId, adminservice.ActionCreate)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewForumClient(conn).CreateThread(ctx, &pb.CreateRequest{
		ContainerId: client.forumId, UserId: userId, Text: elemTitle,
	})
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client forumClient) CreateMessage(userId uint64, threadId uint64, message string) error {
	err := client.authService.AuthQuery(userId, client.groupId, adminservice.ActionUpdate)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewForumClient(conn).CreateMessage(ctx, &pb.CreateRequest{
		ContainerId: threadId, UserId: userId, Text: message,
	})
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client forumClient) CreateComment(userId uint64, elemTitle string, comment string) error {
	err := client.authService.AuthQuery(userId, client.groupId, adminservice.ActionAccess)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	objectId := client.forumId
	forumClient := pb.NewForumClient(conn)
	response, err := searchCommentThread(forumClient, ctx, objectId, elemTitle)
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}

	var threadId uint64
	if response.Total == 0 {
		logCommentThreadNotFound(client.Logger, objectId, elemTitle)

		response2, err := forumClient.CreateThread(ctx, &pb.CreateRequest{
			ContainerId: client.forumId, UserId: userId, Title: elemTitle,
		})
		if err != nil {
			return common.LogOriginalError(client.Logger, err)
		}
		threadId = response2.Id
	} else {
		threadId = response.List[0].Id
	}

	response2, err := forumClient.CreateMessage(ctx, &pb.CreateRequest{
		ContainerId: threadId, UserId: userId, Text: comment,
	})
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	if !response2.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client forumClient) GetThread(userId uint64, threadId uint64, start uint64, end uint64, filter string) (uint64, service.ForumContent, []service.ForumContent, error) {
	err := client.authService.AuthQuery(userId, client.groupId, adminservice.ActionAccess)
	if err != nil {
		return 0, service.ForumContent{}, nil, err
	}

	conn, err := client.Dial()
	if err != nil {
		return 0, service.ForumContent{}, nil, common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	forumClient := pb.NewForumClient(conn)
	response, err := forumClient.GetThread(ctx, &pb.IdRequest{
		ContainerId: client.forumId, Id: threadId,
	})
	if err != nil {
		return 0, service.ForumContent{}, nil, common.LogOriginalError(client.Logger, err)
	}

	response2, err := forumClient.GetMessages(ctx, &pb.SearchRequest{
		ContainerId: threadId, Start: start, End: end, Filter: filter,
	})
	if err != nil {
		return 0, service.ForumContent{}, nil, common.LogOriginalError(client.Logger, err)
	}

	list := response2.List
	userIds := extractUserIds(list)
	threadCreatorId := response.UserId
	userIds = append(userIds, response.UserId)

	users, err := client.profileService.GetProfiles(userIds)
	if err != nil {
		return 0, service.ForumContent{}, nil, err
	}

	thread := convertContent(response, users[threadCreatorId], client.dateFormat)
	messages := sortConvertContents(list, users, client.dateFormat)
	return response2.Total, thread, messages, nil
}

func (client forumClient) GetThreads(userId uint64, start uint64, end uint64, filter string) (uint64, []service.ForumContent, error) {
	err := client.authService.AuthQuery(userId, client.groupId, adminservice.ActionAccess)
	if err != nil {
		return 0, nil, err
	}

	conn, err := client.Dial()
	if err != nil {
		return 0, nil, common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewForumClient(conn).GetThreads(ctx, &pb.SearchRequest{
		ContainerId: client.forumId, Start: start, End: end, Filter: filter,
	})
	if err != nil {
		return 0, nil, common.LogOriginalError(client.Logger, err)
	}
	list := response.List
	if len(list) == 0 {
		return response.Total, nil, nil
	}

	users, err := client.profileService.GetProfiles(extractUserIds(list))
	if err != nil {
		return 0, nil, err
	}
	return response.Total, sortConvertContents(list, users, client.dateFormat), err
}

func (client forumClient) GetCommentThread(userId uint64, elemTitle string, start uint64, end uint64) (uint64, []service.ForumContent, error) {
	err := client.authService.AuthQuery(userId, client.groupId, adminservice.ActionAccess)
	if err != nil {
		return 0, nil, err
	}

	conn, err := client.Dial()
	if err != nil {
		return 0, nil, common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	objectId := client.forumId
	forumClient := pb.NewForumClient(conn)
	response, err := searchCommentThread(forumClient, ctx, objectId, elemTitle)
	if err != nil {
		return 0, nil, common.LogOriginalError(client.Logger, err)
	}
	if response.Total == 0 {
		return 0, nil, logCommentThreadNotFound(client.Logger, objectId, elemTitle)
	}
	threadId := response.List[0].Id

	response2, err := forumClient.GetMessages(ctx, &pb.SearchRequest{
		ContainerId: threadId, Start: start, End: end,
	})
	if err != nil {
		return 0, nil, common.LogOriginalError(client.Logger, err)
	}

	list := response2.List
	users, err := client.profileService.GetProfiles(extractUserIds(list))
	if err != nil {
		return 0, nil, err
	}
	return response2.Total, sortConvertContents(list, users, client.dateFormat), nil
}

func (client forumClient) DeleteThread(userId uint64, threadId uint64) error {
	return client.deleteContent(
		userId, deleteThread, &pb.IdRequest{ContainerId: client.forumId, Id: threadId},
	)
}

func (client forumClient) DeleteCommentThread(userId uint64, elemTitle string) error {
	err := client.authService.AuthQuery(userId, client.groupId, adminservice.ActionDelete)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	objectId := client.forumId
	forumClient := pb.NewForumClient(conn)
	response, err := searchCommentThread(forumClient, ctx, objectId, elemTitle)
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	if response.Total == 0 {
		return nil
	}
	threadId := response.List[0].Id

	response2, err := forumClient.DeleteThread(ctx, &pb.IdRequest{ContainerId: objectId, Id: threadId})
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	if !response2.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client forumClient) DeleteMessage(userId uint64, threadId uint64, messageId uint64) error {
	return client.deleteContent(
		userId, deleteMessage, &pb.IdRequest{ContainerId: threadId, Id: messageId},
	)
}

func (client forumClient) DeleteComment(userId uint64, elemTitle string, commentId uint64) error {
	err := client.authService.AuthQuery(userId, client.groupId, adminservice.ActionDelete)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	objectId := client.forumId
	forumClient := pb.NewForumClient(conn)
	response, err := searchCommentThread(forumClient, ctx, objectId, elemTitle)
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	if response.Total == 0 {
		return logCommentThreadNotFound(client.Logger, objectId, elemTitle)
	}
	threadId := response.List[0].Id

	response2, err := forumClient.DeleteThread(ctx, &pb.IdRequest{ContainerId: threadId, Id: commentId})
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	if !response2.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client forumClient) CreateThreadRight(userId uint64) bool {
	return client.authService.AuthQuery(userId, client.groupId, adminservice.ActionCreate) == nil
}

func (client forumClient) CreateMessageRight(userId uint64) bool {
	return client.authService.AuthQuery(userId, client.groupId, adminservice.ActionUpdate) == nil
}

func (client forumClient) DeleteRight(userId uint64) bool {
	return client.authService.AuthQuery(userId, client.groupId, adminservice.ActionDelete) == nil
}

func (client forumClient) deleteContent(userId uint64, kind deleteRequestKind, request *pb.IdRequest) error {
	err := client.authService.AuthQuery(userId, client.groupId, adminservice.ActionDelete)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := kind(pb.NewForumClient(conn), ctx, request)
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func searchCommentThread(forumClient pb.ForumClient, ctx context.Context, objectId uint64, elemTitle string) (*pb.Contents, error) {
	return forumClient.GetThreads(ctx, &pb.SearchRequest{
		ContainerId: objectId, Start: 0, End: 1, Filter: elemTitle,
	})
}

func deleteThread(forumClient pb.ForumClient, ctx context.Context, request *pb.IdRequest) (*pb.Response, error) {
	return forumClient.DeleteThread(ctx, request)
}

func deleteMessage(forumClient pb.ForumClient, ctx context.Context, request *pb.IdRequest) (*pb.Response, error) {
	return forumClient.DeleteMessage(ctx, request)
}

func sortConvertContents(list []*pb.Content, users map[uint64]profileservice.UserProfile, dateFormat string) []service.ForumContent {
	sort.Sort(sortableContents(list))

	contents := make([]service.ForumContent, 0, len(list))
	for _, content := range list {
		contents = append(contents, convertContent(content, users[content.UserId], dateFormat))
	}
	return contents
}

func convertContent(content *pb.Content, creator profileservice.UserProfile, dateFormat string) service.ForumContent {
	createdAt := time.Unix(content.CreatedAt, 0)
	return service.ForumContent{
		Id: content.Id, Creator: creator, Date: createdAt.Format(dateFormat), Text: content.Text,
	}
}

func logCommentThreadNotFound(logger *zap.Logger, objectId uint64, elemTitle string) error {
	return common.LogOriginalError(logger, fmt.Errorf(
		"comment thread not found : %d, %s", objectId, elemTitle,
	))
}

// no duplicate check, there is one in GetProfiles
func extractUserIds(list []*pb.Content) []uint64 {
	userIds := make([]uint64, 0, len(list))
	for _, content := range list {
		userIds = append(userIds, content.UserId)
	}
	return userIds
}
