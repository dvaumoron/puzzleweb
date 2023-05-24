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
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/dvaumoron/puzzleweb/templates"
	"github.com/gin-gonic/gin"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/trace"
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

func localizedTmpl(groupId uint64, tmpl string) common.TemplateRedirecter {
	return func(data gin.H, c *gin.Context) (string, string) {
		site := getSite(c)
		logger := site.logger.Ctx(c.Request.Context())
		userId, _ := data[common.IdName].(uint64)
		err := site.authService.AuthQuery(logger, userId, groupId, adminservice.ActionAccess)
		if err != nil {
			return "", common.DefaultErrorRedirect(err.Error())
		}
		localesManager := GetLocalesManager(c)
		if lang := localesManager.GetLang(c); lang != localesManager.GetDefaultLang() {
			logger.Info("Using alternative static page", zap.String(locale.LangName, lang))
			var builder strings.Builder
			builder.WriteString(lang)
			builder.WriteByte('/')
			builder.WriteString(tmpl)
			return builder.String(), ""
		}
		return tmpl, ""
	}
}

func newStaticWidget(tracer trace.Tracer, groupId uint64, tmpl string) *staticWidget {
	return &staticWidget{displayHandler: CreateTemplate(tracer, "staticWidget/displayHandler", localizedTmpl(groupId, tmpl))}
}

func MakeStaticPage(tracer trace.Tracer, name string, groupId uint64, tmpl string) Page {
	p := MakePage(name)
	p.Widget = newStaticWidget(tracer, groupId, tmpl)
	return p
}

func MakeHiddenStaticPage(tracer trace.Tracer, name string, groupId uint64, tmpl string) Page {
	p := MakeHiddenPage(name)
	p.Widget = newStaticWidget(tracer, groupId, tmpl)
	return p
}

func (p Page) AddSubPage(page Page) {
	sw, ok := p.Widget.(*staticWidget)
	if ok {
		sw.addSubPage(page)
	}
}

func (p Page) AddStaticPages(logger otelzap.LoggerWithCtx, tracer trace.Tracer, groupId uint64, pagePaths []string) {
	for _, pagePath := range pagePaths {
		if last := len(pagePath) - 1; pagePath[last] == '/' {
			currentPage, name := p.extractSubPageFromPath(pagePath[:last])
			currentPage.AddSubPage(MakeStaticPage(tracer, name, groupId, pagePath+"index"))
		} else {
			currentPage, name := p.extractSubPageFromPath(pagePath)
			currentPage.AddSubPage(MakeStaticPage(tracer, name, groupId, pagePath))
		}
	}
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

func (current Page) extractSubPageFromPath(path string) (Page, string) {
	splitted := strings.Split(path, "/")
	last := len(splitted) - 1
	for _, name := range splitted[:last] {
		subPage, ok := current.GetSubPage(name)
		if !ok {
			break
		}
		current = subPage
	}
	return current, splitted[last]
}

func CreateTemplate(tracer trace.Tracer, spanName string, redirecter common.TemplateRedirecter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		_, span := tracer.Start(ctx, spanName)
		defer span.End()
		data := initData(c)
		tmpl, redirect := redirecter(data, c)
		if redirect == "" {
			otelgin.HTML(c, http.StatusOK, tmpl, templates.ContextAndData{Ctx: ctx, Data: data})
		} else {
			c.Redirect(http.StatusFound, redirect)
		}
	}
}
