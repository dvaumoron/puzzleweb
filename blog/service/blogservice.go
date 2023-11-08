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

package blogservice

import (
	"context"

	profileservice "github.com/dvaumoron/puzzleweb/profile/service"
)

type BlogPost struct {
	PostId  uint64
	Creator profileservice.UserProfile
	Date    string
	Title   string
	Content string
}

type BlogService interface {
	CreatePost(ctx context.Context, userId uint64, title string, content string) (uint64, error)
	GetPost(ctx context.Context, userId uint64, postId uint64) (BlogPost, error)
	GetPosts(ctx context.Context, userId uint64, start uint64, end uint64, filter string) (uint64, []BlogPost, error)
	DeletePost(ctx context.Context, userId uint64, postId uint64) error
	CreateRight(ctx context.Context, userId uint64) bool
	DeleteRight(ctx context.Context, userId uint64) bool
}
