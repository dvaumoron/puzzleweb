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

package forumclient

import (
	"cmp"
	"context"
	"slices"
	"time"

	pb "github.com/dvaumoron/puzzleforumservice"
	grpcclient "github.com/dvaumoron/puzzlegrpcclient"
	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/common/log"
	forumservice "github.com/dvaumoron/puzzleweb/forum/service"
	profileservice "github.com/dvaumoron/puzzleweb/profile/service"
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
	loggerGetter   log.LoggerGetter
}

func New(serviceAddr string, dialOptions []grpc.DialOption, forumId uint64, groupId uint64, dateFormat string, authService adminservice.AuthService, profileService profileservice.ProfileService, loggerGetter log.LoggerGetter) forumservice.FullForumService {
	return forumClient{
		Client: grpcclient.Make(serviceAddr, dialOptions...), forumId: forumId, groupId: groupId, dateFormat: dateFormat,
		authService: authService, profileService: profileService, loggerGetter: loggerGetter,
	}
}

type deleteRequestKind func(pb.ForumClient, context.Context, *pb.IdRequest) (*pb.Response, error)

func cmpContentDesc(a *pb.Content, b *pb.Content) int {
	return cmp.Compare(b.CreatedAt, a.CreatedAt)
}

func cmpContentAsc(a *pb.Content, b *pb.Content) int {
	return cmp.Compare(a.CreatedAt, b.CreatedAt)
}

func (client forumClient) CreateThread(ctx context.Context, userId uint64, title string, message string) (uint64, error) {
	err := client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionCreate)
	if err != nil {
		return 0, err
	}

	conn, err := client.Dial()
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	response, err := pb.NewForumClient(conn).CreateThread(ctx, &pb.CreateRequest{
		ContainerId: client.forumId, UserId: userId, Title: title, Text: message,
	})
	if err != nil {
		return 0, err
	}
	if !response.Success {
		return 0, common.ErrUpdate
	}
	return response.Id, nil
}

func (client forumClient) CreateCommentThread(ctx context.Context, userId uint64, elemTitle string) error {
	err := client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionCreate)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	response, err := pb.NewForumClient(conn).CreateThread(ctx, &pb.CreateRequest{
		ContainerId: client.forumId, UserId: userId, Title: elemTitle,
	})
	if err != nil {
		return err
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client forumClient) CreateMessage(ctx context.Context, userId uint64, threadId uint64, message string) error {
	err := client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionUpdate)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	response, err := pb.NewForumClient(conn).CreateMessage(ctx, &pb.CreateRequest{
		ContainerId: threadId, UserId: userId, Text: message,
	})
	if err != nil {
		return err
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client forumClient) CreateComment(ctx context.Context, userId uint64, elemTitle string, comment string) error {
	err := client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionAccess)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	objectId := client.forumId
	forumClient := pb.NewForumClient(conn)
	response, err := searchCommentThread(forumClient, ctx, objectId, elemTitle)
	if err != nil {
		return err
	}

	if response.Total == 0 {
		client.logCommentThreadNotFound(ctx, objectId, elemTitle)

		_, err := forumClient.CreateThread(ctx, &pb.CreateRequest{
			ContainerId: client.forumId, UserId: userId, Title: elemTitle, Text: comment,
		})
		return err
	}

	threadId := response.List[0].Id
	response2, err := forumClient.CreateMessage(ctx, &pb.CreateRequest{
		ContainerId: threadId, UserId: userId, Text: comment,
	})
	if err != nil {
		return err
	}
	if !response2.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client forumClient) GetThread(ctx context.Context, userId uint64, threadId uint64, start uint64, end uint64, filter string) (uint64, forumservice.ForumContent, []forumservice.ForumContent, error) {
	err := client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionAccess)
	if err != nil {
		return 0, forumservice.ForumContent{}, nil, err
	}

	conn, err := client.Dial()
	if err != nil {
		return 0, forumservice.ForumContent{}, nil, err
	}
	defer conn.Close()

	forumClient := pb.NewForumClient(conn)
	response, err := forumClient.GetThread(ctx, &pb.IdRequest{ContainerId: client.forumId, Id: threadId})
	if err != nil {
		return 0, forumservice.ForumContent{}, nil, err
	}

	response2, err := forumClient.GetMessages(ctx, &pb.SearchRequest{
		ContainerId: threadId, Start: start, End: end, Filter: filter,
	})
	if err != nil {
		return 0, forumservice.ForumContent{}, nil, err
	}

	list := response2.List
	userIds := extractUserIds(list)
	threadCreatorId := response.UserId
	userIds = append(userIds, response.UserId)

	users, err := client.profileService.GetProfiles(ctx, userIds)
	if err != nil {
		return 0, forumservice.ForumContent{}, nil, err
	}

	thread := convertContent(response, users[threadCreatorId], client.dateFormat)
	slices.SortFunc(list, cmpContentAsc)
	messages := convertContents(list, users, client.dateFormat)
	return response2.Total, thread, messages, nil
}

