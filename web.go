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
	"github.com/dvaumoron/puzzleweb/log"
	"github.com/dvaumoron/puzzleweb/session"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"golang.org/x/sync/errgroup"
)

type Widget interface {
	LoadInto(gin.IRouter)
}

type InitDataFunc func(c *gin.Context) gin.H

var puzzleRender render.HTMLRender

type Site struct {
	engine      *gin.Engine
	root        *Page
	Page404Url  string
	InitData    InitDataFunc
	initialized bool
	FaviconPath string
}

const siteName = "site"
const RedirectName = "redirect"

func CreateSite(args ...string) *Site {
	var rootTmpl string
	if size := len(args); size == 0 {
		rootTmpl = "index.html"
	} else {
		rootTmpl = args[0]
		if size > 1 {
			log.Logger.Info("CreateSite should be called with 0 or 1 argument.")
		}
	}

	engine := gin.Default()

	if puzzleRender == nil {
		engine.LoadHTMLGlob(config.TemplatesPath + "/*")
		puzzleRender = engine.HTMLRender
	} else {
		engine.HTMLRender = puzzleRender
	}

	engine.Static("/static", config.StaticPath)

	engine.Use(session.Manage)

	site := &Site{
		engine:      engine,
		root:        NewStaticPage("root", rootTmpl),
		Page404Url:  "/",
		InitData:    initData,
		initialized: false,
	}

	engine.Use(func(c *gin.Context) {
		c.Set(siteName, site)
	})

	return site
}

func (site *Site) AddPage(page *Page) {
	site.root.AddSubPage(page)
}

func (site *Site) initEngine() *gin.Engine {
	engine := site.engine
	if !site.initialized {
		const favicon = "/favicon.ico"
		faviconPath := site.FaviconPath
		if faviconPath == "" {
			faviconPath = favicon
		}
		engine.StaticFile(favicon, config.StaticPath+faviconPath)
		site.root.Widget.LoadInto(engine)
		engine.GET("/changeLang", CreateRedirectHandler(func(c *gin.Context) string {
			locale.SetLangCookie(c, c.Query(locale.LangName))
			return c.Query(RedirectName)
		}))
		engine.NoRoute(Found(checkTarget(site.Page404Url)))
		site.initialized = true
	}
	return engine
}

func checkPort(port string) string {
	if port[0] != ':' {
		port = ":" + port
	}
	return port
}

func (site *Site) Run() error {
	locale.InitMessages()
	return site.initEngine().Run(checkPort(config.Port))
}

type SiteConfig struct {
	site *Site
	port string
}

func MakeSiteConfig(site *Site, port string) SiteConfig {
	return SiteConfig{site: site, port: checkPort(port)}
}

func Run(sites ...SiteConfig) error {
	locale.InitMessages()
	var g errgroup.Group
	for _, siteConfig := range sites {
		port := siteConfig.port
		handler := siteConfig.site.initEngine().Handler()
		g.Go(func() error {
			server := &http.Server{Addr: port, Handler: handler}
			return server.ListenAndServe()
		})
	}
	return g.Wait()
}

func getSite(c *gin.Context) *Site {
	siteAny, _ := c.Get(siteName)
	return siteAny.(*Site)
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

func checkTarget(target string) string {
	if target == "" {
		target = "/"
	}
	return target
}

func CreateRedirectHandler(redirecter Redirecter) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Redirect(http.StatusFound, checkTarget(redirecter(c)))
	}
}

func Found(target string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Redirect(http.StatusFound, checkTarget(target))
	}
}
