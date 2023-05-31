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
	"sort"
	"time"

	pb "github.com/dvaumoron/puzzleforumservice"
	grpcclient "github.com/dvaumoron/puzzlegrpcclient"
	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/forum/service"
	profileservice "github.com/dvaumoron/puzzleweb/profile/service"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type forumClient struct {
	grpcclient.Client
	forumId        uint64
	groupId        uint64
	dateFormat     string
	authService    adminservice.AuthService
	profileService profileservice.ProfileService
}

func New(serviceAddr string, dialOptions []grpc.DialOption, forumId uint64, groupId uint64, dateFormat string, authService adminservice.AuthService, profileService profileservice.ProfileService) service.FullForumService {
	return forumClient{
		Client: grpcclient.Make(serviceAddr, dialOptions...), forumId: forumId, groupId: groupId,
		dateFormat: dateFormat, authService: authService, profileService: profileService,
	}
}

type deleteRequestKind func(pb.ForumClient, otelzap.LoggerWithCtx, *pb.IdRequest) (*pb.Response, error)

type sortableContentsDesc []*pb.Content

func (s sortableContentsDesc) Len() int {
	return len(s)
}

func (s sortableContentsDesc) Less(i, j int) bool {
	return s[i].CreatedAt > s[j].CreatedAt
}

func (s sortableContentsDesc) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type sortableContentsAsc []*pb.Content

func (s sortableContentsAsc) Len() int {
	return len(s)
}

func (s sortableContentsAsc) Less(i, j int) bool {
	return s[i].CreatedAt < s[j].CreatedAt
}

