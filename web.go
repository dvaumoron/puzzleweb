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
	"golang.org/x/sync/errgroup"
)

type Site struct {
	engine      *gin.Engine
	root        *Page
	Page404Url  string
	adders      []DataAdder
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

	engine.HTMLRender = templatesRender

	engine.Static("/static", config.StaticPath)

	site := &Site{
		engine: engine,
		root:   NewStaticPage("root", rootTmpl),
		adders: make([]DataAdder, 0),
	}

	engine.Use(session.Manage, func(c *gin.Context) {
		c.Set(siteName, site)
	})

	return site
}

func (site *Site) AddPage(page *Page) {
	site.root.AddSubPage(page)
}

func (site *Site) AddDefaultData(adder DataAdder) {
	site.adders = append(site.adders, adder)
}

func (site *Site) initEngine() *gin.Engine {
	engine := site.engine
	if !site.initialized {
		favicon := "/favicon.ico"
		faviconPath := site.FaviconPath
		if faviconPath == "" {
			faviconPath = favicon
		}
		engine.StaticFile(favicon, config.StaticPath+faviconPath)
		site.root.Widget.LoadInto(engine)
		engine.GET("/changeLang", CreateRedirect(func(c *gin.Context) string {
			locale.SetLangCookie(c, c.Query(locale.LangName))
			return c.Query(RedirectName)
		}))
		engine.NoRoute(CreateRedirectString(site.Page404Url))
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
	Site *Site
	Port string
}

func Run(sites ...SiteConfig) error {
	locale.InitMessages()
	var g errgroup.Group
	for _, siteConfig := range sites {
		port := checkPort(siteConfig.Port)
		handler := siteConfig.Site.initEngine().Handler()
		g.Go(func() error {
			server := &http.Server{Addr: port, Handler: handler}
			return server.ListenAndServe()
		})
	}
	return g.Wait()
}

type Redirecter func(*gin.Context) string

func checkTarget(target string) string {
	if target == "" {
		target = "/"
	}
	return target
}

func CreateRedirect(redirecter Redirecter) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Redirect(http.StatusFound, checkTarget(redirecter(c)))
	}
}

func CreateRedirectString(target string) gin.HandlerFunc {
	target = checkTarget(target)
	return func(c *gin.Context) {
		c.Redirect(http.StatusFound, target)
	}
}
