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
	profileservice "github.com/dvaumoron/puzzleweb/profile/service"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const siteName = "Site"

type Site struct {
	logger             *zap.Logger
	localesManager     *locale.LocalesManager
	authService        adminservice.AuthService
	pictureService     profileservice.PictureService
	root               Page
	adders             []common.DataAdder
	Page404Url         string
	FaviconPath        string
	HTMLRender         render.HTMLRender
	MaxMultipartMemory int64
}

func NewSite(globalConfig *config.GlobalConfig, localesManager *locale.LocalesManager, settingsManager *SettingsManager) *Site {
	root := MakeStaticPage("root", adminservice.PublicGroupId, "index"+globalConfig.TemplatesExt)
	root.AddSubPage(newLoginPage(globalConfig.ExtractLoginConfig(), settingsManager))
	root.AddSubPage(newAdminPage(globalConfig.ExtractAdminConfig()))
	root.AddSubPage(newSettingsPage(config.CreateServiceExtConfig(globalConfig, settingsManager)))

	authConfig := globalConfig.ExtractAuthExtConfig()
	return &Site{
		logger: authConfig.Logger, localesManager: localesManager, authService: authConfig.Service,
		pictureService: globalConfig.ProfileService, root: root,
	}
}

func (site *Site) AddPage(page Page) {
	site.root.AddSubPage(page)
}

func (site *Site) AddDefaultData(adder common.DataAdder) {
	site.adders = append(site.adders, adder)
}

func (site *Site) initEngine(siteConfig config.SiteConfig) *gin.Engine {
	staticPath := siteConfig.StaticPath

	engine := gin.Default()

	if memorySize := site.MaxMultipartMemory; memorySize != 0 {
		engine.MaxMultipartMemory = memorySize
	}

	if htmlRender := site.HTMLRender; htmlRender == nil {
		siteConfig.Logger.Fatal("no HTMLRender initialized")
	} else {
		engine.HTMLRender = htmlRender
	}

	engine.Static("/static", staticPath)

	favicon := "/favicon.ico"
	faviconPath := site.FaviconPath
	if faviconPath == "" {
		faviconPath = favicon
	}
	engine.StaticFile(favicon, staticPath+faviconPath)

	engine.Use(makeSessionManager(siteConfig.ExtractSessionConfig()).Manage, func(c *gin.Context) {
		c.Set(siteName, site)
	})

	engine.GET("/profilePic/:UserId", site.profilePicHandler)

	if site.localesManager.MultipleLang {
		engine.GET("/changeLang", changeLangHandler)
	}

	site.root.Widget.LoadInto(engine)
	engine.NoRoute(common.CreateRedirectString(site.Page404Url))
	return engine
}

func (site *Site) Run(siteConfig config.SiteConfig) error {
	return site.initEngine(siteConfig).Run(checkPort(siteConfig.Port))
}

type SiteAndConfig struct {
	Site   *Site
	Config config.SiteConfig
}

func Run(sites ...SiteAndConfig) error {
	var g errgroup.Group
	for _, siteAndConfig := range sites {
		port := checkPort(siteAndConfig.Config.Port)
		handler := siteAndConfig.Site.initEngine(siteAndConfig.Config).Handler()
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
	c.Data(http.StatusOK, http.DetectContentType(data), data)
}

var changeLangHandler = common.CreateRedirect(func(c *gin.Context) string {
	getSite(c).localesManager.SetLangCookie(c.Query(locale.LangName), c)
	return c.Query(common.RedirectName)
})

func checkPort(port string) string {
	if port[0] != ':' {
		port = ":" + port
	}
	return port
}