func (s sortableContentsAsc) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (client forumClient) CreateThread(logger otelzap.LoggerWithCtx, userId uint64, title string, message string) (uint64, error) {
	err := client.authService.AuthQuery(logger, userId, client.groupId, adminservice.ActionCreate)
	if err != nil {
		return 0, err
	}

	conn, err := client.Dial()
	if err != nil {
		return 0, common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewForumClient(conn).CreateThread(logger.Context(), &pb.CreateRequest{
		ContainerId: client.forumId, UserId: userId, Title: title, Text: message,
	})
	if err != nil {
		return 0, common.LogOriginalError(logger, err)
	}
	if !response.Success {
		return 0, common.ErrUpdate
	}
	return response.Id, nil
}

func (client forumClient) CreateCommentThread(logger otelzap.LoggerWithCtx, userId uint64, elemTitle string) error {
	err := client.authService.AuthQuery(logger, userId, client.groupId, adminservice.ActionCreate)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewForumClient(conn).CreateThread(logger.Context(), &pb.CreateRequest{
		ContainerId: client.forumId, UserId: userId, Title: elemTitle,
	})
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client forumClient) CreateMessage(logger otelzap.LoggerWithCtx, userId uint64, threadId uint64, message string) error {
	err := client.authService.AuthQuery(logger, userId, client.groupId, adminservice.ActionUpdate)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewForumClient(conn).CreateMessage(logger.Context(), &pb.CreateRequest{
		ContainerId: threadId, UserId: userId, Text: message,
	})
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client forumClient) CreateComment(logger otelzap.LoggerWithCtx, userId uint64, elemTitle string, comment string) error {
	err := client.authService.AuthQuery(logger, userId, client.groupId, adminservice.ActionAccess)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	objectId := client.forumId
	forumClient := pb.NewForumClient(conn)
	response, err := searchCommentThread(forumClient, logger, objectId, elemTitle)
	if err != nil {
		return common.LogOriginalError(logger, err)
	}

	var threadId uint64
	if response.Total == 0 {
		logCommentThreadNotFound(logger, objectId, elemTitle)

		response2, err := forumClient.CreateThread(logger.Context(), &pb.CreateRequest{
			ContainerId: client.forumId, UserId: userId, Title: elemTitle,
		})
		if err != nil {
			return common.LogOriginalError(logger, err)
		}
		threadId = response2.Id
	} else {
		threadId = response.List[0].Id
	}

	response2, err := forumClient.CreateMessage(logger.Context(), &pb.CreateRequest{
		ContainerId: threadId, UserId: userId, Text: comment,
	})
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	if !response2.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client forumClient) GetThread(logger otelzap.LoggerWithCtx, userId uint64, threadId uint64, start uint64, end uint64, filter string) (uint64, service.ForumContent, []service.ForumContent, error) {
	err := client.authService.AuthQuery(logger, userId, client.groupId, adminservice.ActionAccess)
	if err != nil {
		return 0, service.ForumContent{}, nil, err
	}

	conn, err := client.Dial()
	if err != nil {
		return 0, service.ForumContent{}, nil, common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	forumClient := pb.NewForumClient(conn)
	response, err := forumClient.GetThread(logger.Context(), &pb.IdRequest{ContainerId: client.forumId, Id: threadId})
	if err != nil {
		return 0, service.ForumContent{}, nil, common.LogOriginalError(logger, err)
	}

	response2, err := forumClient.GetMessages(logger.Context(), &pb.SearchRequest{
		ContainerId: threadId, Start: start, End: end, Filter: filter,
	})
	if err != nil {
		return 0, service.ForumContent{}, nil, common.LogOriginalError(logger, err)
	}

	list := response2.List
	userIds := extractUserIds(list)
	threadCreatorId := response.UserId
	userIds = append(userIds, response.UserId)

	users, err := client.profileService.GetProfiles(logger, userIds)
	if err != nil {
		return 0, service.ForumContent{}, nil, err
	}

	thread := convertContent(response, users[threadCreatorId], client.dateFormat)
	sort.Sort(sortableContentsAsc(list))
	messages := convertContents(list, users, client.dateFormat)
	return response2.Total, thread, messages, nil
}

func (client forumClient) GetThreads(logger otelzap.LoggerWithCtx, userId uint64, start uint64, end uint64, filter string) (uint64, []service.ForumContent, error) {
	err := client.authService.AuthQuery(logger, userId, client.groupId, adminservice.ActionAccess)
	if err != nil {
		return 0, nil, err
	}

	conn, err := client.Dial()
	if err != nil {
		return 0, nil, common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewForumClient(conn).GetThreads(logger.Context(), &pb.SearchRequest{
		ContainerId: client.forumId, Start: start, End: end, Filter: filter,
	})
	if err != nil {
		return 0, nil, common.LogOriginalError(logger, err)
	}

	total := response.Total
	list := response.List
	if len(list) == 0 {
		return total, nil, nil
	}

	users, err := client.profileService.GetProfiles(logger, extractUserIds(list))
	if err != nil {
		return 0, nil, err
	}
	sort.Sort(sortableContentsDesc(list))
	return total, convertContents(list, users, client.dateFormat), nil
}

func (client forumClient) GetCommentThread(logger otelzap.LoggerWithCtx, userId uint64, elemTitle string, start uint64, end uint64) (uint64, []service.ForumContent, error) {
	err := client.authService.AuthQuery(logger, userId, client.groupId, adminservice.ActionAccess)
	if err != nil {
		return 0, nil, err
	}

	conn, err := client.Dial()
	if err != nil {
		return 0, nil, common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	objectId := client.forumId
	forumClient := pb.NewForumClient(conn)
	response, err := searchCommentThread(forumClient, logger, objectId, elemTitle)
	if err != nil {
		return 0, nil, common.LogOriginalError(logger, err)
	}
	if response.Total == 0 {
		return 0, nil, logCommentThreadNotFound(logger, objectId, elemTitle)
	}
	threadId := response.List[0].Id

	response2, err := forumClient.GetMessages(logger.Context(), &pb.SearchRequest{
		ContainerId: threadId, Start: start, End: end,
	})
	if err != nil {
		return 0, nil, common.LogOriginalError(logger, err)
	}

	total := response2.Total
	list := response2.List
	if len(list) == 0 {
		return total, nil, nil
	}

	users, err := client.profileService.GetProfiles(logger, extractUserIds(list))
	if err != nil {
		return 0, nil, err
	}
	sort.Sort(sortableContentsAsc(list))
	return total, convertContents(list, users, client.dateFormat), nil
}

func (client forumClient) DeleteThread(logger otelzap.LoggerWithCtx, userId uint64, threadId uint64) error {
	return client.deleteContent(logger, userId, deleteThread, &pb.IdRequest{ContainerId: client.forumId, Id: threadId})
}

func (client forumClient) DeleteCommentThread(logger otelzap.LoggerWithCtx, userId uint64, elemTitle string) error {
	err := client.authService.AuthQuery(logger, userId, client.groupId, adminservice.ActionDelete)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	objectId := client.forumId
	forumClient := pb.NewForumClient(conn)
	response, err := searchCommentThread(forumClient, logger, objectId, elemTitle)
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	if response.Total == 0 {
		return nil
	}
	threadId := response.List[0].Id

	response2, err := forumClient.DeleteThread(logger.Context(), &pb.IdRequest{ContainerId: objectId, Id: threadId})
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	if !response2.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client forumClient) DeleteMessage(logger otelzap.LoggerWithCtx, userId uint64, threadId uint64, messageId uint64) error {
	return client.deleteContent(
		logger, userId, deleteMessage, &pb.IdRequest{ContainerId: threadId, Id: messageId},
	)
}

func (client forumClient) DeleteComment(logger otelzap.LoggerWithCtx, userId uint64, elemTitle string, commentId uint64) error {
	err := client.authService.AuthQuery(logger, userId, client.groupId, adminservice.ActionDelete)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	objectId := client.forumId
	forumClient := pb.NewForumClient(conn)
	response, err := searchCommentThread(forumClient, logger, objectId, elemTitle)
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	if response.Total == 0 {
		return logCommentThreadNotFound(logger, objectId, elemTitle)
	}
	threadId := response.List[0].Id

	response2, err := forumClient.DeleteThread(logger.Context(), &pb.IdRequest{ContainerId: threadId, Id: commentId})
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	if !response2.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client forumClient) CreateThreadRight(logger otelzap.LoggerWithCtx, userId uint64) bool {
	return client.authService.AuthQuery(logger, userId, client.groupId, adminservice.ActionCreate) == nil
}

func (client forumClient) CreateMessageRight(logger otelzap.LoggerWithCtx, userId uint64) bool {
	return client.authService.AuthQuery(logger, userId, client.groupId, adminservice.ActionUpdate) == nil
}

func (client forumClient) DeleteRight(logger otelzap.LoggerWithCtx, userId uint64) bool {
	return client.authService.AuthQuery(logger, userId, client.groupId, adminservice.ActionDelete) == nil
}

func (client forumClient) deleteContent(logger otelzap.LoggerWithCtx, userId uint64, kind deleteRequestKind, request *pb.IdRequest) error {
	err := client.authService.AuthQuery(logger, userId, client.groupId, adminservice.ActionDelete)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := kind(pb.NewForumClient(conn), logger, request)
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func searchCommentThread(forumClient pb.ForumClient, logger otelzap.LoggerWithCtx, objectId uint64, elemTitle string) (*pb.Contents, error) {
	return forumClient.GetThreads(logger.Context(), &pb.SearchRequest{
		ContainerId: objectId, Start: 0, End: 1, Filter: elemTitle,
	})
}

func deleteThread(forumClient pb.ForumClient, logger otelzap.LoggerWithCtx, request *pb.IdRequest) (*pb.Response, error) {
	return forumClient.DeleteThread(logger.Context(), request)
}

func deleteMessage(forumClient pb.ForumClient, logger otelzap.LoggerWithCtx, request *pb.IdRequest) (*pb.Response, error) {
	return forumClient.DeleteMessage(logger.Context(), request)
}

func convertContents(list []*pb.Content, users map[uint64]profileservice.UserProfile, dateFormat string) []service.ForumContent {
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

func logCommentThreadNotFound(logger otelzap.LoggerWithCtx, objectId uint64, elemTitle string) error {
	logger.Warn(
		"comment thread not found", zap.Uint64("objectId", objectId), zap.String("elemTitle", elemTitle),
		zap.String(common.ReportingPlaceName, "ForumClient27"),
	)
	return common.ErrTechnical
}

// no duplicate check, there is one in GetProfiles
func extractUserIds(list []*pb.Content) []uint64 {
	userIds := make([]uint64, 0, len(list))
	for _, content := range list {
		userIds = append(userIds, content.UserId)
	}
	return userIds
}
