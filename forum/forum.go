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
	"fmt"
	"strconv"
	"strings"

	"github.com/dvaumoron/puzzleweb"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const threadIdName = "threadId"

const parsingThreadIdErrorMsg = "Failed to parse threadId"

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
	logger := forumConfig.Logger
	forumService := forumConfig.Service
	defaultPageSize := forumConfig.PageSize

	listTmpl := "forum/list.html"
	createTmpl := "forum/create.html"
	viewTmpl := "forum/view.html"
	args := forumConfig.Args
	switch len(args) {
	default:
		logger.Info("MakeForumPage should be called with 3 to 6 arguments")
		fallthrough
	case 3:
		if args[2] != "" {
			viewTmpl = args[2]
		}
		fallthrough
	case 2:
		if args[1] != "" {
			createTmpl = args[1]
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
			userId := puzzleweb.GetSessionUserId(c)

			pageNumber, start, end, filter := common.GetPagination(c, defaultPageSize)

			total, threads, err := forumService.GetThreads(userId, start, end, filter)
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			common.InitPagination(data, filter, pageNumber, end, total)
			data["Threads"] = threads
			data[common.AllowedToCreateName] = forumService.CreateThreadRight(userId)
			data[common.AllowedToDeleteName] = forumService.DeleteRight(userId)
			puzzleweb.InitNoELementMsg(data, len(threads), c)
			return listTmpl, ""
		}),
		createThreadHandler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			// nothing to init
			return createTmpl, ""
		}),
		saveThreadHandler: common.CreateRedirect(func(c *gin.Context) string {
			title := c.PostForm("title")
			message := c.PostForm("message")

			err := forumService.CreateThread(puzzleweb.GetSessionUserId(c), title, message)

			var targetBuilder strings.Builder
			targetBuilder.WriteString(common.GetBaseUrl(1, c))
			if err != nil {
				common.WriteError(&targetBuilder, err.Error())
			}
			return targetBuilder.String()
		}),
		deleteThreadHandler: common.CreateRedirect(func(c *gin.Context) string {
			threadId, err := strconv.ParseUint(c.Param(threadIdName), 10, 64)
			if err == nil {
				err = forumService.DeleteThread(puzzleweb.GetSessionUserId(c), threadId)
			} else {
				logger.Warn(parsingThreadIdErrorMsg, zap.Error(err))
				err = common.ErrTechnical
			}

			var targetBuilder strings.Builder
			targetBuilder.WriteString(common.GetBaseUrl(2, c))
			if err != nil {
				common.WriteError(&targetBuilder, err.Error())
			}
			return targetBuilder.String()
		}),
		viewThreadHandler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			threadId, err := strconv.ParseUint(c.Param(threadIdName), 10, 64)
			if err != nil {
				logger.Warn(parsingThreadIdErrorMsg, zap.Error(err))
				return "", common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}

			pageNumber, start, end, filter := common.GetPagination(c, defaultPageSize)

			userId := puzzleweb.GetSessionUserId(c)
			total, thread, messages, err := forumService.GetThread(userId, threadId, start, end, filter)
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			common.InitPagination(data, filter, pageNumber, end, total)
			data["Thread"] = thread
			data["Messages"] = messages
			data[common.AllowedToCreateName] = forumService.CreateMessageRight(userId)
			data[common.AllowedToDeleteName] = forumService.DeleteRight(userId)
			puzzleweb.InitNoELementMsg(data, len(messages), c)
			return viewTmpl, ""
		}),
		saveMessageHandler: common.CreateRedirect(func(c *gin.Context) string {
			threadId, err := strconv.ParseUint(c.Param(threadIdName), 10, 64)
			if err != nil {
				logger.Warn(parsingThreadIdErrorMsg, zap.Error(err))
				return common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}
			message := c.PostForm("message")

			err = forumService.CreateMessage(puzzleweb.GetSessionUserId(c), threadId, message)

			targetBuilder := threadUrlBuilder(common.GetBaseUrl(3, c), threadId)
			if err != nil {
				common.WriteError(targetBuilder, err.Error())
			}
			return targetBuilder.String()
		}),
		deleteMessageHandler: common.CreateRedirect(func(c *gin.Context) string {
			threadId, err := strconv.ParseUint(c.Param(threadIdName), 10, 64)
			if err != nil {
				logger.Warn(parsingThreadIdErrorMsg, zap.Error(err))
				return common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}
			messageId, err := strconv.ParseUint(c.Param("messageId"), 10, 64)
			if err != nil {
				logger.Warn("Failed to parse messageId", zap.Error(err))
				return common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}

			err = forumService.DeleteMessage(puzzleweb.GetSessionUserId(c), threadId, messageId)

			targetBuilder := threadUrlBuilder(common.GetBaseUrl(4, c), threadId)
			if err != nil {
				common.WriteError(targetBuilder, err.Error())
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
	targetBuilder.WriteString(fmt.Sprint(threadId))
	return targetBuilder
}
