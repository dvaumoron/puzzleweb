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

	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
)

type Widget interface {
	LoadInto(gin.IRouter)
}

type InitDataFunc func(c *gin.Context) gin.H

type Site struct {
	engine             *gin.Engine
	root               *Page
	Page404Url         string
	InitData           InitDataFunc
	availableLanguages []language.Tag
}

const siteName = "site"

func CreateSite(defaultLang language.Tag) *Site {
	engine := gin.Default()

	engine.LoadHTMLGlob(config.TemplatesPath + "/**/*.html")

	engine.Static("/static", config.StaticPath)
	const favicon = "/favicon.ico"
	engine.StaticFile(favicon, config.StaticPath+favicon)

	engine.Use(manageSession)

	site := &Site{
		engine:             engine,
		root:               NewStaticPage("root", "index.html"),
		Page404Url:         "/",
		InitData:           initData,
		availableLanguages: []language.Tag{defaultLang},
	}

	engine.Use(func(c *gin.Context) {
		c.Set(siteName, site)
	})

	return site
}

func (site *Site) AddPage(page *Page) {
	site.root.AddSubPage(page)
}

func (site *Site) AddAvailableLanguage(lang language.Tag) {
	site.availableLanguages = append(site.availableLanguages, lang)
}

func (site *Site) Run() error {
	engine := site.engine
	site.root.Widget.LoadInto(engine)
	engine.NoRoute(Found(site.Page404Url))
	locale.InitAvailableLanguages(site.availableLanguages)
	return engine.Run(":" + config.Port)
}

func getSite(c *gin.Context) *Site {
	siteAny, _ := c.Get(siteName)
	return siteAny.(*Site)
}

func initData(c *gin.Context) gin.H {
	page, path := getSite(c).root.extractPageAndPath(c.Request.URL.Path)
	return gin.H{
		"ariane":   extractAriane(path),
		"subPages": page.extractSubPageNames(),
	}
}

type InfoAdder func(gin.H, *gin.Context)

func CreateTemplateHandler(tmplName string, adder InfoAdder) gin.HandlerFunc {
	return func(c *gin.Context) {
		data := getSite(c).InitData(c)
		adder(data, c)
		c.HTML(http.StatusOK, tmplName, data)
	}
}

func AddNothing(data gin.H, c *gin.Context) {}

type Redirecter func(*gin.Context) string

func CreateRedirectHandler(redirecter Redirecter) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Redirect(http.StatusFound, redirecter(c))
	}
}

func Found(target string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Redirect(http.StatusFound, target)
	}
}
