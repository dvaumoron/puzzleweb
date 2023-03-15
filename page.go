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
	"github.com/gin-gonic/gin"
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
		userId, _ := data[common.IdName].(uint64)
		err := site.authService.AuthQuery(userId, groupId, adminservice.ActionAccess)
		if err != nil {
			return "", common.DefaultErrorRedirect(err.Error())
		}
		localesManager := GetLocalesManager(c)
		if lang := localesManager.GetLang(c); lang != localesManager.GetDefaultLang() {
			site.logger.Info("Using alternative static page", zap.String(locale.LangName, lang))
			var builder strings.Builder
			builder.WriteString(lang)
			builder.WriteString("/")
			builder.WriteString(tmpl)
			return builder.String(), ""
		}
		return tmpl, ""
	}
}

func newStaticWidget(groupId uint64, tmpl string) *staticWidget {
	return &staticWidget{displayHandler: CreateTemplate(localizedTmpl(groupId, tmpl))}
}

func MakeStaticPage(name string, groupId uint64, tmpl string) Page {
	p := MakePage(name)
	p.Widget = newStaticWidget(groupId, tmpl)
	return p
}

func MakeHiddenStaticPage(name string, groupId uint64, tmpl string) Page {
	p := MakeHiddenPage(name)
	p.Widget = newStaticWidget(groupId, tmpl)
	return p
}

func (p Page) AddSubPage(page Page) {
	sw, ok := p.Widget.(*staticWidget)
	if ok {
		sw.addSubPage(page)
	}
}

func (p Page) getSubPage(name string) (Page, bool) {
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

func (current Page) extractPageAndPath(path string) (Page, []string) {
	splitted := strings.Split(path, "/")[1:]
	names := make([]string, 0, len(splitted))
	for _, name := range splitted {
		subPage, ok := current.getSubPage(name)
		if !ok {
			break
		}
		current = subPage
		names = append(names, name)
	}
	return current, names
}

func (p Page) extractSubPageNames(messages map[string]string, url string, c *gin.Context) []PageDesc {
	sw, ok := p.Widget.(*staticWidget)
	if !ok {
		return nil
	}

	pages := sw.subPages
	size := len(pages)
	if size == 0 {
		return nil
	}

	pageDescs := make([]PageDesc, 0, size)
	for _, page := range pages {
		if page.visible {
			name := page.name
			pageDescs = append(pageDescs, makePageDesc(messages, name, url+name))
		}
	}
	return pageDescs
}

func CreateTemplate(redirecter common.TemplateRedirecter) gin.HandlerFunc {
	return func(c *gin.Context) {
		data := initData(c)
		tmpl, redirect := redirecter(data, c)
		if redirect == "" {
			c.HTML(http.StatusOK, tmpl, data)
		} else {
			c.Redirect(http.StatusFound, redirect)
		}
	}
}
