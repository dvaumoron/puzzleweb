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
	rightclient "github.com/dvaumoron/puzzleweb/admin/client"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/forum/client"
	"github.com/dvaumoron/puzzleweb/log"
	"github.com/dvaumoron/puzzleweb/session"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const threadIdName = "threadId"

const parsingThreadIdErrorMsg = "Failed to parse threadId."

type forumWidget struct {
	listThreadHandler    gin.HandlerFunc
	createThreadHandler  gin.HandlerFunc
	saveThreadHandler    gin.HandlerFunc
	deleteThreadHandler  gin.HandlerFunc
	viewThreadHandler    gin.HandlerFunc
	saveMessageHandler   gin.HandlerFunc
	deleteMessageHandler gin.HandlerFunc
}

func (w *forumWidget) LoadInto(router gin.IRouter) {
	router.GET("/", w.listThreadHandler)
	router.GET("/create", w.createThreadHandler)
	router.POST("/save", w.saveThreadHandler)
	router.GET("/delete/:threadId", w.deleteThreadHandler)
	router.GET("/view/:threadId", w.viewThreadHandler)
	router.POST("/message/save/:threadId", w.saveMessageHandler)
	router.GET("/message/delete/:threadId/:messageId", w.deleteMessageHandler)
}

func NewForumPage(forumName string, groupId uint64, forumId uint64, args ...string) *puzzleweb.Page {
	config.Shared.LoadForum()

	listTmpl := "forum/list.html"
	createTmpl := "forum/create.html"
	viewTmpl := "forum/view.html"
	switch len(args) {
	default:
		log.Logger.Info("NewForumPage should be called with 3 to 6 arguments.")
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

	// TODO
	p := puzzleweb.NewPage(forumName)
	p.Widget = &forumWidget{
		listThreadHandler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			userId := session.GetUserId(c)

			pageNumber, start, end, filter := common.GetPagination(c)

			total, threads, err := client.GetThreads(forumId, groupId, userId, start, end, filter)
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			createRight := rightclient.AuthQuery(userId, groupId, rightclient.ActionCreate) == nil

			common.InitPagination(data, filter, pageNumber, end, total)
			data["Threads"] = threads
			data[common.AllowedToCreateName] = createRight
			common.InitNoELementMsg(data, len(threads), c)
			return listTmpl, ""
		}),
		createThreadHandler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			if err := rightclient.AuthQuery(session.GetUserId(c), groupId, rightclient.ActionCreate); err == nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}
			return createTmpl, ""
		}),
		saveThreadHandler: common.CreateRedirect(func(c *gin.Context) string {
			title := c.PostForm("title")
			message := c.PostForm("message")

			err := client.CreateThread(forumId, groupId, session.GetUserId(c), title, message)

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
				err = client.DeleteThread(forumId, groupId, session.GetUserId(c), threadId)
			} else {
				log.Logger.Warn("Failed to parse threadId.", zap.Error(err))
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
				log.Logger.Warn(parsingThreadIdErrorMsg, zap.Error(err))
				return "", common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}

			pageNumber, start, end, filter := common.GetPagination(c)

			userId := session.GetUserId(c)
			total, thread, messages, err := client.GetThread(forumId, groupId, userId, threadId, start, end, filter)
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			createRight := rightclient.AuthQuery(userId, groupId, rightclient.ActionUpdate) == nil

			common.InitPagination(data, filter, pageNumber, end, total)
			data["Thread"] = thread
			data["Messages"] = messages
			data[common.AllowedToCreateName] = createRight
			common.InitNoELementMsg(data, len(messages), c)
			return viewTmpl, ""
		}),
		saveMessageHandler: common.CreateRedirect(func(c *gin.Context) string {
			threadId, err := strconv.ParseUint(c.Param(threadIdName), 10, 64)
			if err != nil {
				log.Logger.Warn(parsingThreadIdErrorMsg, zap.Error(err))
				return common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}
			message := c.PostForm("message")

			err = client.CreateMessage(groupId, session.GetUserId(c), threadId, message)

			targetBuilder := threadUrlBuilder(common.GetBaseUrl(3, c), threadId)
			if err != nil {
				common.WriteError(targetBuilder, err.Error())
			}
			return targetBuilder.String()
		}),
		deleteMessageHandler: common.CreateRedirect(func(c *gin.Context) string {
			threadId, err := strconv.ParseUint(c.Param(threadIdName), 10, 64)
			if err != nil {
				log.Logger.Warn(parsingThreadIdErrorMsg, zap.Error(err))
				return common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}
			messageId, err := strconv.ParseUint(c.Param("messageId"), 10, 64)
			if err != nil {
				log.Logger.Warn("Failed to parse messageId.", zap.Error(err))
				return common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}

			err = client.DeleteMessage(groupId, session.GetUserId(c), threadId, messageId)

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