func (client forumClient) GetThreads(ctx context.Context, userId uint64, start uint64, end uint64, filter string) (uint64, []forumservice.ForumContent, error) {
	err := client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionAccess)
	if err != nil {
		return 0, nil, err
	}

	conn, err := client.Dial()
	if err != nil {
		return 0, nil, err
	}
	defer conn.Close()

	response, err := pb.NewForumClient(conn).GetThreads(ctx, &pb.SearchRequest{
		ContainerId: client.forumId, Start: start, End: end, Filter: filter,
	})
	if err != nil {
		return 0, nil, err
	}

	total := response.Total
	list := response.List
	if len(list) == 0 {
		return total, nil, nil
	}

	users, err := client.profileService.GetProfiles(ctx, extractUserIds(list))
	if err != nil {
		return 0, nil, err
	}
	slices.SortFunc(list, cmpContentDesc)
	return total, convertContents(list, users, client.dateFormat), nil
}

func (client forumClient) GetCommentThread(ctx context.Context, userId uint64, elemTitle string, start uint64, end uint64) (uint64, []forumservice.ForumContent, error) {
	err := client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionAccess)
	if err != nil {
		return 0, nil, err
	}

	conn, err := client.Dial()
	if err != nil {
		return 0, nil, err
	}
	defer conn.Close()

	objectId := client.forumId
	forumClient := pb.NewForumClient(conn)
	response, err := searchCommentThread(forumClient, ctx, objectId, elemTitle)
	if err != nil {
		return 0, nil, err
	}
	if response.Total == 0 {
		return 0, nil, client.logCommentThreadNotFound(ctx, objectId, elemTitle)
	}
	threadId := response.List[0].Id

	response2, err := forumClient.GetMessages(ctx, &pb.SearchRequest{
		ContainerId: threadId, Start: start, End: end,
	})
	if err != nil {
		return 0, nil, err
	}

	total := response2.Total
	list := response2.List
	if len(list) == 0 {
		return total, nil, nil
	}

	users, err := client.profileService.GetProfiles(ctx, extractUserIds(list))
	if err != nil {
		return 0, nil, err
	}
	slices.SortFunc(list, cmpContentAsc)
	return total, convertContents(list, users, client.dateFormat), nil
}

func (client forumClient) DeleteThread(ctx context.Context, userId uint64, threadId uint64) error {
	return client.deleteContent(ctx, userId, deleteThread, &pb.IdRequest{ContainerId: client.forumId, Id: threadId})
}

func (client forumClient) DeleteCommentThread(ctx context.Context, userId uint64, elemTitle string) error {
	err := client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionDelete)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	objectId := client.forumId
	forumClient := pb.NewForumClient(conn)
	response, err := searchCommentThread(forumClient, ctx, objectId, elemTitle)
	if err != nil {
		return err
	}
	if response.Total == 0 {
		return nil
	}
	threadId := response.List[0].Id

	response2, err := forumClient.DeleteThread(ctx, &pb.IdRequest{ContainerId: objectId, Id: threadId})
	if err != nil {
		return err
	}
	if !response2.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client forumClient) DeleteMessage(ctx context.Context, userId uint64, threadId uint64, messageId uint64) error {
	return client.deleteContent(
		ctx, userId, deleteMessage, &pb.IdRequest{ContainerId: threadId, Id: messageId},
	)
}

func (client forumClient) DeleteComment(ctx context.Context, userId uint64, elemTitle string, commentId uint64) error {
	err := client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionDelete)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	objectId := client.forumId
	forumClient := pb.NewForumClient(conn)
	response, err := searchCommentThread(forumClient, ctx, objectId, elemTitle)
	if err != nil {
		return err
	}
	if response.Total == 0 {
		return client.logCommentThreadNotFound(ctx, objectId, elemTitle)
	}
	threadId := response.List[0].Id

	response2, err := forumClient.DeleteMessage(ctx, &pb.IdRequest{ContainerId: threadId, Id: commentId})
	if err != nil {
		return err
	}
	if !response2.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client forumClient) CreateThreadRight(ctx context.Context, userId uint64) bool {
	return client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionCreate) == nil
}

func (client forumClient) CreateMessageRight(ctx context.Context, userId uint64) bool {
	return client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionUpdate) == nil
}

func (client forumClient) DeleteRight(ctx context.Context, userId uint64) bool {
	return client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionDelete) == nil
}

func (client forumClient) deleteContent(ctx context.Context, userId uint64, kind deleteRequestKind, request *pb.IdRequest) error {
	err := client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionDelete)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	response, err := kind(pb.NewForumClient(conn), ctx, request)
	if err != nil {
		return err
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client forumClient) logCommentThreadNotFound(ctx context.Context, objectId uint64, elemTitle string) error {
	client.loggerGetter.Logger(ctx).Warn(
		"comment thread not found", zap.Uint64("objectId", objectId), zap.String("elemTitle", elemTitle),
		zap.String(common.ReportingPlaceName, "ForumClient27"),
	)
	return common.ErrTechnical
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

func convertContents(list []*pb.Content, users map[uint64]profileservice.UserProfile, dateFormat string) []forumservice.ForumContent {
	contents := make([]forumservice.ForumContent, 0, len(list))
	for _, content := range list {
		contents = append(contents, convertContent(content, users[content.UserId], dateFormat))
	}
	return contents
}

func convertContent(content *pb.Content, creator profileservice.UserProfile, dateFormat string) forumservice.ForumContent {
	createdAt := time.Unix(content.CreatedAt, 0)
	return forumservice.ForumContent{
		Id: content.Id, Creator: creator, Date: createdAt.Format(dateFormat), Text: content.Text,
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
