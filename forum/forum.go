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

package forum

import (
	"errors"
	"strconv"
	"strings"

	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/common/config"
	puzzleweb "github.com/dvaumoron/puzzleweb/core"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const emptyMessage = "EmptyForumMessage"

const threadIdName = "threadId"

const parsingThreadIdErrorMsg = "Failed to parse threadId"

var errEmptyMessage = errors.New(emptyMessage)

// TODO preview && markdown ?
type forumWidget struct {
	listThreadHandler    gin.HandlerFunc
	createThreadHandler  gin.HandlerFunc
	saveThreadHandler    gin.HandlerFunc
	deleteThreadHandler  gin.HandlerFunc
	viewThreadHandler    gin.HandlerFunc
	saveMessageHandler   gin.HandlerFunc
	deleteMessageHandler gin.HandlerFunc
}

func (w forumWidget) LoadInto(router gin.IRouter) {
	router.GET("/", w.listThreadHandler)
	router.GET("/create", w.createThreadHandler)
	router.POST("/save", w.saveThreadHandler)
	router.GET("/delete/:threadId", w.deleteThreadHandler)
	router.GET("/view/:threadId", w.viewThreadHandler)
	router.POST("/message/save/:threadId", w.saveMessageHandler)
	router.GET("/message/delete/:threadId/:messageId", w.deleteMessageHandler)
}

func MakeForumPage(forumName string, forumConfig config.ForumConfig) puzzleweb.Page {
	forumService := forumConfig.Service
	defaultPageSize := forumConfig.PageSize

	listTmpl := "forum/list"
	viewTmpl := "forum/view"
	createTmpl := "forum/create"
	switch args := forumConfig.Args; len(args) {
	default:
		forumConfig.Logger.Info("MakeForumPage should be called with 0 to 3 optional arguments")
		fallthrough
	case 3:
		if args[2] != "" {
			createTmpl = args[2]
		}
		fallthrough
	case 2:
		if args[1] != "" {
			viewTmpl = args[1]
		}
		fallthrough
	case 1:
		if args[0] != "" {
			listTmpl = args[0]
		}
		fallthrough
	case 0:
	}

	p := puzzleweb.MakePage(forumName)
	p.Widget = forumWidget{
		listThreadHandler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			ctx := c.Request.Context()
			userId, _ := data[common.UserIdName].(uint64)

			pageNumber, start, end, filter := common.GetPagination(defaultPageSize, c)

			total, threads, err := forumService.GetThreads(ctx, userId, start, end, filter)
			if err != nil {
				return "", common.DefaultErrorRedirect(puzzleweb.GetLogger(c), err.Error())
			}

			common.InitPagination(data, filter, pageNumber, end, total)
			data["Threads"] = threads
			data[common.AllowedToCreateName] = forumService.CreateThreadRight(ctx, userId)
			data[common.AllowedToDeleteName] = forumService.DeleteRight(ctx, userId)
			puzzleweb.InitNoELementMsg(data, len(threads), c)
			return listTmpl, ""
		}),
		createThreadHandler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			data[common.BaseUrlName] = common.GetBaseUrl(1, c)
			return createTmpl, ""
		}),
		saveThreadHandler: common.CreateRedirect(func(c *gin.Context) string {
			logger := puzzleweb.GetLogger(c)
			title := c.PostForm("title")
			message := c.PostForm("message")

			if title == "" {
				return common.DefaultErrorRedirect(logger, "EmptyThreadTitle")
			}
			if message == "" {
				return common.DefaultErrorRedirect(logger, emptyMessage)
			}

			threadId, err := forumService.CreateThread(c.Request.Context(), puzzleweb.GetSessionUserId(c), title, message)
			if err != nil {
				return common.DefaultErrorRedirect(logger, err.Error())
			}
			return threadUrlBuilder(common.GetBaseUrl(1, c), threadId).String()
		}),
		deleteThreadHandler: common.CreateRedirect(func(c *gin.Context) string {
			logger := puzzleweb.GetLogger(c)
			threadId, err := strconv.ParseUint(c.Param(threadIdName), 10, 64)
			if err == nil {
				err = forumService.DeleteThread(c.Request.Context(), puzzleweb.GetSessionUserId(c), threadId)
			} else {
				logger.Warn(parsingThreadIdErrorMsg, zap.Error(err))
				err = common.ErrTechnical
			}

			var targetBuilder strings.Builder
			targetBuilder.WriteString(common.GetBaseUrl(2, c))
			if err != nil {
				common.WriteError(&targetBuilder, logger, err.Error())
			}
			return targetBuilder.String()
		}),
		viewThreadHandler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			logger := puzzleweb.GetLogger(c)
			threadId, err := strconv.ParseUint(c.Param(threadIdName), 10, 64)
			if err != nil {
				logger.Warn(parsingThreadIdErrorMsg, zap.Error(err))
				return "", common.DefaultErrorRedirect(logger, common.ErrorTechnicalKey)
			}

			pageNumber, start, end, filter := common.GetPagination(defaultPageSize, c)

			ctx := c.Request.Context()
			userId, _ := data[common.UserIdName].(uint64)
			total, thread, messages, err := forumService.GetThread(ctx, userId, threadId, start, end, filter)
			if err != nil {
				return "", common.DefaultErrorRedirect(logger, err.Error())
			}

			common.InitPagination(data, filter, pageNumber, end, total)
			data[common.BaseUrlName] = common.GetBaseUrl(2, c)
			data["Thread"] = thread
			data["ForumMessages"] = messages
			data[common.AllowedToCreateName] = forumService.CreateMessageRight(ctx, userId)
			data[common.AllowedToDeleteName] = forumService.DeleteRight(ctx, userId)
			puzzleweb.InitNoELementMsg(data, len(messages), c)
			return viewTmpl, ""
		}),
		saveMessageHandler: common.CreateRedirect(func(c *gin.Context) string {
			logger := puzzleweb.GetLogger(c)
			threadId, err := strconv.ParseUint(c.Param(threadIdName), 10, 64)
			if err != nil {
				logger.Warn(parsingThreadIdErrorMsg, zap.Error(err))
				return common.DefaultErrorRedirect(logger, common.ErrorTechnicalKey)
			}
			message := c.PostForm("message")

			err = errEmptyMessage
			if message != "" {
				err = forumService.CreateMessage(c.Request.Context(), puzzleweb.GetSessionUserId(c), threadId, message)
			}

			targetBuilder := threadUrlBuilder(common.GetBaseUrl(3, c), threadId)
			if err != nil {
				common.WriteError(targetBuilder, logger, err.Error())
			}
			return targetBuilder.String()
		}),
		deleteMessageHandler: common.CreateRedirect(func(c *gin.Context) string {
			logger := puzzleweb.GetLogger(c)
			threadId, err := strconv.ParseUint(c.Param(threadIdName), 10, 64)
			if err != nil {
				logger.Warn(parsingThreadIdErrorMsg, zap.Error(err))
				return common.DefaultErrorRedirect(logger, common.ErrorTechnicalKey)
			}
			messageId, err := strconv.ParseUint(c.Param("messageId"), 10, 64)
			if err != nil {
				logger.Warn("Failed to parse messageId", zap.Error(err))
				return common.DefaultErrorRedirect(logger, common.ErrorTechnicalKey)
			}

			err = forumService.DeleteMessage(c.Request.Context(), puzzleweb.GetSessionUserId(c), threadId, messageId)

			targetBuilder := threadUrlBuilder(common.GetBaseUrl(4, c), threadId)
			if err != nil {
				common.WriteError(targetBuilder, logger, err.Error())
			}
			return targetBuilder.String()
		}),
	}
	return p
}

func threadUrlBuilder(base string, threadId uint64) *strings.Builder {
	targetBuilder := new(strings.Builder)
	targetBuilder.WriteString(base)
	targetBuilder.WriteString("view/")
	targetBuilder.WriteString(strconv.FormatUint(threadId, 10))
	return targetBuilder
}
