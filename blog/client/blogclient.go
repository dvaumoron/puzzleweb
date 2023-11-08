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
	"cmp"
	"context"
	"slices"
	"time"

	pb "github.com/dvaumoron/puzzleblogservice"
	grpcclient "github.com/dvaumoron/puzzlegrpcclient"
	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	blogservice "github.com/dvaumoron/puzzleweb/blog/service"
	"github.com/dvaumoron/puzzleweb/common"
	profileservice "github.com/dvaumoron/puzzleweb/profile/service"
	"google.golang.org/grpc"
)

type blogClient struct {
	grpcclient.Client
	blogId         uint64
	groupId        uint64
	dateFormat     string
	authService    adminservice.AuthService
	profileService profileservice.ProfileService
}

func New(serviceAddr string, dialOptions []grpc.DialOption, blogId uint64, groupId uint64, dateFormat string, authService adminservice.AuthService, profileService profileservice.ProfileService) blogservice.BlogService {
	return blogClient{
		Client: grpcclient.Make(serviceAddr, dialOptions...), blogId: blogId, groupId: groupId,
		dateFormat: dateFormat, authService: authService, profileService: profileService,
	}
}

func cmpDesc(a *pb.Content, b *pb.Content) int {
	return cmp.Compare(b.CreatedAt, a.CreatedAt)
}

func (client blogClient) CreatePost(ctx context.Context, userId uint64, title string, content string) (uint64, error) {
	err := client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionCreate)
	if err != nil {
		return 0, err
	}

	conn, err := client.Dial()
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	response, err := pb.NewBlogClient(conn).CreatePost(ctx, &pb.CreateRequest{
		BlogId: client.blogId, UserId: userId, Title: title, Text: content,
	})
	if err != nil {
		return 0, err
	}
	if !response.Success {
		return 0, common.ErrUpdate
	}
	return response.Id, nil
}

func (client blogClient) GetPost(ctx context.Context, userId uint64, postId uint64) (blogservice.BlogPost, error) {
	err := client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionAccess)
	if err != nil {
		return blogservice.BlogPost{}, err
	}

	conn, err := client.Dial()
	if err != nil {
		return blogservice.BlogPost{}, err
	}
	defer conn.Close()

	response, err := pb.NewBlogClient(conn).GetPost(ctx, &pb.IdRequest{
		BlogId: client.blogId, PostId: postId,
	})
	if err != nil {
		return blogservice.BlogPost{}, err
	}

	creatorId := response.UserId
	users, err := client.profileService.GetProfiles(ctx, []uint64{creatorId})
	if err != nil {
		return blogservice.BlogPost{}, err
	}
	return convertPost(response, users[creatorId], client.dateFormat), nil
}

func (client blogClient) GetPosts(ctx context.Context, userId uint64, start uint64, end uint64, filter string) (uint64, []blogservice.BlogPost, error) {
	err := client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionAccess)
	if err != nil {
		return 0, nil, err
	}

	conn, err := client.Dial()
	if err != nil {
		return 0, nil, err
	}
	defer conn.Close()

	response, err := pb.NewBlogClient(conn).GetPosts(ctx, &pb.SearchRequest{
		BlogId: client.blogId, Start: start, End: end, Filter: filter,
	})
	if err != nil {
		return 0, nil, err
	}

	total := response.Total
	list := response.List
	if len(list) == 0 {
		return total, nil, nil
	}

	posts, err := client.sortConvertPosts(ctx, list)
	if err != nil {
		return 0, nil, err
	}
	return total, posts, nil
}

func (client blogClient) DeletePost(ctx context.Context, userId uint64, postId uint64) error {
	err := client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionDelete)
	if err != nil {
		return err
	}

	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	response, err := pb.NewBlogClient(conn).DeletePost(ctx, &pb.IdRequest{
		BlogId: client.blogId, PostId: postId,
	})
	if err != nil {
		return err
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client blogClient) CreateRight(ctx context.Context, userId uint64) bool {
	return client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionCreate) == nil
}

func (client blogClient) DeleteRight(ctx context.Context, userId uint64) bool {
	return client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionDelete) == nil
}

func (client blogClient) sortConvertPosts(ctx context.Context, list []*pb.Content) ([]blogservice.BlogPost, error) {
	slices.SortFunc(list, cmpDesc)

	size := len(list)
	// no duplicate check, there is one in GetProfiles
	userIds := make([]uint64, 0, size)
	for _, content := range list {
		userIds = append(userIds, content.UserId)
	}

	users, err := client.profileService.GetProfiles(ctx, userIds)
	if err != nil {
		return nil, err
	}

	contents := make([]blogservice.BlogPost, 0, size)
	for _, content := range list {
		contents = append(contents, convertPost(content, users[content.UserId], client.dateFormat))
	}
	return contents, nil
}

func convertPost(post *pb.Content, creator profileservice.UserProfile, dateFormat string) blogservice.BlogPost {
	createdAt := time.Unix(post.CreatedAt, 0)
	return blogservice.BlogPost{
		PostId: post.PostId, Creator: creator, Date: createdAt.Format(dateFormat), Title: post.Title, Content: post.Text,
	}
}
