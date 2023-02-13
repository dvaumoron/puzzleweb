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
package blog

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

const postIdName = "postId"

const parsingPostIdErrorMsg = "Failed to parse postId"

type blogWidget struct {
	listHandler          gin.HandlerFunc
	viewHandler          gin.HandlerFunc
	saveCommentHandler   gin.HandlerFunc
	deleteCommentHandler gin.HandlerFunc
	createHandler        gin.HandlerFunc
	previewHandler       gin.HandlerFunc
	saveHandler          gin.HandlerFunc
	deleteHandler        gin.HandlerFunc
}

func (w blogWidget) LoadInto(router gin.IRouter) {
	router.GET("/", w.listHandler)
	router.GET("/view/:postId", w.viewHandler)
	router.POST("/comment/save/:postId", w.saveCommentHandler)
	router.POST("/comment/delete/:postId/:commentId", w.deleteCommentHandler)
	router.GET("/create", w.createHandler)
	router.POST("/preview", w.previewHandler)
	router.POST("/save", w.saveHandler)
	router.GET("/delete/:postId", w.deleteHandler)
}

func MakeBlogPage(blogName string, blogConfig config.BlogConfig) puzzleweb.Page {
	logger := blogConfig.Logger
	blogService := blogConfig.Service
	commentService := blogConfig.CommentService
	defaultPageSize := blogConfig.PageSize

	listTmpl := "blog/list.html"
	viewTmpl := "blog/view.html"
	createTmpl := "blog/create.html"
	previewTmpl := "blog/preview.html"

	// TODO
	p := puzzleweb.MakePage(blogName)
	p.Widget = blogWidget{
		listHandler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			userId := puzzleweb.GetSessionUserId(c)

			pageNumber, start, end, filter := common.GetPagination(c, defaultPageSize)

			total, posts, err := blogService.GetPosts(userId, start, end, filter)
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			common.InitPagination(data, filter, pageNumber, end, total)
			data["Posts"] = posts
			data[common.AllowedToCreateName] = blogService.CreateRight(userId)
			data[common.AllowedToDeleteName] = blogService.DeleteRight(userId)
			puzzleweb.InitNoELementMsg(data, len(posts), c)
			return listTmpl, ""
		}),
		viewHandler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			userId := puzzleweb.GetSessionUserId(c)

			pageNumber, start, end, _ := common.GetPagination(c, defaultPageSize)

			postId, err := strconv.ParseUint(c.Param(postIdName), 10, 64)
			if err != nil {
				logger.Warn(parsingPostIdErrorMsg, zap.Error(err))
				return "", common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}

			post, err := blogService.GetPost(userId, postId)
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			total, comments, err := commentService.GetCommentThread(userId, post.Title, start, end)
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			common.InitPagination(data, "", pageNumber, end, total)
			data["Post"] = post
			data["Comments"] = comments
			data[common.AllowedToCreateName] = commentService.CreateMessageRight(userId)
			data[common.AllowedToDeleteName] = commentService.DeleteRight(userId)
			if len(comments) == 0 {
				data["CommentMsg"] = puzzleweb.GetMessages(c)["NoComment"]
			}
			return viewTmpl, ""
		}),
		saveCommentHandler: common.CreateRedirect(func(c *gin.Context) string {
			userId := puzzleweb.GetSessionUserId(c)

			postId, err := strconv.ParseUint(c.Param(postIdName), 10, 64)
			if err != nil {
				logger.Warn(parsingPostIdErrorMsg, zap.Error(err))
				return common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}
			comment := c.PostForm("comment")

			post, err := blogService.GetPost(userId, postId)
			if err != nil {
				return common.DefaultErrorRedirect(err.Error())
			}

			err = commentService.CreateComment(userId, post.Title, comment)
			if err != nil {
				return common.DefaultErrorRedirect(err.Error())
			}

			var targetBuilder strings.Builder
			targetBuilder.WriteString(common.GetBaseUrl(3, c))
			targetBuilder.WriteString("view/")
			targetBuilder.WriteString(fmt.Sprint(postId))
			if err != nil {
				common.WriteError(&targetBuilder, err.Error())
			}
			return targetBuilder.String()
		}),
	}
	return p
}
