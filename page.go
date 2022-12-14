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
	"strings"

	"github.com/dvaumoron/puzzleweb/admin/client"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/dvaumoron/puzzleweb/session"
	"github.com/gin-gonic/gin"
)

type Widget interface {
	LoadInto(gin.IRouter)
}

type Page struct {
	name    string
	visible bool
	Widget  Widget
}

func NewPage(name string) *Page {
	return &Page{name: name, visible: true}
}

func NewHiddenPage(name string) *Page {
	return &Page{name: name, visible: false}
}

type staticWidget struct {
	displayHandler gin.HandlerFunc
	subPages       []*Page
}

func (w *staticWidget) addSubPage(page *Page) {
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
		redirect := ""
		err := client.AuthQuery(session.GetUserId(c), groupId, client.ActionAccess)
		if err == nil {
			if lang := locale.GetLang(c); lang != locale.DefaultLang {
				var builder strings.Builder
				builder.WriteString(lang)
				builder.WriteString("/")
				builder.WriteString(tmpl)
				tmpl = builder.String()
			}
		} else {
			redirect = common.DefaultErrorRedirect(err.Error(), c)
		}
		return tmpl, redirect
	}
}

func newStaticWidget(groupId uint64, tmpl string) Widget {
	return &staticWidget{displayHandler: CreateTemplate(localizedTmpl(groupId, tmpl))}
}

func NewStaticPage(name string, groupId uint64, tmpl string) *Page {
	p := NewPage(name)
	p.Widget = newStaticWidget(groupId, tmpl)
	return p
}

func NewHiddenStaticPage(name string, groupId uint64, tmpl string) *Page {
	p := NewHiddenPage(name)
	p.Widget = newStaticWidget(groupId, tmpl)
	return p
}

func (p *Page) AddSubPage(page *Page) {
	sw, ok := p.Widget.(*staticWidget)
	if ok {
		sw.addSubPage(page)
	}
}

func (p *Page) getSubPage(name string) *Page {
	sw, ok := p.Widget.(*staticWidget)
	if ok {
		for _, sub := range sw.subPages {
			if sub.name == name {
				return sub
			}
		}
	}
	return nil
}

func (current *Page) extractPageAndPath(path string) (*Page, []string) {
	splitted := strings.Split(path, "/")[1:]
	names := make([]string, 0, len(splitted))
	for _, name := range splitted {
		subPage := current.getSubPage(name)
		if subPage == nil {
			break
		}
		current = subPage
		names = append(names, name)
	}
	return current, names
}

func (p *Page) extractSubPageNames(c *gin.Context) []PageDesc {
	var pageDescs []PageDesc
	sw, ok := p.Widget.(*staticWidget)
	if ok {
		pages := sw.subPages
		if size := len(pages); size != 0 {
			url := common.GetCurrentUrl(c)
			pageDescs = make([]PageDesc, 0, size)
			for _, page := range pages {
				if page.visible {
					pageDescs = append(pageDescs, makePageDesc(page.name, url+page.name, c))
				}
			}
		}
	}
	return pageDescs
}
