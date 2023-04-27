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

import (
	profileservice "github.com/dvaumoron/puzzleweb/profile/service"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
)

type ForumContent struct {
	Id      uint64
	Creator profileservice.UserProfile
	Date    string
	Text    string
}

type ForumService interface {
	CreateThread(logger otelzap.LoggerWithCtx, userId uint64, title string, message string) (uint64, error)
	CreateMessage(logger otelzap.LoggerWithCtx, userId uint64, threadId uint64, message string) error
	GetThread(logger otelzap.LoggerWithCtx, userId uint64, threadId uint64, start uint64, end uint64, filter string) (uint64, ForumContent, []ForumContent, error)
	GetThreads(logger otelzap.LoggerWithCtx, userId uint64, start uint64, end uint64, filter string) (uint64, []ForumContent, error)
	DeleteThread(logger otelzap.LoggerWithCtx, userId uint64, threadId uint64) error
	DeleteMessage(logger otelzap.LoggerWithCtx, userId uint64, threadId uint64, messageId uint64) error
	CreateThreadRight(logger otelzap.LoggerWithCtx, userId uint64) bool
	CreateMessageRight(logger otelzap.LoggerWithCtx, userId uint64) bool
	DeleteRight(logger otelzap.LoggerWithCtx, userId uint64) bool
}

type CommentService interface {
	CreateCommentThread(logger otelzap.LoggerWithCtx, userId uint64, elemTitle string) error
	CreateComment(logger otelzap.LoggerWithCtx, userId uint64, elemTitle string, message string) error
	GetCommentThread(logger otelzap.LoggerWithCtx, userId uint64, elemTitle string, start uint64, end uint64) (uint64, []ForumContent, error)
	DeleteCommentThread(logger otelzap.LoggerWithCtx, userId uint64, elemTitle string) error
	DeleteComment(logger otelzap.LoggerWithCtx, userId uint64, elemTitle string, commentId uint64) error
	CreateMessageRight(logger otelzap.LoggerWithCtx, userId uint64) bool
	DeleteRight(logger otelzap.LoggerWithCtx, userId uint64) bool
}

type FullForumService interface {
	ForumService
	CommentService
}
