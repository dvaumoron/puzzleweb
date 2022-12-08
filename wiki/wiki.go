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
	"html/template"
	"net/url"
	"strings"

	"github.com/dvaumoron/puzzleweb"
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/dvaumoron/puzzleweb/wiki/client"
	"github.com/gin-gonic/gin"
)

type wikiWidget struct {
	wikiId      uint64
	defaultPage string
	viewTmpl    string
	editTmpl    string
}

func (w *wikiWidget) LoadInto(router gin.IRouter) {
	const viewMode = "/view/"
	const editMode = "/edit/"
	const titleName = "title"
	const wikiTitleName = "WikiTitle"
	const wikiContentName = "WikiContent"
	const wrongLang = "wrong.lang"

	router.GET("/", puzzleweb.CreateRedirect(func(c *gin.Context) string {
		return urlBuilder(
			puzzleweb.GetCurrentUrl(c), locale.GetLang(c),
			viewMode, w.defaultPage,
		).String()
	}))
	router.GET("/:Lang/view/:title", puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
		askedLang := c.Param(locale.LangName)
		title := c.Param(titleName)
		lang := locale.CheckLang(askedLang)

		redirect := ""
		if lang == askedLang {
			userId := uint64(0) // TODO
			version := c.Query(client.VersionName)
			content, err := client.LoadContent(w.wikiId, userId, lang, title, version)
			if err == nil {
				if content == nil {
					redirect = urlBuilder(getBase(c), lang, editMode, title).String()
				} else {
					var body template.HTML
					body, err = content.GetBody()
					if err == nil {
						data[wikiTitleName] = title
						data["EditLinkName"] = locale.GetText("edit.link.name", c)
						data[wikiContentName] = body
					} else {
						redirect = puzzleweb.DefaultErrorRedirect(
							locale.GetText(err.Error(), c),
						)
					}
				}
			} else {
				redirect = puzzleweb.DefaultErrorRedirect(
					locale.GetText(err.Error(), c),
				)
			}
		} else {
			targetBuilder := urlBuilder(getBase(c), lang, viewMode, title)
			targetBuilder.WriteString(puzzleweb.QueryError)
			targetBuilder.WriteString(url.QueryEscape(locale.GetText(wrongLang, c)))
			redirect = targetBuilder.String()
		}
		return w.viewTmpl, redirect
	}))
	router.GET("/:Lang/edit/:title", puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
		const wikiVersionName = "WikiVersion"

		askedLang := c.Param(locale.LangName)
		title := c.Param(titleName)
		lang := locale.CheckLang(askedLang)

		redirect := ""
		if lang == askedLang {
			userId := uint64(0) // TODO
			version := c.Query(client.VersionName)
			content, err := client.LoadContent(w.wikiId, userId, lang, title, version)
			if err == nil {
				data["EditTitle"] = locale.GetText("edit.title", c)
				data[wikiTitleName] = title
				data["CancelLinkName"] = locale.GetText("cancel.link.name", c)
				if content == nil {
					data[wikiVersionName] = "0"
				} else {
					data[wikiContentName] = content.Markdown
					data[wikiVersionName] = content.Version
				}
				data["SaveLinkName"] = locale.GetText("save.link.name", c)
			} else {
				redirect = puzzleweb.DefaultErrorRedirect(
					locale.GetText(err.Error(), c),
				)
			}
		} else {
			targetBuilder := urlBuilder(getBase(c), lang, editMode, title)
			targetBuilder.WriteString(puzzleweb.QueryError)
			targetBuilder.WriteString(url.QueryEscape(locale.GetText(wrongLang, c)))
			redirect = targetBuilder.String()
		}
		return w.editTmpl, redirect
	}))
	router.POST("/:Lang/save/:title", puzzleweb.CreateRedirect(func(c *gin.Context) string {
		askedLang := c.Param(locale.LangName)
		title := c.Param(titleName)
		lang := locale.CheckLang(askedLang)

		redirect := ""
		if lang == askedLang {
			content := c.PostForm("content")
			version := c.PostForm(client.VersionName)

			userId := uint64(0) // TODO
			err := client.StoreContent(w.wikiId, userId, lang, title, version, content)

			targetBuilder := urlBuilder(getBase(c), lang, viewMode, title)
			if err != nil {
				targetBuilder.WriteString(puzzleweb.QueryError)
				targetBuilder.WriteString(url.QueryEscape(
					locale.GetText(err.Error(), c),
				))
			}
			redirect = targetBuilder.String()
		} else {
			redirect = puzzleweb.DefaultErrorRedirect(locale.GetText(wrongLang, c))
		}
		return redirect
	}))
}

func urlBuilder(base, lang, mode, title string) *strings.Builder {
	var targetBuilder strings.Builder
	targetBuilder.WriteString(base)
	targetBuilder.WriteString(lang)
	targetBuilder.WriteString(mode)
	targetBuilder.WriteString(title)
	return &targetBuilder
}

func getBase(c *gin.Context) string {
	return puzzleweb.GetCurrentUrl(c) + "../../../"
}
