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

package config

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/dvaumoron/puzzlesaltclient"
	"github.com/dvaumoron/puzzletelemetry"
	adminclient "github.com/dvaumoron/puzzleweb/admin/client"
	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	blogclient "github.com/dvaumoron/puzzleweb/blog/client"
	"github.com/dvaumoron/puzzleweb/config/parser"
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
	widgetservice "github.com/dvaumoron/puzzleweb/remotewidget/service"
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

const WebKey = "puzzleWeb"

const defaultName = "default"
const defaultSessionTimeOut = 1200
const defaultServiceTimeOut = 5 * time.Second

const DefaultFavicon = "/favicon.ico"

type AuthConfig = ServiceConfig[adminservice.AuthService]
type LoginConfig = ServiceConfig[loginservice.LoginService]
type SettingsConfig = ServiceConfig[sessionservice.SessionService]
type TemplateConfig = ServiceConfig[templateservice.TemplateService]
type WidgetConfig = ServiceConfig[widgetservice.WidgetService]

type BaseConfigExtracter interface {
	BaseConfig
	GetServiceTimeOut() time.Duration
	ExtractLoginConfig() LoginConfig
	ExtractAdminConfig() AdminConfig
	ExtractProfileConfig() ProfileConfig
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

	StaticPath  string
	FaviconPath string
	Page404Url  string

	CtxLogger        otelzap.LoggerWithCtx
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
	tracer := tp.Tracer(WebKey)

	ctx, initSpan := tracer.Start(context.Background(), "initialization")
	ctxLogger := logger.Ctx(ctx)
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
	templateService := templateclient.New(parsedConfig.TemplateServiceAddr, dialOptions)
	settingsService := sessionclient.New(parsedConfig.SettingsServiceAddr, dialOptions)
	strengthService := strengthclient.New(parsedConfig.PasswordStrengthServiceAddr, dialOptions)
	saltService := puzzlesaltclient.Make(parsedConfig.SaltServiceAddr, dialOptions)
	loginService := loginclient.New(parsedConfig.LoginServiceAddr, dialOptions, dateFormat, saltService, strengthService)
	rightClient := adminclient.Make(parsedConfig.RightServiceAddr, dialOptions)

	staticPath := retrievePath(ctxLogger, "staticPath", parsedConfig.StaticPath, "static")
	augmentedStaticPath := staticPath + "/"
	faviconPath := parsedConfig.FaviconPath
	if faviconPath == "" {
		faviconPath = staticPath + DefaultFavicon
		ctxLogger.Info("faviconPath empty, using default", zap.String(defaultName, faviconPath))
	} else if faviconPath[0] != '/' {
		// user should use absolute path or path relative to staticPath
		faviconPath = augmentedStaticPath + faviconPath
	}

	defaultPicturePath := retrieveWithDefault(ctxLogger, "profileDefaultPicturePath", parsedConfig.ProfileDefaultPicturePath, staticPath+"/images/unknownuser.png")
	defaultPicture, err := os.ReadFile(defaultPicturePath)
	if err != nil {
		ctxLogger.Fatal("Can not read", zap.String("filepath", defaultPicturePath), zap.Error(err))
	}

	allLang := parsedConfig.AllLang
	ctxLogger.Info("Declared locales", zap.Strings("locales", allLang))

	langNumber := len(allLang)
	langPicturePaths := make(map[string]string, langNumber)
	confLangPicturePaths := parsedConfig.LocalePicturePaths
	confLangPicturePathsLen := len(confLangPicturePaths)
	for index, lang := range allLang {
		if index >= confLangPicturePathsLen {
			ctxLogger.Warn("localePicturePaths have less element than availableLocales")
			break
		}

		langPicturePath := confLangPicturePaths[index]
		if langPicturePath == "" {
			// skip not configured picture
			continue
		}
		// user should use absolute path or path relative to staticPath
		if langPicturePath[0] != '/' {
			langPicturePath = augmentedStaticPath + langPicturePath
		}
		langPicturePaths[lang] = langPicturePath
	}

	// if not setted in configuration, profile are public
	profileGroupId := retrieveUintWithDefault(ctxLogger, "profileGroupId", parsedConfig.ProfileGroupId, adminservice.PublicGroupId)
	profileService := profileclient.New(
		parsedConfig.ProfileServiceAddr, dialOptions, profileGroupId, loginService, rightClient, defaultPicture,
	)

	globalConfig := &GlobalConfig{
		Domain: domain, Port: port, AllLang: allLang, SessionTimeOut: sessionTimeOut, ServiceTimeOut: serviceTimeOut,
		MaxMultipartMemory: maxMultipartMemory, DateFormat: dateFormat, PageSize: pageSize, ExtractSize: extractSize,
		FeedFormat: feedFormat, FeedSize: feedSize,

		StaticPath:  staticPath,
		FaviconPath: faviconPath,
		Page404Url:  parsedConfig.Page404Url,

		CtxLogger:      ctxLogger,
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

func (c *GlobalConfig) loadMarkdown() {
	if c.MarkdownService == nil {
		require(c.CtxLogger, "markdownServiceAddr", c.MarkdownServiceAddr)
		c.MarkdownService = markdownclient.New(c.MarkdownServiceAddr, c.DialOptions)
	}
}

func (c *GlobalConfig) loadWiki() {
	c.loadMarkdown()
	require(c.CtxLogger, "wikiServiceAddr", c.WikiServiceAddr)
}

func (c *GlobalConfig) loadForum() {
	require(c.CtxLogger, "forumServiceAddr", c.ForumServiceAddr)
}

func (c *GlobalConfig) loadBlog() {
	c.loadForum()
	c.loadMarkdown()
	require(c.CtxLogger, "blogServiceAddr", c.BlogServiceAddr)
}

func (c *GlobalConfig) GetLogger() *otelzap.Logger {
	return c.CtxLogger.Logger()
}

func (c *GlobalConfig) GetTracer() trace.Tracer {
	return c.Tracer
}

func (c *GlobalConfig) GetServiceTimeOut() time.Duration {
	return c.ServiceTimeOut
}

func (c *GlobalConfig) ExtractAuthConfig() AuthConfig {
	return MakeServiceConfig[adminservice.AuthService](c, c.RightClient)
}

func (c *GlobalConfig) ExtractLocalesConfig() LocalesConfig {
	return LocalesConfig{Logger: c.GetLogger(), Domain: c.Domain, SessionTimeOut: c.SessionTimeOut, AllLang: c.AllLang}
}

func (c *GlobalConfig) ExtractSiteConfig() SiteConfig {
	return SiteConfig{
		ServiceConfig: MakeServiceConfig(c, c.SessionService), TemplateService: c.TemplateService,
		TracerProvider: c.TracerProvider, Domain: c.Domain, Port: c.Port, SessionTimeOut: c.SessionTimeOut,
		MaxMultipartMemory: c.MaxMultipartMemory, StaticPath: c.StaticPath, FaviconPath: c.FaviconPath,
		LangPicturePaths: c.LangPicturePaths, Page404Url: c.Page404Url,
	}
}

func (c *GlobalConfig) ExtractLoginConfig() LoginConfig {
	return MakeServiceConfig[loginservice.LoginService](c, c.LoginService)
}

func (c *GlobalConfig) ExtractAdminConfig() AdminConfig {
	return AdminConfig{
		ServiceConfig: MakeServiceConfig[adminservice.AdminService](c, c.RightClient),
		UserService:   c.LoginService, ProfileService: c.ProfileService, PageSize: c.PageSize,
	}
}

func (c *GlobalConfig) ExtractProfileConfig() ProfileConfig {
	return ProfileConfig{
		ServiceConfig: MakeServiceConfig(c, c.ProfileService),
		AdminService:  c.RightClient, LoginService: c.LoginService,
	}
}

func (c *GlobalConfig) ExtractSettingsConfig() SettingsConfig {
	return MakeServiceConfig(c, c.SettingsService)
}

func (c *GlobalConfig) CreateWikiConfig(wikiId uint64, groupId uint64, args ...string) WikiConfig {
	c.loadWiki()
	return WikiConfig{
		ServiceConfig: MakeServiceConfig(c, wikiclient.New(
			c.WikiServiceAddr, c.DialOptions, wikiId, groupId, c.DateFormat, c.RightClient, c.ProfileService,
		)),
		MarkdownService: c.MarkdownService, Args: args,
	}
}

func (c *GlobalConfig) CreateForumConfig(forumId uint64, groupId uint64, args ...string) ForumConfig {
	c.loadForum()
	return ForumConfig{
		ServiceConfig: MakeServiceConfig[forumservice.ForumService](c, forumclient.New(
			c.ForumServiceAddr, c.DialOptions, forumId, groupId, c.DateFormat, c.RightClient, c.ProfileService,
		)),
		PageSize: c.PageSize, Args: args,
	}
}

func (c *GlobalConfig) CreateBlogConfig(blogId uint64, groupId uint64, args ...string) BlogConfig {
	c.loadBlog()
	return BlogConfig{
		ServiceConfig: MakeServiceConfig(c, blogclient.New(
			c.BlogServiceAddr, c.DialOptions, blogId, groupId, c.DateFormat, c.RightClient, c.ProfileService,
		)),
		MarkdownService: c.MarkdownService, CommentService: forumclient.New(
			c.ForumServiceAddr, c.DialOptions, blogId, groupId, c.DateFormat, c.RightClient, c.ProfileService,
		),
		Domain: c.Domain, Port: c.Port, DateFormat: c.DateFormat, PageSize: c.PageSize, ExtractSize: c.ExtractSize,
		FeedFormat: c.FeedFormat, FeedSize: c.FeedSize, Args: args,
	}
}

func (c *GlobalConfig) CreateWidgetConfig(serviceAddr string, objectId uint64, groupId uint64) WidgetConfig {
	return MakeServiceConfig(c, widgetclient.New(serviceAddr, c.DialOptions, objectId, groupId))
}

func retrieveWithDefault(logger otelzap.LoggerWithCtx, name string, value string, defaultValue string) string {
	if value == "" {
		logger.Info(name+" empty, using default", zap.String(defaultName, defaultValue))
		return defaultValue
	}
	return value
}

func retrieveUintWithDefault(logger otelzap.LoggerWithCtx, name string, value uint64, defaultValue uint64) uint64 {
	if value == 0 {
		logger.Info(name+" empty, using default", zap.Uint64(defaultName, defaultValue))
		return defaultValue
	}
	return value
}

func retrievePath(logger otelzap.LoggerWithCtx, name string, path string, defaultPath string) string {
	path = retrieveWithDefault(logger, name, path, defaultPath)
	if last := len(path) - 1; path[last] == '/' {
		path = path[:last]
	}
	return path
}

func require(logger otelzap.LoggerWithCtx, name string, value string) {
	if value == "" {
		logger.Fatal(name + " is required")
	}
}
