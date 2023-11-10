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

package globalconfig

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dvaumoron/puzzlesaltclient"
	"github.com/dvaumoron/puzzletelemetry"
	adminclient "github.com/dvaumoron/puzzleweb/admin/client"
	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	blogclient "github.com/dvaumoron/puzzleweb/blog/client"
	"github.com/dvaumoron/puzzleweb/common/config"
	"github.com/dvaumoron/puzzleweb/common/config/parser"
	"github.com/dvaumoron/puzzleweb/common/log"
	forumclient "github.com/dvaumoron/puzzleweb/forum/client"
	forumservice "github.com/dvaumoron/puzzleweb/forum/service"
	loginclient "github.com/dvaumoron/puzzleweb/login/client"
	loginservice "github.com/dvaumoron/puzzleweb/login/service"
	markdownclient "github.com/dvaumoron/puzzleweb/markdown/client"
	markdownservice "github.com/dvaumoron/puzzleweb/markdown/service"
	strengthclient "github.com/dvaumoron/puzzleweb/passwordstrength/client"
	profileclient "github.com/dvaumoron/puzzleweb/profile/client"
	profileservice "github.com/dvaumoron/puzzleweb/profile/service"
	widgetclient "github.com/dvaumoron/puzzleweb/remotewidget/client"
	sessionclient "github.com/dvaumoron/puzzleweb/session/client"
	sessionservice "github.com/dvaumoron/puzzleweb/session/service"
	templateclient "github.com/dvaumoron/puzzleweb/templates/client"
	templateservice "github.com/dvaumoron/puzzleweb/templates/service"
	wikiclient "github.com/dvaumoron/puzzleweb/wiki/client"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultName           = "default"
	defaultSessionTimeOut = 1200
	defaultServiceTimeOut = 5 * time.Second
)

type loggerWrapper struct {
	logger *otelzap.Logger
}

func (lg loggerWrapper) Logger(ctx context.Context) log.Logger {
	return lg.logger.Ctx(ctx)
}

type GlobalConfig struct {
	Domain string
	Port   string

	AllLang            []string
	SessionTimeOut     int
	ServiceTimeOut     time.Duration
	MaxMultipartMemory int64
	DateFormat         string
	PageSize           uint64
	ExtractSize        uint64
	FeedFormat         string
	FeedSize           uint64

	StaticFileSystem http.FileSystem
	FaviconPath      string
	Page404Url       string

	InitCtx          context.Context
	Logger           log.Logger // for init phase (have the context)
	LoggerGetter     log.LoggerGetter
	TracerProvider   *sdktrace.TracerProvider
	Tracer           trace.Tracer
	LangPicturePaths map[string]string

	DialOptions     []grpc.DialOption
	SessionService  sessionservice.SessionService
	TemplateService templateservice.TemplateService
	SaltService     loginservice.SaltService
	SettingsService sessionservice.SessionService
	LoginService    loginservice.FullLoginService
	RightClient     adminclient.RightClient
	ProfileService  profileservice.AdvancedProfileService

	// lazy service
	MarkdownServiceAddr string
	MarkdownService     markdownservice.MarkdownService

	// lazy & only adresses (instance need specific data)
	WikiServiceAddr  string
	ForumServiceAddr string
	BlogServiceAddr  string
}

