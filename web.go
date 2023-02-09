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

	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/dvaumoron/puzzleweb/profile/service"
	"github.com/dvaumoron/puzzleweb/session"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const siteName = "Site"

type Site struct {
	engine         *gin.Engine
	logger         *zap.Logger
	staticPath     string
	localesManager *locale.LocalesManager
	pictureService service.PictureService
	root           Page
	Page404Url     string
	adders         []common.DataAdder
	initialized    bool
	FaviconPath    string
}

func NewSite(staticPath string, authConfig config.BasicConfig[adminservice.AuthService], pictureService service.PictureService, sessionManager session.SessionManager, localesManager *locale.LocalesManager, args ...string) *Site {
	logger := authConfig.Logger

	size := len(args)
	rootTmpl := "index.html"
	if size != 0 && args[0] != "" {
		rootTmpl = args[0]
	}
	if size > 1 {
		logger.Info("CreateSite should be called with at most 1 argument.")
	}

	engine := gin.Default()

	engine.Static("/static", staticPath)

	site := &Site{
		engine: engine, logger: logger, staticPath: staticPath,
		localesManager: localesManager, pictureService: pictureService,
		root: MakeStaticPage("root", authConfig, adminservice.PublicGroupId, rootTmpl),
	}

	engine.Use(sessionManager.Manage, func(c *gin.Context) {
		c.Set(siteName, site)
	})

	engine.GET("/profilePic/:UserId", site.profilePicHandler)

	return site
}

func (site *Site) AddPage(page Page) {
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

func (site *Site) initEngine(localesPath string) *gin.Engine {
	engine := site.engine
	if !site.initialized {
		site.localesManager.InitMessages(localesPath)

		if engine.HTMLRender == nil {
			site.logger.Fatal("no HTMLRender initialized")
		}

		favicon := "/favicon.ico"
		faviconPath := site.FaviconPath
		if faviconPath == "" {
			faviconPath = favicon
		}
		engine.StaticFile(favicon, site.staticPath+faviconPath)
		site.root.Widget.LoadInto(engine)
		if site.localesManager.MultipleLang {
			engine.GET("/changeLang", changeLangHandler)
		}
		engine.NoRoute(common.CreateRedirectString(site.Page404Url))
		site.initialized = true
	}
	return engine
}

func (site *Site) Run(localesPath string, port string) error {
	return site.initEngine(localesPath).Run(checkPort(port))
}

type SiteConfig struct {
	Site *Site
	Port string
}

func Run(localesPath string, sites ...SiteConfig) error {
	var g errgroup.Group
	for _, siteConfig := range sites {
		port := checkPort(siteConfig.Port)
		handler := siteConfig.Site.initEngine(localesPath).Handler()
		g.Go(func() error {
			server := &http.Server{Addr: port, Handler: handler}
			return server.ListenAndServe()
		})
	}
	return g.Wait()
}

func (site *Site) profilePicHandler(c *gin.Context) {
	userId := common.GetRequestedUserId(site.logger, c)
	if userId == 0 {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	data, err := site.pictureService.GetPicture(userId)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Data(http.StatusFound, http.DetectContentType(data), data)
}

var changeLangHandler = common.CreateRedirect(func(c *gin.Context) string {
	getSite(c).localesManager.SetLangCookie(c, c.Query(locale.LangName))
	return c.Query(common.RedirectName)
})

func checkPort(port string) string {
	if port[0] != ':' {
		port = ":" + port
	}
	return port
}
