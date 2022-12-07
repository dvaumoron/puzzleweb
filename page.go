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

	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/gin-gonic/gin"
)

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
	tmplName string
	subPages []*Page
}

func (w *staticWidget) addSubPage(page *Page) {
	w.subPages = append(w.subPages, page)
}

func (w *staticWidget) LoadInto(router gin.IRouter) {
	router.GET("/", CreateTemplate(LocalizedTmpl(w.tmplName, AddNothing)))
	for _, page := range w.subPages {
		page.Widget.LoadInto(router.Group("/" + page.name))
	}
}

func LocalizedTmpl(tmplName string, adder DataAdder) DataRedirecter {
	return func(data gin.H, c *gin.Context) string {
		adder(data, c)
		lang := locale.GetLang(c)
		if lang != locale.DefaultLang {
			var builder strings.Builder
			builder.WriteString(lang)
			builder.WriteString("/")
			builder.WriteString(tmplName)
			tmplName = builder.String()
		}
		return tmplName
	}
}

func AddNothing(data gin.H, c *gin.Context) {}

func newStaticWidget(tmplName string) Widget {
	return &staticWidget{tmplName: tmplName, subPages: make([]*Page, 0)}
}

func NewStaticPage(name, tmplName string) *Page {
	p := NewPage(name)
	p.Widget = newStaticWidget(tmplName)
	return p
}

func NewHiddenStaticPage(name, tmplName string) *Page {
	p := NewHiddenPage(name)
	p.Widget = newStaticWidget(tmplName)
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

func (p *Page) extractSubPageNames() []string {
	var names []string
	sw, ok := p.Widget.(*staticWidget)
	if ok {
		pages := sw.subPages
		if size := len(pages); size == 0 {
			names = make([]string, 0)
		} else {
			names = make([]string, 0, size)
			for _, page := range pages {
				if page.visible {
					names = append(names, page.name)
				}
			}
		}
	} else {
		names = make([]string, 0)
	}
	return names
}
