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

package puzzleweb

import (
	"net/http"
	"strings"

	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/common/config/parser"
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/dvaumoron/puzzleweb/templates"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"
)

type Widget interface {
	LoadInto(gin.IRouter)
}

type Page struct {
	name    string
	visible bool
	Widget  Widget
}

func MakePage(name string) Page {
	return Page{name: name, visible: true}
}

func MakeHiddenPage(name string) Page {
	return Page{name: name, visible: false}
}

type staticWidget struct {
	displayHandler gin.HandlerFunc
	subPages       []Page
}

func (w *staticWidget) addSubPage(page Page) {
	w.subPages = append(w.subPages, page)
}

func (w *staticWidget) LoadInto(router gin.IRouter) {
	router.GET("/", w.displayHandler)
	for _, page := range w.subPages {
		page.Widget.LoadInto(router.Group("/" + page.name))
	}
}

func newStaticWidget(groupId uint64, templateName string) *staticWidget {
	return &staticWidget{displayHandler: CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
		site := getSite(c)
		ctx := c.Request.Context()
		logger := site.loggerGetter.Logger(ctx)
		userId, _ := data[common.UserIdName].(uint64)
		err := site.authService.AuthQuery(ctx, userId, groupId, adminservice.ActionAccess)
		if err != nil {
			return "", common.DefaultErrorRedirect(logger, err.Error())
		}
		localesManager := GetLocalesManager(c)
		if lang := localesManager.GetLang(c); lang != localesManager.GetDefaultLang() {
			logger.Info("Using alternative static page", zap.String(locale.LangName, lang))
			var builder strings.Builder
			builder.WriteString(lang)
			builder.WriteByte('/')
			builder.WriteString(templateName)
			return builder.String(), ""
		}
		return templateName, ""
	})}
}

func MakeStaticPage(name string, groupId uint64, templateName string) Page {
	p := MakePage(name)
	p.Widget = newStaticWidget(groupId, templateName)
	return p
}

func MakeHiddenStaticPage(name string, groupId uint64, templateName string) Page {
	p := MakeHiddenPage(name)
	p.Widget = newStaticWidget(groupId, templateName)
	return p
}

func (p Page) AddSubPage(page Page) bool {
	sw, ok := p.Widget.(*staticWidget)
	if ok {
		sw.addSubPage(page)
	}
	return ok
}

func (p Page) AddStaticPages(pageGroup parser.StaticPagesConfig) bool {
	for _, pagePath := range pageGroup.Locations {
		subPage, pageName, templateName, ok := p.extractSubPageAndNamesFromPath(pagePath)
		if !ok {
			return false
		}

		var newPage Page
		if pageGroup.Hidden {
			newPage = MakeHiddenStaticPage(pageName, pageGroup.GroupId, templateName)
		} else {
			newPage = MakeStaticPage(pageName, pageGroup.GroupId, templateName)
		}
		if !subPage.AddSubPage(newPage) {
			return false
		}
	}
	return true
}

func (p Page) GetSubPage(name string) (Page, bool) {
	if name == "" {
		return Page{}, false
	}
	sw, ok := p.Widget.(*staticWidget)
	if ok {
		for _, sub := range sw.subPages {
			if sub.name == name {
				return sub, true
			}
		}
	}
	return Page{}, false
}

func (p Page) GetSubPageWithPath(path string) (Page, bool) {
	return p.getPageWithSplittedPath(strings.Split(path, "/"))
}

func (current Page) getPageWithSplittedPath(splittedPath []string) (Page, bool) {
	for _, name := range splittedPath {
		subPage, ok := current.GetSubPage(name)
		if !ok {
			return current, false
		}
		current = subPage
	}
	return current, true
}

func (p Page) extractSubPageAndNamesFromPath(path string) (Page, string, string, bool) {
	splitted := strings.Split(path, "/")
	last := len(splitted) - 1
	if splitted[last] == "" {
		last--
		path += "index"
	}
	resPage, ok := p.getPageWithSplittedPath(splitted[:last])
	return resPage, splitted[last], path, ok
}

func CreateTemplate(redirecter common.TemplateRedirecter) gin.HandlerFunc {
	return func(c *gin.Context) {
		data := initData(c)
		if tmpl, redirect := redirecter(data, c); redirect == "" {
			if pagePart := c.Query("pagePart"); pagePart != "" {
				var tmplBuilder strings.Builder
				tmplBuilder.WriteString(tmpl)
				tmplBuilder.WriteByte('#')
				tmplBuilder.WriteString(pagePart)
				tmpl = tmplBuilder.String()
			}
			otelgin.HTML(c, http.StatusOK, tmpl, templates.ContextAndData{
				Ctx: c.Request.Context(), Data: data,
			})
		} else {
			c.Redirect(http.StatusFound, redirect)
		}
	}
}