func Init(serviceName string, version string, parsedConfig parser.ParsedConfig, err error) (*GlobalConfig, trace.Span) {
	logger, tp := puzzletelemetry.Init(serviceName, version)
	tracer := tp.Tracer(config.WebKey)

	initCtx, initSpan := tracer.Start(context.Background(), "initialization")
	ctxLogger := logger.Ctx(initCtx)
	loggerGetter := loggerWrapper{logger: logger}
	if err != nil {
		ctxLogger.Fatal("Failed to read configuration file", zap.Error(err))
	}

	var serviceTimeOut time.Duration

	domain := retrieveWithDefault(ctxLogger, "domain", parsedConfig.Domain, "localhost")
	port := retrieveWithDefault(ctxLogger, "port", parsedConfig.Port, "8080")

	sessionTimeOut := parsedConfig.SessionTimeOut
	if sessionTimeOut == 0 {
		ctxLogger.Info("sessionTimeOut empty, using default", zap.Int(defaultName, defaultSessionTimeOut))
		sessionTimeOut = defaultSessionTimeOut
	}

	serviceTimeOutStr := parsedConfig.ServiceTimeOut
	if serviceTimeOutStr == "" {
		ctxLogger.Info("serviceTimeOut empty, using default", zap.Duration(defaultName, defaultServiceTimeOut))
		serviceTimeOut = defaultServiceTimeOut
	} else if timeOut, _ := strconv.ParseInt(serviceTimeOutStr, 10, 64); timeOut == 0 {
		ctxLogger.Warn("Failed to parse serviceTimeOut, using default", zap.Duration(defaultName, defaultServiceTimeOut))
		serviceTimeOut = defaultServiceTimeOut
	} else {
		serviceTimeOut = time.Duration(timeOut) * time.Second
	}

	maxMultipartMemory := parsedConfig.MaxMultipartMemory
	if maxMultipartMemory == 0 {
		ctxLogger.Warn("maxMultipartMemory empty, using gin default")
	}

	dateFormat := retrieveWithDefault(ctxLogger, "dateFormat", parsedConfig.DateFormat, "2/1/2006 15:04:05")
	pageSize := retrieveUintWithDefault(ctxLogger, "pageSize", parsedConfig.PageSize, 20)
	extractSize := retrieveUintWithDefault(ctxLogger, "extractSize", parsedConfig.ExtractSize, 200)
	feedFormat := retrieveWithDefault(ctxLogger, "feedFormat", parsedConfig.FeedFormat, "atom")
	feedSize := retrieveUintWithDefault(ctxLogger, "feedSize", parsedConfig.FeedSize, 100)

	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	}

	sessionService := sessionclient.New(parsedConfig.SessionServiceAddr, dialOptions)
	templateService := templateclient.New(parsedConfig.TemplateServiceAddr, dialOptions, loggerGetter)
	settingsService := sessionclient.New(parsedConfig.SettingsServiceAddr, dialOptions)
	strengthService := strengthclient.New(parsedConfig.PasswordStrengthServiceAddr, dialOptions)
	saltService := puzzlesaltclient.Make(parsedConfig.SaltServiceAddr, dialOptions)
	loginService := loginclient.New(parsedConfig.LoginServiceAddr, dialOptions, dateFormat, saltService, strengthService)
	rightClient := adminclient.Make(parsedConfig.RightServiceAddr, dialOptions, logger)

	staticPath := retrievePath(ctxLogger, "staticPath", parsedConfig.StaticPath, "static")
	faviconPath := retrieveWithDefault(ctxLogger, "faviconPath", parsedConfig.FaviconPath, config.DefaultFavicon)

	defaultPicturePath := retrieveWithDefault(
		ctxLogger, "profileDefaultPicturePath", parsedConfig.ProfileDefaultPicturePath, staticPath+"/images/unknownuser.png",
	)
	defaultPicture, err := os.ReadFile(defaultPicturePath)
	if err != nil {
		ctxLogger.Fatal("Can not read", zap.String("filepath", defaultPicturePath), zap.Error(err))
	}

	locales := parsedConfig.Locales
	langNumber := len(locales)
	allLang := make([]string, 0, langNumber)
	langPicturePaths := make(map[string]string, langNumber)
	for _, locale := range locales {
		allLang = append(allLang, locale.Lang)
		langPicturePaths[locale.Lang] = locale.PicturePath
	}
	ctxLogger.Info("Declared locales", zap.Strings("locales", allLang))

	// if not setted in configuration, profile are public
	profileGroupId := retrieveUintWithDefault(ctxLogger, "profileGroupId", parsedConfig.ProfileGroupId, adminservice.PublicGroupId)
	profileService := profileclient.New(
		parsedConfig.ProfileServiceAddr, dialOptions, profileGroupId, defaultPicture, loginService, rightClient, loggerGetter,
	)

	globalConfig := &GlobalConfig{
		Domain: domain, Port: port, AllLang: allLang, SessionTimeOut: sessionTimeOut, ServiceTimeOut: serviceTimeOut,
		MaxMultipartMemory: maxMultipartMemory, DateFormat: dateFormat, PageSize: pageSize, ExtractSize: extractSize,
		FeedFormat: feedFormat, FeedSize: feedSize,

		StaticFileSystem: http.FS(os.DirFS(staticPath)),
		FaviconPath:      faviconPath,
		Page404Url:       parsedConfig.Page404Url,

		InitCtx:        initCtx,
		Logger:         ctxLogger,
		LoggerGetter:   loggerGetter,
		TracerProvider: tp,
		Tracer:         tracer,

		LangPicturePaths: langPicturePaths,
		DialOptions:      dialOptions,
		SessionService:   sessionService,
		TemplateService:  templateService,
		SaltService:      saltService,
		SettingsService:  settingsService,
		LoginService:     loginService,
		RightClient:      rightClient,
		ProfileService:   profileService,

		ForumServiceAddr:    parsedConfig.ForumServiceAddr,
		MarkdownServiceAddr: parsedConfig.MarkdownServiceAddr,
		BlogServiceAddr:     parsedConfig.BlogServiceAddr,
		WikiServiceAddr:     parsedConfig.WikiServiceAddr,
	}

	return globalConfig, initSpan
}

