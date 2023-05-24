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
	"errors"
	"html/template"
	"strconv"
	"strings"

	"github.com/dvaumoron/puzzleweb"
	"github.com/dvaumoron/puzzleweb/blog/service"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const emptyTitle = "EmptyPostTitle"
const emptyContent = "EmptyPostContent"

const postIdName = "postId"

const parsingPostIdErrorMsg = "Failed to parse postId"

var errEmptyComment = errors.New("EmptyComment")

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
	router.GET("/comment/delete/:postId/:commentId", w.deleteCommentHandler)
	router.GET("/create", w.createHandler)
	router.POST("/preview", w.previewHandler)
	router.POST("/save", w.saveHandler)
	router.GET("/delete/:postId", w.deleteHandler)
}

func MakeBlogPage(blogName string, blogConfig config.BlogConfig) puzzleweb.Page {
	tracer := blogConfig.Tracer
	blogService := blogConfig.Service
	commentService := blogConfig.CommentService
	markdownService := blogConfig.MarkdownService
	defaultPageSize := blogConfig.PageSize
	extractSize := blogConfig.ExtractSize

	listTmpl := "blog/list"
	viewTmpl := "blog/view"
	createTmpl := "blog/create"
	previewTmpl := "blog/preview"
	switch args := blogConfig.Args; len(args) {
	default:
		blogConfig.Logger.Info("MakeBlogPage should be called with 0 to 4 optional arguments.")
		fallthrough
	case 4:
		if args[3] != "" {
			previewTmpl = args[3]
		}
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
	case 0:
	}

	p := puzzleweb.MakePage(blogName)
	p.Widget = blogWidget{
		listHandler: puzzleweb.CreateTemplate(tracer, "blogWidget/listHandler", func(data gin.H, c *gin.Context) (string, string) {
			logger := puzzleweb.GetLogger(c)
			userId, _ := data[common.IdName].(uint64)

			pageNumber, start, end, filter := common.GetPagination(defaultPageSize, c)

			total, posts, err := blogService.GetPosts(logger, userId, start, end, filter)
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			filterPostsExtract(posts, extractSize)

			common.InitPagination(data, filter, pageNumber, end, total)
			data["Posts"] = posts
			data[common.AllowedToCreateName] = blogService.CreateRight(logger, userId)
			data[common.AllowedToDeleteName] = blogService.DeleteRight(logger, userId)
			puzzleweb.InitNoELementMsg(data, len(posts), c)
			return listTmpl, ""
		}),
		viewHandler: puzzleweb.CreateTemplate(tracer, "blogWidget/viewHandler", func(data gin.H, c *gin.Context) (string, string) {
			logger := puzzleweb.GetLogger(c)
			userId, _ := data[common.IdName].(uint64)

			pageNumber, start, end, _ := common.GetPagination(defaultPageSize, c)

			postId, err := strconv.ParseUint(c.Param(postIdName), 10, 64)
			if err != nil {
				logger.Warn(parsingPostIdErrorMsg, zap.Error(err))
				return "", common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}

			post, err := blogService.GetPost(logger, userId, postId)
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			total, comments, err := commentService.GetCommentThread(logger, userId, post.Title, start, end)
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			common.InitPagination(data, "", pageNumber, end, total)
			data[common.BaseUrlName] = common.GetBaseUrl(2, c)
			data["Post"] = post
			data["Comments"] = comments
			data[common.AllowedToCreateName] = commentService.CreateMessageRight(logger, userId)
			data[common.AllowedToDeleteName] = commentService.DeleteRight(logger, userId)
			if len(comments) == 0 {
				data["CommentMsg"] = puzzleweb.GetMessages(c)["NoComment"]
			}
			return viewTmpl, ""
		}),
		saveCommentHandler: common.CreateRedirect(tracer, "blogWidget/saveCommentHandler", func(c *gin.Context) string {
			logger := puzzleweb.GetLogger(c)
			userId := puzzleweb.GetSessionUserId(logger, c)

			postId, err := strconv.ParseUint(c.Param(postIdName), 10, 64)
			if err != nil {
				logger.Warn(parsingPostIdErrorMsg, zap.Error(err))
				return common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}
			comment := c.PostForm("comment")

			err = errEmptyComment
			if comment != "" {
				var post service.BlogPost
				post, err = blogService.GetPost(logger, userId, postId)
				if err != nil {
					return common.DefaultErrorRedirect(err.Error())
				}

				err = commentService.CreateComment(logger, userId, post.Title, comment)
				if err != nil {
					return common.DefaultErrorRedirect(err.Error())
				}
			}

			targetBuilder := postUrlBuilder(common.GetBaseUrl(3, c), postId)
			if err != nil {
				common.WriteError(targetBuilder, err.Error())
			}
			return targetBuilder.String()
		}),
		deleteCommentHandler: common.CreateRedirect(tracer, "blogWidget/deleteCommentHandler", func(c *gin.Context) string {
			logger := puzzleweb.GetLogger(c)
			userId := puzzleweb.GetSessionUserId(logger, c)

			postId, err := strconv.ParseUint(c.Param(postIdName), 10, 64)
			if err != nil {
				logger.Warn(parsingPostIdErrorMsg, zap.Error(err))
				return common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}
			commentId, err := strconv.ParseUint(c.Param("commentId"), 10, 64)
			if err != nil {
				logger.Warn("Failed to parse commentId", zap.Error(err))
				return common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}

			post, err := blogService.GetPost(logger, userId, postId)
			if err != nil {
				return common.DefaultErrorRedirect(err.Error())
			}

			err = commentService.DeleteComment(logger, userId, post.Title, commentId)
			if err != nil {
				return common.DefaultErrorRedirect(err.Error())
			}

			targetBuilder := postUrlBuilder(common.GetBaseUrl(4, c), postId)
			if err != nil {
				common.WriteError(targetBuilder, err.Error())
			}
			return targetBuilder.String()
		}),
		createHandler: puzzleweb.CreateTemplate(tracer, "blogWidget/createHandler", func(data gin.H, c *gin.Context) (string, string) {
			data[common.BaseUrlName] = common.GetBaseUrl(1, c)
			return createTmpl, ""
		}),
		previewHandler: puzzleweb.CreateTemplate(tracer, "blogWidget/previewHandler", func(data gin.H, c *gin.Context) (string, string) {
			title := c.PostForm("title")
			markdown := c.PostForm("markdown")

			if title == "" {
				return "", common.DefaultErrorRedirect(emptyTitle)
			}
			if markdown == "" {
				return "", common.DefaultErrorRedirect(emptyContent)
			}

			html, err := markdownService.Apply(puzzleweb.GetLogger(c), markdown)
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			data[common.BaseUrlName] = common.GetBaseUrl(1, c)
			data["PreviewTitle"] = title
			data["Markdown"] = markdown
			data["PreviewHTML"] = html
			return previewTmpl, ""
		}),
		saveHandler: common.CreateRedirect(tracer, "blogWidget/saveHandler", func(c *gin.Context) string {
			logger := puzzleweb.GetLogger(c)
			title := c.PostForm("title")
			userId := puzzleweb.GetSessionUserId(logger, c)
			markdown := c.PostForm("markdown")

			if title == "" {
				return common.DefaultErrorRedirect(emptyTitle)
			}
			if markdown == "" {
				return common.DefaultErrorRedirect(emptyContent)
			}

			html, err := markdownService.Apply(logger, markdown)
			if err != nil {
				return common.DefaultErrorRedirect(err.Error())
			}

			postId, err := blogService.CreatePost(logger, userId, title, string(html))
			if err != nil {
				return common.DefaultErrorRedirect(err.Error())
			}

			err = commentService.CreateCommentThread(logger, userId, title)
			if err != nil {
				return common.DefaultErrorRedirect(err.Error())
			}
			return postUrlBuilder(common.GetBaseUrl(1, c), postId).String()
		}),
		deleteHandler: common.CreateRedirect(tracer, "blogWidget/deleteHandler", func(c *gin.Context) string {
			logger := puzzleweb.GetLogger(c)
			var targetBuilder strings.Builder
			targetBuilder.WriteString(common.GetBaseUrl(2, c))

			postId, err := strconv.ParseUint(c.Param(postIdName), 10, 64)
			if err != nil {
				logger.Warn(parsingPostIdErrorMsg, zap.Error(err))
				common.WriteError(&targetBuilder, common.ErrTechnical.Error())
				return targetBuilder.String()
			}
			userId := puzzleweb.GetSessionUserId(logger, c)

			post, err := blogService.GetPost(logger, userId, postId)
			if err != nil {
				common.WriteError(&targetBuilder, err.Error())
				return targetBuilder.String()
			}

			if err = blogService.DeletePost(logger, userId, postId); err != nil {
				common.WriteError(&targetBuilder, err.Error())
				return targetBuilder.String()
			}

			if err = commentService.DeleteCommentThread(logger, userId, post.Title); err != nil {
				common.WriteError(&targetBuilder, err.Error())
			}
			return targetBuilder.String()
		}),
	}
	return p
}

func postUrlBuilder(base string, postId uint64) *strings.Builder {
	targetBuilder := new(strings.Builder)
	targetBuilder.WriteString(base)
	targetBuilder.WriteString("view/")
	targetBuilder.WriteString(strconv.FormatUint(postId, 10))
	return targetBuilder
}

func filterPostsExtract(posts []service.BlogPost, extractSize uint64) {
	for index := range posts {
		posts[index].Content = template.HTML(common.FilterExtractHtml(string(posts[index].Content), extractSize))
	}
}
