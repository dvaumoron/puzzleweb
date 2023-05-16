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
	"context"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const siteName = "Site"
const unknownUserKey = "ErrorUnknownUser"

type Site struct {
	logger         *otelzap.Logger
	tracer         trace.Tracer
	localesManager locale.Manager
	authService    adminservice.AuthService
	timeOut        time.Duration
	root           Page
	adders         []common.DataAdder
	HTMLRender     render.HTMLRender
}

func NewSite(configExtracter config.BaseConfigExtracter, localesManager locale.Manager, settingsManager *SettingsManager) *Site {
	adminConfig := configExtracter.ExtractAdminConfig()
	root := MakeStaticPage(configExtracter.GetTracer(), "root", adminservice.PublicGroupId, "index"+configExtracter.GetTemplatesExt())
	root.AddSubPage(newLoginPage(configExtracter.ExtractLoginConfig(), settingsManager))
	root.AddSubPage(newAdminPage(adminConfig))
	root.AddSubPage(newSettingsPage(config.MakeServiceConfig(configExtracter, settingsManager)))
	root.AddSubPage(newProfilePage(configExtracter.ExtractProfileConfig()))

	return &Site{
		logger: configExtracter.GetLogger(), localesManager: localesManager, authService: adminConfig.Service,
		timeOut: configExtracter.GetServiceTimeOut(), root: root,
	}
}

func (site *Site) AddPage(page Page) {
	site.root.AddSubPage(page)
}

func (site *Site) AddStaticPagesFromFolder(logger otelzap.LoggerWithCtx, groupId uint64, folderName string, templatesPath string, templateExt string) {
	site.root.AddStaticPagesFromFolder(logger, site.tracer, groupId, folderName, templatesPath, templateExt)
}

func (site *Site) GetPage(name string) (Page, bool) {
	return site.root.GetSubPage(name)
}

func (site *Site) AddDefaultData(adder common.DataAdder) {
	site.adders = append(site.adders, adder)
}

func (site *Site) manageTimeOut(c *gin.Context) {
	newCtx, cancel := context.WithTimeout(c.Request.Context(), site.timeOut)
	defer cancel()

	c.Request = c.Request.WithContext(newCtx)
	c.Next()
}

func (site *Site) initEngine(siteConfig config.SiteConfig) *gin.Engine {
	tracer := siteConfig.Tracer

	engine := gin.New()
	engine.Use(site.manageTimeOut, otelgin.Middleware(config.WebKey), gin.Recovery())

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

	engine.Use(makeSessionManager(siteConfig.ExtractSessionConfig()).manage, func(c *gin.Context) {
		c.Set(siteName, site)
	})

	if localesManager := site.localesManager; localesManager.GetMultipleLang() {
		engine.GET("/changeLang", common.CreateRedirect(tracer, "changeLangHandler", changeLangRedirecter))

		langPicturePaths := siteConfig.LangPicturePaths
		for _, lang := range localesManager.GetAllLang() {
			if langPicturePath, ok := langPicturePaths[lang]; ok {
				// allow modified time check (instead of always sending same data)
				engine.StaticFile("/langPicture/"+lang, langPicturePath)
			}
		}
	}

	site.root.Widget.LoadInto(engine)
	engine.NoRoute(common.CreateRedirectString(tracer, "noRouteHandler", siteConfig.Page404Url))
	return engine
}

// Launch the site server, when finished shutdown the TracerProvider
func (site *Site) Run(siteConfig config.SiteConfig) error {
	tracerProvider := siteConfig.TracerProvider
	tracer := siteConfig.Tracer
	logger := siteConfig.Logger
	defer func() {
		ctx := context.Background()
		if err := tracerProvider.Shutdown(ctx); err != nil {
			ctx, stopSpan := tracer.Start(ctx, "shutdown")
			logger.WarnContext(ctx, "Failed to shutdown trace provider", zap.Error(err))
			stopSpan.End()
		}
	}()
	return site.initEngine(siteConfig).Run(checkPort(siteConfig.Port))
}

type SiteAndConfig struct {
	Site   *Site
	Config config.SiteConfig
}

func Run(ginLogger *zap.Logger, sites ...SiteAndConfig) error {
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

func changeLangRedirecter(c *gin.Context) string {
	getSite(c).localesManager.SetLangCookie(c.Query(locale.LangName), c)
	return c.Query(common.RedirectName)
}

func checkPort(port string) string {
	if port[0] != ':' {
		port = ":" + port
	}
	return port
}

func BuildDefaultSite(serviceName string, version string) (*Site, *config.GlobalConfig, trace.Span) {
	globalConfig, initSpan := config.LoadDefault(serviceName, version)
	localesManager := locale.NewManager(globalConfig.ExtractLocalesConfig())
	settingsManager := NewSettingsManager(globalConfig.ExtractSettingsConfig())

	site := NewSite(globalConfig, localesManager, settingsManager)

	return site, globalConfig, initSpan
}

func (p Page) AddStaticPagesFromFolder(logger otelzap.LoggerWithCtx, tracer trace.Tracer, groupId uint64, folderName string, templatesPath string, templateExt string) {
	templatesPath, err := filepath.Abs(templatesPath)
	if err != nil {
		logger.Fatal("Wrong templatesPath", zap.Error(err))
	}

	inSize := len(templatesPath)
	var folderPathBuilder strings.Builder
	folderPathBuilder.WriteString(templatesPath)
	if last := inSize - 1; templatesPath[last] != '/' {
		folderPathBuilder.WriteByte('/')
		inSize++
	}
	folderPathBuilder.WriteString(folderName)
	folderSize := len(folderName) + 1

	extSize := len(templateExt)
	slashIndexName := "/index" + templateExt
	err = filepath.WalkDir(folderPathBuilder.String(), func(path string, d fs.DirEntry, err error) error {
		if err == nil {
			if innerPath := path[inSize:]; d.IsDir() {
				if len(innerPath) > folderSize {
					currentPage, name := p.extractSubPageFromPath(innerPath[folderSize:])
					currentPage.AddSubPage(MakeStaticPage(tracer, name, groupId, innerPath+slashIndexName))
				}
			} else if cut := len(innerPath) - extSize; innerPath[cut:] == templateExt {
				if currentPage, name := p.extractSubPageFromPath(innerPath[folderSize:cut]); name != "index" {
					currentPage.AddSubPage(MakeStaticPage(tracer, name, groupId, innerPath))
				}
			}
		}
		return err
	})

	if err != nil {
		logger.Fatal("Failed to load static pages", zap.Error(err))
	}
}