func (c *GlobalConfig) loadMarkdown() bool {
	if c.MarkdownService == nil {
		if !require(c.Logger, "markdownServiceAddr", c.MarkdownServiceAddr) {
			return false
		}
		c.MarkdownService = markdownclient.New(c.MarkdownServiceAddr, c.DialOptions)
	}
	return true
}

func (c *GlobalConfig) loadWiki() bool {
	return c.loadMarkdown() && require(c.Logger, "wikiServiceAddr", c.WikiServiceAddr)
}

func (c *GlobalConfig) loadForum() bool {
	return require(c.Logger, "forumServiceAddr", c.ForumServiceAddr)
}

func (c *GlobalConfig) loadBlog() bool {
	return c.loadForum() && c.loadMarkdown() && require(c.Logger, "blogServiceAddr", c.BlogServiceAddr)
}

func (c *GlobalConfig) GetLogger() log.Logger {
	return c.Logger
}

func (c *GlobalConfig) GetLoggerGetter() log.LoggerGetter {
	return c.LoggerGetter
}

func (c *GlobalConfig) GetServiceTimeOut() time.Duration {
	return c.ServiceTimeOut
}

func (c *GlobalConfig) ExtractAuthConfig() config.AuthConfig {
	return config.MakeServiceConfig[adminservice.AuthService](c, c.RightClient)
}

func (c *GlobalConfig) ExtractLocalesConfig() config.LocalesConfig {
	return config.LocalesConfig{
		Logger: c.Logger, LoggerGetter: c.LoggerGetter, Domain: c.Domain, SessionTimeOut: c.SessionTimeOut, AllLang: c.AllLang,
	}
}

func (c *GlobalConfig) ExtractSiteConfig() config.SiteConfig {
	return config.SiteConfig{
		ServiceConfig: config.MakeServiceConfig(c, c.SessionService), TemplateService: c.TemplateService,
		Domain: c.Domain, Port: c.Port, SessionTimeOut: c.SessionTimeOut, MaxMultipartMemory: c.MaxMultipartMemory,
		StaticFileSystem: c.StaticFileSystem, FaviconPath: c.FaviconPath, LangPicturePaths: c.LangPicturePaths,
		Page404Url: c.Page404Url,
	}
}

func (c *GlobalConfig) ExtractLoginConfig() config.LoginConfig {
	return config.MakeServiceConfig[loginservice.LoginService](c, c.LoginService)
}

func (c *GlobalConfig) ExtractAdminConfig() config.AdminConfig {
	return config.AdminConfig{
		ServiceConfig: config.MakeServiceConfig[adminservice.AdminService](c, c.RightClient),
		UserService:   c.LoginService, ProfileService: c.ProfileService, PageSize: c.PageSize,
	}
}

