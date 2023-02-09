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
	"html/template"
	"sort"
	"time"

	pb "github.com/dvaumoron/puzzleblogservice"
	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/blog/service"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/grpcclient"
	profileservice "github.com/dvaumoron/puzzleweb/profile/service"
	"go.uber.org/zap"
)

// check matching with interface
var _ service.BlogService = BlogClient{}

type BlogClient struct {
	grpcclient.Client
	blogId         uint64
	groupId        uint64
	dateFormat     string
	authService    adminservice.AuthService
	profileService profileservice.ProfileService
}

func Make(serviceAddr string, logger *zap.Logger, blogId uint64, groupId uint64, dateFormat string, authService adminservice.AuthService, profileService profileservice.ProfileService) BlogClient {
	return BlogClient{
		Client: grpcclient.Make(serviceAddr, logger), blogId: blogId, groupId: groupId,
		dateFormat: dateFormat, authService: authService, profileService: profileService,
	}
}

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

func (client BlogClient) CreatePost(userId uint64, title string, content string) error {
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

	response, err := pb.NewBlogClient(conn).CreatePost(ctx, &pb.CreateRequest{
		BlogId: client.blogId, UserId: userId, Title: title, Text: content,
	})
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client BlogClient) GetPost(userId uint64, postId uint64) (service.BlogPost, error) {
	err := client.authService.AuthQuery(userId, client.groupId, adminservice.ActionAccess)
	if err != nil {
		return service.BlogPost{}, err
	}

	conn, err := client.Dial()
	if err != nil {
		return service.BlogPost{}, common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewBlogClient(conn).GetPost(ctx, &pb.IdRequest{
		BlogId: client.blogId, PostId: postId,
	})
	if err != nil {
		return service.BlogPost{}, common.LogOriginalError(client.Logger, err)
	}

	creatorId := response.UserId
	users, err := client.profileService.GetProfiles([]uint64{creatorId})
	if err != nil {
		return service.BlogPost{}, err
	}
	return convertPost(response, users[creatorId], client.dateFormat), nil
}

func (client BlogClient) GetPosts(userId uint64, start uint64, end uint64, filter string) ([]service.BlogPost, error) {
	err := client.authService.AuthQuery(userId, client.groupId, adminservice.ActionAccess)
	if err != nil {
		return nil, err
	}

	conn, err := client.Dial()
	if err != nil {
		return nil, common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	response, err := pb.NewBlogClient(conn).GetPosts(ctx, &pb.SearchRequest{
		BlogId: client.blogId, Start: start, End: end, Filter: filter,
	})
	if err != nil {
		return nil, common.LogOriginalError(client.Logger, err)
	}
	list := response.List
	if len(list) == 0 {
		return nil, nil
	}
	return client.sortConvertPosts(list)
}

func (client BlogClient) DeletePost(userId uint64, postId uint64) error {
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

	response, err := pb.NewBlogClient(conn).DeletePost(ctx, &pb.IdRequest{
		BlogId: client.blogId, PostId: postId,
	})
	if err != nil {
		return common.LogOriginalError(client.Logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client BlogClient) sortConvertPosts(list []*pb.Content) ([]service.BlogPost, error) {
	sort.Sort(sortableContents(list))

	size := len(list)
	// no duplicate check, there is one in GetProfiles
	userIds := make([]uint64, 0, size)
	for _, content := range list {
		userIds = append(userIds, content.UserId)
	}

	users, err := client.profileService.GetProfiles(userIds)
	if err != nil {
		return nil, err
	}

	contents := make([]service.BlogPost, 0, size)
	for _, content := range list {
		contents = append(contents, convertPost(content, users[content.UserId], client.dateFormat))
	}
	return contents, nil
}

func convertPost(post *pb.Content, creator profileservice.UserProfile, dateFormat string) service.BlogPost {
	createdAt := time.Unix(post.CreatedAt, 0)
	return service.BlogPost{
		PostId: post.PostId, Creator: creator, Date: createdAt.Format(dateFormat),
		Title: post.Title, Content: template.HTML(post.Text),
	}
}
