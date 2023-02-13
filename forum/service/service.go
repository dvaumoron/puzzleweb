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
package service

import profileservice "github.com/dvaumoron/puzzleweb/profile/service"

type ForumContent struct {
	Id      uint64
	Creator profileservice.UserProfile
	Date    string
	Text    string
}

type ForumService interface {
	CreateThread(userId uint64, title string, message string) (uint64, error)
	CreateMessage(userId uint64, threadId uint64, message string) error
	GetThread(userId uint64, threadId uint64, start uint64, end uint64, filter string) (uint64, ForumContent, []ForumContent, error)
	GetThreads(userId uint64, start uint64, end uint64, filter string) (uint64, []ForumContent, error)
	DeleteThread(userId uint64, threadId uint64) error
	DeleteMessage(userId uint64, threadId uint64, messageId uint64) error
	CreateThreadRight(userId uint64) bool
	CreateMessageRight(userId uint64) bool
	DeleteRight(userId uint64) bool
}

type CommentService interface {
	CreateCommentThread(userId uint64, elemTitle string) error
	CreateComment(userId uint64, elemTitle string, message string) error
	GetCommentThread(userId uint64, elemTitle string, start uint64, end uint64) (uint64, []ForumContent, error)
	DeleteCommentThread(userId uint64, elemTitle string) error
	DeleteComment(userId uint64, elemTitle string, commentId uint64) error
	CreateMessageRight(userId uint64) bool
	DeleteRight(userId uint64) bool
}

type FullForumService interface {
	ForumService
	CommentService
}
