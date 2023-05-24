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
package wiki

import (
	"strconv"
	"strings"

	"github.com/dvaumoron/puzzleweb"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/gin-gonic/gin"
)

const versionName = "version"
const versionsName = "Versions"
const viewMode = "/view/"
const listMode = "/list/"
const titleName = "title"
const wikiTitleName = "WikiTitle"
const wikiVersionName = "WikiVersion"
const wikiContentName = "WikiContent"

type wikiWidget struct {
	defaultHandler gin.HandlerFunc
	viewHandler    gin.HandlerFunc
	editHandler    gin.HandlerFunc
	saveHandler    gin.HandlerFunc
	listHandler    gin.HandlerFunc
	deleteHandler  gin.HandlerFunc
}

func (w wikiWidget) LoadInto(router gin.IRouter) {
	router.GET("/", w.defaultHandler)
	router.GET("/:lang/view/:title", w.viewHandler)
	router.GET("/:lang/edit/:title", w.editHandler)
	router.POST("/:lang/save/:title", w.saveHandler)
	router.GET("/:lang/list/:title", w.listHandler)
	router.GET("/:lang/delete/:title", w.deleteHandler)
}

func MakeWikiPage(wikiName string, wikiConfig config.WikiConfig) puzzleweb.Page {
	tracer := wikiConfig.Tracer
	wikiService := wikiConfig.Service
	markdownService := wikiConfig.MarkdownService

	defaultPage := "Welcome"
	viewTmpl := "wiki/view"
	editTmpl := "wiki/edit"
	listTmpl := "wiki/list"
	switch args := wikiConfig.Args; len(args) {
	default:
		wikiConfig.Logger.Info("MakeWikiPage should be called with 0 to 4 optional arguments.")
		fallthrough
	case 4:
		if args[3] != "" {
			listTmpl = args[3]
		}
		fallthrough
	case 3:
		if args[2] != "" {
			editTmpl = args[2]
		}
		fallthrough
	case 2:
		if args[1] != "" {
			viewTmpl = args[1]
		}
		fallthrough
	case 1:
		if args[0] != "" {
			defaultPage = args[0]
		}
	case 0:
	}

	p := puzzleweb.MakePage(wikiName)
	p.Widget = wikiWidget{
		defaultHandler: common.CreateRedirect(tracer, "wikiWidget/defaultHandler", func(c *gin.Context) string {
			lang := puzzleweb.GetLocalesManager(c).GetLang(c)
			return wikiUrlBuilder(common.GetCurrentUrl(c), lang, viewMode, defaultPage).String()
		}),
		viewHandler: puzzleweb.CreateTemplate(tracer, "wikiWidget/viewHandler", func(data gin.H, c *gin.Context) (string, string) {
			logger := puzzleweb.GetLogger(c)
			askedLang := c.Param(locale.LangName)
			title := c.Param(titleName)
			lang := puzzleweb.GetLocalesManager(c).CheckLang(askedLang)

			if lang != askedLang {
				targetBuilder := wikiUrlBuilder(common.GetBaseUrl(3, c), lang, viewMode, title)
				common.WriteError(targetBuilder, common.WrongLangKey)
				return "", targetBuilder.String()
			}

			userId, _ := data[common.IdName].(uint64)
			version := c.Query(versionName)
			content, err := wikiService.LoadContent(logger, userId, lang, title, version)
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			if content == nil {
				base := common.GetBaseUrl(3, c)
				if version == "" {
					return "", wikiUrlBuilder(base, lang, "/edit/", title).String()
				}
				return "", wikiUrlBuilder(base, lang, viewMode, title).String()
			}

			body, err := content.GetBody(logger, markdownService)
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			data[wikiTitleName] = title
			if version != "" {
				data[wikiVersionName] = strconv.FormatUint(content.Version, 10)
			}
			data[common.BaseUrlName] = common.GetBaseUrl(2, c)
			data[wikiContentName] = body
			return viewTmpl, ""
		}),
		editHandler: puzzleweb.CreateTemplate(tracer, "wikiWidget/editHandler", func(data gin.H, c *gin.Context) (string, string) {
			logger := puzzleweb.GetLogger(c)
			askedLang := c.Param(locale.LangName)
			title := c.Param(titleName)
			lang := puzzleweb.GetLocalesManager(c).CheckLang(askedLang)

			if lang != askedLang {
				targetBuilder := wikiUrlBuilder(common.GetBaseUrl(3, c), lang, viewMode, title)
				common.WriteError(targetBuilder, common.WrongLangKey)
				return "", targetBuilder.String()
			}

			userId, _ := data[common.IdName].(uint64)
			content, err := wikiService.LoadContent(logger, userId, lang, title, "")
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			data[wikiTitleName] = title
			data[common.BaseUrlName] = common.GetBaseUrl(2, c)
			if content == nil {
				data[wikiVersionName] = "0"
			} else {
				data[wikiVersionName] = strconv.FormatUint(content.Version, 10)
				data[wikiContentName] = content.Markdown
			}
			return editTmpl, ""
		}),
		saveHandler: common.CreateRedirect(tracer, "wikiWidget/saveHandler", func(c *gin.Context) string {
			logger := puzzleweb.GetLogger(c)
			askedLang := c.Param(locale.LangName)
			lang := puzzleweb.GetLocalesManager(c).CheckLang(askedLang)
			title := c.Param(titleName)

			targetBuilder := wikiUrlBuilder(common.GetBaseUrl(3, c), lang, viewMode, title)
			if lang != askedLang {
				common.WriteError(targetBuilder, common.WrongLangKey)
				return targetBuilder.String()
			}

			userId := puzzleweb.GetSessionUserId(logger, c)
			last := c.PostForm(versionName)
			content := c.PostForm("content")

			success, err := wikiService.StoreContent(logger, userId, lang, title, last, content)
			if err != nil {
				common.WriteError(targetBuilder, err.Error())
				return targetBuilder.String()
			}
			if !success {
				common.WriteError(targetBuilder, "BaseVersionOutdated")
			}
			return targetBuilder.String()
		}),
		listHandler: puzzleweb.CreateTemplate(tracer, "wikiWidget/listHandler", func(data gin.H, c *gin.Context) (string, string) {
			logger := puzzleweb.GetLogger(c)
			askedLang := c.Param(locale.LangName)
			lang := puzzleweb.GetLocalesManager(c).CheckLang(askedLang)
			title := c.Param(titleName)

			targetBuilder := wikiUrlBuilder(common.GetBaseUrl(3, c), lang, listMode, title)
			if lang != askedLang {
				common.WriteError(targetBuilder, common.WrongLangKey)
				return "", targetBuilder.String()
			}

			userId, _ := data[common.IdName].(uint64)
			versions, err := wikiService.GetVersions(logger, userId, lang, title)
			if err != nil {
				common.WriteError(targetBuilder, err.Error())
				return "", targetBuilder.String()
			}

			data[wikiTitleName] = title
			data[versionsName] = versions
			data[common.BaseUrlName] = common.GetBaseUrl(2, c)
			data[common.AllowedToDeleteName] = wikiService.DeleteRight(logger, userId)
			puzzleweb.InitNoELementMsg(data, len(versions), c)
			return listTmpl, ""
		}),
		deleteHandler: common.CreateRedirect(tracer, "wikiWidget/deleteHandler", func(c *gin.Context) string {
			logger := puzzleweb.GetLogger(c)
			askedLang := c.Param(locale.LangName)
			lang := puzzleweb.GetLocalesManager(c).CheckLang(askedLang)
			title := c.Param(titleName)

			targetBuilder := wikiUrlBuilder(common.GetBaseUrl(3, c), lang, listMode, title)
			if lang != askedLang {
				common.WriteError(targetBuilder, common.WrongLangKey)
				return targetBuilder.String()
			}

			userId := puzzleweb.GetSessionUserId(logger, c)
			version := c.Query(versionName)
			err := wikiService.DeleteContent(logger, userId, lang, title, version)
			if err != nil {
				common.WriteError(targetBuilder, err.Error())
			}
			return targetBuilder.String()
		}),
	}
	return p
}

func wikiUrlBuilder(base string, lang string, mode string, title string) *strings.Builder {
	targetBuilder := new(strings.Builder)
	targetBuilder.WriteString(base)
	targetBuilder.WriteString(lang)
	targetBuilder.WriteString(mode)
	targetBuilder.WriteString(title)
	return targetBuilder
}
