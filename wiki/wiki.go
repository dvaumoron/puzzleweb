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
	"fmt"
	"html/template"
	"net/url"
	"strings"

	"github.com/dvaumoron/puzzleweb"
	"github.com/dvaumoron/puzzleweb/errors"
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/dvaumoron/puzzleweb/log"
	"github.com/dvaumoron/puzzleweb/login"
	"github.com/dvaumoron/puzzleweb/wiki/client"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type VersionDisplay struct {
	Lang           string
	Title          string
	Number         string
	BaseUrl        string
	ViewLinkName   string
	DeleteLinkName string
}

type wikiWidget struct {
	wikiId      uint64
	defaultPage string
	viewTmpl    string
	editTmpl    string
	listTmpl    string
}

func (w *wikiWidget) LoadInto(router gin.IRouter) {
	const viewMode = "/view/"
	const listMode = "/list/"
	const titleName = "title"
	const wikiTitleName = "WikiTitle"
	const wikiContentName = "WikiContent"

	router.GET("/", puzzleweb.CreateRedirect(func(c *gin.Context) string {
		return urlBuilder(
			puzzleweb.GetCurrentUrl(c), locale.GetLang(c),
			viewMode, w.defaultPage,
		).String()
	}))
	router.GET("/:lang/view/:title", puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
		askedLang := c.Param(locale.LangName)
		title := c.Param(titleName)
		lang := locale.CheckLang(askedLang)

		base := getBase(c)

		redirect := ""
		if lang == askedLang {
			userId := login.GetUserId(c)
			version := c.Query(client.VersionName)
			content, err := client.LoadContent(w.wikiId, userId, lang, title, version)
			if err == nil {
				if content == nil {
					redirect = urlBuilder(base, lang, "/edit/", title).String()
				} else {
					var body template.HTML
					body, err = content.GetBody()
					if err == nil {
						data[wikiTitleName] = title
						if version != "" {
							data["EditLinkName"] = locale.GetText("edit.link.name", c)
						}
						data[wikiContentName] = body
					} else {
						log.Logger.Info("Failed to apply markdown.",
							zap.Error(err),
						)
						redirect = errors.DefaultErrorRedirect(err.Error(), c)
					}
				}
			} else {
				redirect = errors.DefaultErrorRedirect(err.Error(), c)
			}
		} else {
			targetBuilder := urlBuilder(base, lang, viewMode, title)
			writeError(targetBuilder, errors.WrongLang, c)
			redirect = targetBuilder.String()
		}
		return w.viewTmpl, redirect
	}))
	router.GET("/:lang/edit/:title", puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
		const wikiVersionName = "WikiVersion"

		askedLang := c.Param(locale.LangName)
		title := c.Param(titleName)
		lang := locale.CheckLang(askedLang)

		redirect := ""
		if lang == askedLang {
			userId := login.GetUserId(c)
			content, err := client.LoadContent(w.wikiId, userId, lang, title, "")
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
				redirect = errors.DefaultErrorRedirect(err.Error(), c)
			}
		} else {
			targetBuilder := urlBuilder(getBase(c), lang, viewMode, title)
			writeError(targetBuilder, errors.WrongLang, c)
			redirect = targetBuilder.String()
		}
		return w.editTmpl, redirect
	}))
	router.POST("/:lang/save/:title", puzzleweb.CreateRedirect(func(c *gin.Context) string {
		askedLang := c.Param(locale.LangName)
		title := c.Param(titleName)
		lang := locale.CheckLang(askedLang)

		targetBuilder := urlBuilder(getBase(c), lang, viewMode, title)
		if lang == askedLang {
			content := c.PostForm("content")
			last := c.PostForm(client.VersionName)

			userId := login.GetUserId(c)
			err := client.StoreContent(w.wikiId, userId, lang, title, last, content)
			if err != nil {
				writeError(targetBuilder, err.Error(), c)
			}
		} else {
			writeError(targetBuilder, errors.WrongLang, c)
		}
		return targetBuilder.String()
	}))
	router.GET("/:lang/list/:title", puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
		const versionsName = "Versions"

		askedLang := c.Param(locale.LangName)
		title := c.Param(titleName)
		lang := locale.CheckLang(askedLang)

		redirect := ""
		base := getBase(c)
		if lang == askedLang {
			userId := login.GetUserId(c)
			versions, err := client.GetVersions(w.wikiId, userId, lang, title)
			if err == nil {
				data[wikiTitleName] = title
				size := len(versions)
				if size == 0 {
					data[errors.Msg] = locale.GetText(errors.NoElement, c)
					data[versionsName] = versions
				} else {
					viewLinkName := locale.GetText("view.link.name", c)
					deleteLinkName := locale.GetText("delete.link.name", c)

					converted := make([]*VersionDisplay, 0, size)
					for _, version := range versions {
						converted = append(converted, &VersionDisplay{
							Lang: lang, Title: title, Number: fmt.Sprint(version),
							BaseUrl: base, ViewLinkName: viewLinkName,
							DeleteLinkName: deleteLinkName,
						})
					}
					data[versionsName] = converted
				}
			} else {
				targetBuilder := urlBuilder(base, lang, listMode, title)
				writeError(targetBuilder, err.Error(), c)
				redirect = targetBuilder.String()
			}
		} else {
			targetBuilder := urlBuilder(base, lang, listMode, title)
			writeError(targetBuilder, errors.WrongLang, c)
			redirect = targetBuilder.String()
		}
		return w.listTmpl, redirect
	}))
	router.GET("/:lang/delete/:title", puzzleweb.CreateRedirect(func(c *gin.Context) string {
		askedLang := c.Param(locale.LangName)
		title := c.Param(titleName)
		lang := locale.CheckLang(askedLang)

		targetBuilder := urlBuilder(getBase(c), lang, listMode, title)
		if lang == askedLang {
			userId := login.GetUserId(c)
			version := c.Query(client.VersionName)
			err := client.DeleteContent(w.wikiId, userId, lang, title, version)
			if err != nil {
				writeError(targetBuilder, err.Error(), c)
			}
		} else {
			writeError(targetBuilder, errors.WrongLang, c)
		}
		return targetBuilder.String()
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
	res := puzzleweb.GetCurrentUrl(c)
	i := len(res) - 2
	count := 0
	for count < 3 {
		if res[i] == '/' {
			count++
		}
		i--
	}
	return res[:i+1]
}

func writeError(builder *strings.Builder, errMsg string, c *gin.Context) {
	builder.WriteString(errors.QueryError)
	builder.WriteString(url.QueryEscape(locale.GetText(errMsg, c)))
}