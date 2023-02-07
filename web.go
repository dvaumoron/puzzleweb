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

	rightclient "github.com/dvaumoron/puzzleweb/admin/client"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/dvaumoron/puzzleweb/log"
	profileclient "github.com/dvaumoron/puzzleweb/profile/client"
	"github.com/dvaumoron/puzzleweb/session"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"golang.org/x/sync/errgroup"
)

const siteName = "Site"

type Site struct {
	engine      *gin.Engine
	root        *Page
	Page404Url  string
	adders      []common.DataAdder
	initialized bool
	FaviconPath string
}

func NewSite(args ...string) *Site {
	size := len(args)
	rootTmpl := "index.html"
	if size != 0 && args[0] != "" {
		rootTmpl = args[0]
	}
	if size > 1 {
		log.Logger.Info("CreateSite should be called with 0 or 1 argument.")
	}

	engine := gin.Default()

	engine.Static("/static", config.Shared.StaticPath)
	engine.GET("/profilePic/:UserId", profilePicHandler)

	site := &Site{
		engine: engine,
		root:   NewStaticPage("root", rightclient.PublicGroupId, rootTmpl),
	}

	engine.Use(session.Manage, func(c *gin.Context) {
		c.Set(siteName, site)
	})

	return site
}

func (site *Site) AddPage(page *Page) {
	site.root.AddSubPage(page)
}

func (site *Site) AddDefaultData(adder common.DataAdder) {
	site.adders = append(site.adders, adder)
}

func (site *Site) SetHTMLRender(r render.HTMLRender) {
	site.engine.HTMLRender = r
}

func (site *Site) SetMaxMultipartMemory(memorySize int64) {
	site.engine.MaxMultipartMemory = memorySize
}

func (site *Site) initEngine() *gin.Engine {
	engine := site.engine
	if !site.initialized {
		if engine.HTMLRender == nil {
			engine.HTMLRender = loadTemplates()
		}

		favicon := "/favicon.ico"
		faviconPath := site.FaviconPath
		if faviconPath == "" {
			faviconPath = favicon
		}
		engine.StaticFile(favicon, config.Shared.StaticPath+faviconPath)
		site.root.Widget.LoadInto(engine)
		if len(locale.AllLang) != 1 {
			engine.GET("/changeLang", changeLangHandler)
		}
		engine.NoRoute(common.CreateRedirectString(site.Page404Url))
		site.initialized = true
	}
	return engine
}

func (site *Site) Run() error {
	locale.InitMessages()
	return site.initEngine().Run(checkPort(config.Shared.Port))
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

func profilePicHandler(c *gin.Context) {
	userId, err := common.GetRequestedUserId(c)
	if err != nil {
		c.AbortWithError(http.StatusNotFound, err)
		return
	}

	data, err := profileclient.GetPicture(userId)
	if err != nil {
		c.AbortWithError(http.StatusNotFound, err)
		return
	}
	c.Data(http.StatusFound, http.DetectContentType(data), data)
}

var changeLangHandler = common.CreateRedirect(func(c *gin.Context) string {
	locale.SetLangCookie(c, c.Query(locale.LangName))
	return c.Query(common.RedirectName)
})

func checkPort(port string) string {
	if port[0] != ':' {
		port = ":" + port
	}
	return port
}