func (c *GlobalConfig) ExtractProfileConfig() config.ProfileConfig {
	return config.ProfileConfig{
		ServiceConfig: config.MakeServiceConfig(c, c.ProfileService),
		AdminService:  c.RightClient, LoginService: c.LoginService,
	}
}

func (c *GlobalConfig) ExtractSettingsConfig() config.SettingsConfig {
	return config.MakeServiceConfig(c, c.SettingsService)
}

func (c *GlobalConfig) MakeWikiConfig(widgetConfig parser.WidgetConfig) (config.WikiConfig, bool) {
	return config.WikiConfig{
		ServiceConfig: config.MakeServiceConfig(c, wikiclient.New(
			c.WikiServiceAddr, c.DialOptions, widgetConfig.ObjectId, widgetConfig.GroupId, c.DateFormat,
			c.RightClient, c.ProfileService, c.LoggerGetter,
		)),
		MarkdownService: c.MarkdownService, Args: widgetConfig.Templates,
	}, c.loadWiki()
}

func (c *GlobalConfig) MakeForumConfig(widgetConfig parser.WidgetConfig) (config.ForumConfig, bool) {
	return config.ForumConfig{
		ServiceConfig: config.MakeServiceConfig[forumservice.ForumService](c, forumclient.New(
			c.ForumServiceAddr, c.DialOptions, widgetConfig.ObjectId, widgetConfig.GroupId, c.DateFormat,
			c.RightClient, c.ProfileService, c.LoggerGetter,
		)),
		PageSize: c.PageSize, Args: widgetConfig.Templates,
	}, c.loadForum()
}

func (c *GlobalConfig) MakeBlogConfig(widgetConfig parser.WidgetConfig) (config.BlogConfig, bool) {
	return config.BlogConfig{
		ServiceConfig: config.MakeServiceConfig(c, blogclient.New(
			c.BlogServiceAddr, c.DialOptions, widgetConfig.ObjectId, widgetConfig.GroupId, c.DateFormat,
			c.RightClient, c.ProfileService,
		)),
		MarkdownService: c.MarkdownService, CommentService: forumclient.New(
			c.ForumServiceAddr, c.DialOptions, widgetConfig.ObjectId, widgetConfig.GroupId, c.DateFormat,
			c.RightClient, c.ProfileService, c.LoggerGetter,
		),
		Domain: c.Domain, Port: c.Port, DateFormat: c.DateFormat, PageSize: c.PageSize, ExtractSize: c.ExtractSize,
		FeedFormat: c.FeedFormat, FeedSize: c.FeedSize, Args: widgetConfig.Templates,
	}, c.loadBlog()
}

func (c *GlobalConfig) MakeWidgetConfig(widgetConfig parser.WidgetConfig) (config.RemoteWidgetConfig, bool) {
	widgetName, remoteKind := strings.CutPrefix(widgetConfig.Kind, "remote/")
	return config.MakeServiceConfig(c, widgetclient.New(
		widgetConfig.ServiceAddr, c.DialOptions, c.LoggerGetter, widgetName, widgetConfig.ObjectId, widgetConfig.GroupId,
	)), remoteKind
}

func retrieveWithDefault(logger log.Logger, name string, value string, defaultValue string) string {
	if value == "" {
		logger.Info(name+" empty, using default", zap.String(defaultName, defaultValue))
		return defaultValue
	}
	return value
}

func retrieveUintWithDefault(logger log.Logger, name string, value uint64, defaultValue uint64) uint64 {
	if value == 0 {
		logger.Info(name+" empty, using default", zap.Uint64(defaultName, defaultValue))
		return defaultValue
	}
	return value
}

func retrievePath(logger log.Logger, name string, path string, defaultPath string) string {
	path = retrieveWithDefault(logger, name, path, defaultPath)
	if last := len(path) - 1; path[last] == '/' {
		path = path[:last]
	}
	return path
}

func require(logger log.Logger, name string, value string) bool {
	if value == "" {
		logger.Error(name + " is required")
		return false
	}
	return true
}
