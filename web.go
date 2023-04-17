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
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const siteName = "Site"
const unknownUserKey = "ErrorUnknownUser"

type Site struct {
	logger         *zap.Logger
	localesManager locale.Manager
	authService    adminservice.AuthService
	root           Page
	adders         []common.DataAdder
	HTMLRender     render.HTMLRender
}

func NewSite(configExtracter config.BaseConfigExtracter, localesManager locale.Manager, settingsManager *SettingsManager) *Site {
	adminConfig := configExtracter.ExtractAdminConfig()
	root := MakeStaticPage("root", adminservice.PublicGroupId, "index"+configExtracter.GetTemplatesExt())
	root.AddSubPage(newLoginPage(configExtracter.ExtractLoginConfig(), settingsManager))
	root.AddSubPage(newAdminPage(adminConfig))
	root.AddSubPage(newSettingsPage(config.MakeServiceConfig(configExtracter, settingsManager)))
	root.AddSubPage(newProfilePage(configExtracter.ExtractProfileConfig()))

	return &Site{
		logger: configExtracter.GetLogger(), localesManager: localesManager, authService: adminConfig.Service, root: root,
	}
}

func (site *Site) AddPage(page Page) {
	site.root.AddSubPage(page)
}

func (site *Site) AddDefaultData(adder common.DataAdder) {
	site.adders = append(site.adders, adder)
}

func (site *Site) initEngine(siteConfig config.SiteConfig) *gin.Engine {
	engine := gin.Default()

	if memorySize := siteConfig.MaxMultipartMemory; memorySize != 0 {
		engine.MaxMultipartMemory = memorySize
	}

	if htmlRender := site.HTMLRender; htmlRender == nil {
		site.logger.Fatal("no HTMLRender initialized")
	} else {
		engine.HTMLRender = htmlRender
	}

	engine.Static("/static", siteConfig.StaticPath)
	engine.StaticFile(config.DefaultFavicon, siteConfig.FaviconPath)

	engine.Use(makeSessionManager(siteConfig.ExtractSessionConfig()).Manage, func(c *gin.Context) {
		c.Set(siteName, site)
	})

	if localesManager := site.localesManager; localesManager.GetMultipleLang() {
		engine.GET("/changeLang", changeLangHandler)

		langPicturePaths := siteConfig.LangPicturePaths
		for _, lang := range localesManager.GetAllLang() {
			if langPicturePath, ok := langPicturePaths[lang]; ok {
				// allow modified time check (instead of always sending same data)
				engine.StaticFile("/langPicture/"+lang, langPicturePath)
			}
		}
	}

	site.root.Widget.LoadInto(engine)
	engine.NoRoute(common.CreateRedirectString(siteConfig.Page404Url))
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

func BuildDefaultSite() (*Site, *config.GlobalConfig) {
	globalConfig := config.LoadDefault()
	localesManager := locale.NewManager(globalConfig.ExtractLocalesConfig())
	settingsManager := NewSettingsManager(globalConfig.ExtractSettingsConfig())

	site := NewSite(globalConfig, localesManager, settingsManager)

	return site, globalConfig
}

func (site *Site) AddStaticPagesFromFolder(groupId uint64, folderPath string, templateExt string) {
	folderPath, err := filepath.Abs(folderPath)
	if err != nil {
		site.logger.Fatal("Wrong folderPath", zap.Error(err))
	}
	if last := len(folderPath) - 1; folderPath[last] != '/' {
		folderPath += "/"
	}

	inSize := len(folderPath)
	extSize := len(templateExt)
	indexName := "index" + templateExt
	slashIndexName := "/" + indexName
	err = filepath.WalkDir(folderPath, func(path string, d fs.DirEntry, err error) error {
		if err == nil {
			innerPath := path[inSize:]
			if d.IsDir() {
				currentPage, name := site.extractPageFromPath(innerPath)
				currentPage.AddSubPage(MakeStaticPage(name, groupId, innerPath+slashIndexName))
			} else {
				cut := len(innerPath) - extSize
				if innerPath[cut:] == templateExt {
					currentPage, name := site.extractPageFromPath(innerPath)
					if name != indexName {
						cut = len(name) - extSize
						currentPage.AddSubPage(MakeStaticPage(name[:cut], groupId, innerPath))
					}
				}
			}
		}
		return err
	})

	if err != nil {
		site.logger.Fatal("Failed to load static pages", zap.Error(err))
	}
}

func (site *Site) extractPageFromPath(path string) (Page, string) {
	current := site.root
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
