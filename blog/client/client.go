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
	"html/template"
	"sort"
	"time"

	pb "github.com/dvaumoron/puzzleblogservice"
	rightclient "github.com/dvaumoron/puzzleweb/admin/client"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	profileclient "github.com/dvaumoron/puzzleweb/profile/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type BlogPost struct {
	PostId  uint64
	Creator profileclient.Profile
	Date    string
	Title   string
	content template.HTML // markdown apply is done before storage
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

func CreatePost(blogId uint64, groupId uint64, userId uint64, title string, content string) error {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionCreate)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(config.BlogServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewBlogClient(conn).CreatePost(ctx, &pb.CreateRequest{
		BlogId: blogId, UserId: userId, Title: title, Text: content,
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

func GetPost(blogId uint64, groupId uint64, userId uint64, postId uint64) (*BlogPost, error) {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionAccess)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.Dial(config.BlogServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewBlogClient(conn).GetPost(ctx, &pb.IdRequest{
		BlogId: blogId, PostId: postId,
	})
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrTechnical
	}

	creatorId := response.UserId
	users, err := profileclient.GetProfiles([]uint64{creatorId})
	if err != nil {
		return nil, err
	}
	return convertPost(response, users[creatorId]), nil
}

func GetPosts(blogId uint64, groupId uint64, userId uint64, start uint64, end uint64, filter string) ([]*BlogPost, error) {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionAccess)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.Dial(config.BlogServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewBlogClient(conn).GetPosts(ctx, &pb.SearchRequest{
		BlogId: blogId, Start: start, End: end, Filter: filter,
	})
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrTechnical
	}
	list := response.List
	if len(list) == 0 {
		return nil, nil
	}
	return sortConvertPosts(list)
}

func DeletePost(blogId uint64, groupId uint64, userId uint64, postId uint64) error {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionAccess)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(config.BlogServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewBlogClient(conn).DeletePost(ctx, &pb.IdRequest{
		BlogId: blogId, PostId: postId,
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

func sortConvertPosts(list []*pb.Content) ([]*BlogPost, error) {
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

	contents := make([]*BlogPost, 0, size)
	for _, content := range list {
		contents = append(contents, convertPost(content, users[content.UserId]))
	}
	return contents, nil
}

func convertPost(post *pb.Content, creator profileclient.Profile) *BlogPost {
	createdAt := time.Unix(post.CreatedAt, 0)
	return &BlogPost{
		PostId: post.PostId, Creator: creator, Date: createdAt.Format(config.DateFormat),
		Title: post.Title, content: template.HTML(post.Text),
	}
}
