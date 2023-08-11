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
	"strings"
	"time"

	"github.com/dvaumoron/puzzlesaltclient"
	"github.com/dvaumoron/puzzletelemetry"
	adminclient "github.com/dvaumoron/puzzleweb/admin/client"
	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	blogclient "github.com/dvaumoron/puzzleweb/blog/client"
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
	DateFormat         string // TODO move this to template service
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
	MarkdownService markdownservice.MarkdownService

	// lazy & only adresses (instance need specific data)
	WikiServiceAddr  string
	ForumServiceAddr string
	BlogServiceAddr  string
}

func LoadDefault(serviceName string, version string) (*GlobalConfig, trace.Span) {
	logger, tp := puzzletelemetry.Init(serviceName, version)
	tracer := tp.Tracer(WebKey)

	ctx, initSpan := tracer.Start(context.Background(), "initialization")
	ctxLogger := logger.Ctx(ctx)

	var sessionTimeOut int
	var serviceTimeOut time.Duration
	var maxMultipartMemory int64

	domain := retrieveWithDefault(ctxLogger, "SITE_DOMAIN", "localhost")
	port := retrieveWithDefault(ctxLogger, "SITE_PORT", "8080")

	sessionTimeOutStr := os.Getenv("SESSION_TIME_OUT")
	if sessionTimeOutStr == "" {
		ctxLogger.Info("SESSION_TIME_OUT not found, using default", zap.Int(defaultName, defaultSessionTimeOut))
		sessionTimeOut = defaultSessionTimeOut
	} else if sessionTimeOut, _ = strconv.Atoi(sessionTimeOutStr); sessionTimeOut == 0 {
		ctxLogger.Warn("Failed to parse SESSION_TIME_OUT, using default", zap.Int(defaultName, defaultSessionTimeOut))
		sessionTimeOut = defaultSessionTimeOut
	}

	serviceTimeOutStr := os.Getenv("SERVICE_TIME_OUT")
	if serviceTimeOutStr == "" {
		ctxLogger.Info("SERVICE_TIME_OUT not found, using default", zap.Duration(defaultName, defaultServiceTimeOut))
		serviceTimeOut = defaultServiceTimeOut
	} else if timeOut, _ := strconv.ParseInt(serviceTimeOutStr, 10, 64); timeOut == 0 {
		ctxLogger.Warn("Failed to parse SERVICE_TIME_OUT, using default", zap.Duration(defaultName, defaultServiceTimeOut))
		serviceTimeOut = defaultServiceTimeOut
	} else {
		serviceTimeOut = time.Duration(timeOut) * time.Second
	}

	maxMultipartMemoryStr := os.Getenv("MAX_MULTIPART_MEMORY")
	if maxMultipartMemoryStr == "" {
		ctxLogger.Info("MAX_MULTIPART_MEMORY not found, using gin default")
	} else {
		if maxMultipartMemory, _ = strconv.ParseInt(maxMultipartMemoryStr, 10, 64); maxMultipartMemory == 0 {
			ctxLogger.Warn("Failed to parse MAX_MULTIPART_MEMORY, using gin default")
		}
	}

	dateFormat := retrieveWithDefault(ctxLogger, "DATE_FORMAT", "2/1/2006 15:04:05")
	pageSize := retrieveUintWithDefault(ctxLogger, "PAGE_SIZE", 20)
	extractSize := retrieveUintWithDefault(ctxLogger, "EXTRACT_SIZE", 200)
	feedFormat := retrieveWithDefault(ctxLogger, "FEED_FORMAT", "atom")
	feedSize := retrieveUintWithDefault(ctxLogger, "FEED_SIZE", 100)

	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	}

	sessionService := sessionclient.New(requiredFromEnv(ctxLogger, "SESSION_SERVICE_ADDR"), dialOptions)
	templateService := templateclient.New(requiredFromEnv(ctxLogger, "TEMPLATE_SERVICE_ADDR"), dialOptions)
	settingsService := sessionclient.New(requiredFromEnv(ctxLogger, "SETTINGS_SERVICE_ADDR"), dialOptions)
	strengthService := strengthclient.New(requiredFromEnv(ctxLogger, "PASSSTRENGTH_SERVICE_ADDR"), dialOptions)
	saltService := puzzlesaltclient.Make(requiredFromEnv(ctxLogger, "SALT_SERVICE_ADDR"), dialOptions)
	loginService := loginclient.New(
		requiredFromEnv(ctxLogger, "LOGIN_SERVICE_ADDR"), dialOptions, dateFormat, saltService, strengthService,
	)
	rightClient := adminclient.Make(requiredFromEnv(ctxLogger, "RIGHT_SERVICE_ADDR"), dialOptions)

	staticPath := retrievePath(ctxLogger, "STATIC_PATH", "static")
	augmentedStaticPath := staticPath + "/"
	faviconPath := os.Getenv("FAVICON_PATH")
	if faviconPath == "" {
		faviconPath = staticPath + DefaultFavicon
		ctxLogger.Info("FAVICON_PATH not found, using default", zap.String(defaultName, faviconPath))
	} else if faviconPath[0] != '/' {
		// user should use absolute path or path relative to STATIC_PATH
		faviconPath = augmentedStaticPath + faviconPath
	}

	defaultPicturePath := retrieveWithDefault(ctxLogger, "PROFILE_DEFAULT_PICTURE_PATH", staticPath+"/images/unknownuser.png")
	defaultPicture, err := os.ReadFile(defaultPicturePath)
	if err != nil {
		ctxLogger.Fatal("Can not read", zap.String("filepath", defaultPicturePath), zap.Error(err))
	}

	confLangs := strings.Split(os.Getenv("AVAILABLE_LOCALES"), ",")
	langNumber := len(confLangs)
	allLang := make([]string, 0, langNumber)
	for _, lang := range confLangs {
		allLang = append(allLang, strings.TrimSpace(lang))
	}
	ctxLogger.Info("Declared locales", zap.Strings("locales", allLang))

	langPicturePaths := make(map[string]string, langNumber)
	confLangPicturePaths := strings.Split(os.Getenv("LOCALE_PICTURE_PATHS"), ",")
	confLangPicturePathsLen := len(confLangPicturePaths)
	for index, lang := range allLang {
		if index >= confLangPicturePathsLen {
			ctxLogger.Warn("LOCALE_PICTURE_PATHS have less element than AVAILABLE_LOCALES")
			break
		}

		langPicturePath := strings.TrimSpace(confLangPicturePaths[index])
		if langPicturePath == "" {
			// skip not configured picture
			continue
		}
		// user should use absolute path or path relative to STATIC_PATH
		if langPicturePath[0] != '/' {
			langPicturePath = augmentedStaticPath + langPicturePath
		}
		langPicturePaths[lang] = langPicturePath
	}

	// if not setted in configuration, profile are public
	profileGroupId := retrieveUintWithDefault(ctxLogger, "PROFILE_GROUP_ID", adminservice.PublicGroupId)
	profileService := profileclient.New(
		requiredFromEnv(ctxLogger, "PROFILE_SERVICE_ADDR"), dialOptions,
		profileGroupId, loginService, rightClient, defaultPicture,
	)

	globalConfig := &GlobalConfig{
		Domain: domain, Port: port, AllLang: allLang, SessionTimeOut: sessionTimeOut, ServiceTimeOut: serviceTimeOut,
		MaxMultipartMemory: maxMultipartMemory, DateFormat: dateFormat, PageSize: pageSize, ExtractSize: extractSize,
		FeedFormat: feedFormat, FeedSize: feedSize,

		StaticPath:  staticPath,
		FaviconPath: faviconPath,
		Page404Url:  os.Getenv("PAGE_404_URL"),

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
	}

	return globalConfig, initSpan
}

func (c *GlobalConfig) loadMarkdown() {
	if c.MarkdownService == nil {
		c.MarkdownService = markdownclient.New(requiredFromEnv(c.CtxLogger, "MARKDOWN_SERVICE_ADDR"), c.DialOptions)
	}
}

func (c *GlobalConfig) loadWiki() {
	if c.WikiServiceAddr == "" {
		c.loadMarkdown()
		c.WikiServiceAddr = requiredFromEnv(c.CtxLogger, "WIKI_SERVICE_ADDR")
	}
}

func (c *GlobalConfig) loadForum() {
	if c.ForumServiceAddr == "" {
		c.ForumServiceAddr = requiredFromEnv(c.CtxLogger, "FORUM_SERVICE_ADDR")
	}
}

func (c *GlobalConfig) loadBlog() {
	if c.BlogServiceAddr == "" {
		c.loadForum()
		c.loadMarkdown()
		c.BlogServiceAddr = requiredFromEnv(c.CtxLogger, "BLOG_SERVICE_ADDR")
	}
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
		DateFormat: c.DateFormat, PageSize: c.PageSize, ExtractSize: c.ExtractSize, FeedFormat: c.FeedFormat,
		FeedSize: c.FeedSize, Args: args,
	}
}

func (c *GlobalConfig) CreateWidgetConfig(serviceAddr string, objectId uint64, groupId uint64) WidgetConfig {
	return MakeServiceConfig(c, widgetclient.New(serviceAddr, c.DialOptions, objectId, groupId))
}

func retrieveWithDefault(logger otelzap.LoggerWithCtx, name string, defaultValue string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	logger.Info(name+" not found, using default", zap.String(defaultName, defaultValue))
	return defaultValue
}

func retrieveUintWithDefault(logger otelzap.LoggerWithCtx, name string, defaultValue uint64) uint64 {
	valueStr := os.Getenv(name)
	if valueStr == "" {
		logger.Info(name+" not found, using default", zap.Uint64(defaultName, defaultValue))
		return defaultValue
	}
	value, _ := strconv.ParseUint(valueStr, 10, 64)
	if value == 0 {
		var messageBuilder strings.Builder
		messageBuilder.WriteString("Failed to parse ")
		messageBuilder.WriteString(name)
		messageBuilder.WriteString(" using default")
		logger.Warn(messageBuilder.String(), zap.Uint64(defaultName, defaultValue))
		return defaultValue
	}
	return value
}

func retrievePath(logger otelzap.LoggerWithCtx, name string, defaultPath string) string {
	if path := os.Getenv(name); path != "" {
		if last := len(path) - 1; path[last] == '/' {
			path = path[:last]
		}
		return path
	}
	logger.Info(name+" not found, using default", zap.String(defaultName, defaultPath))
	return defaultPath
}

func requiredFromEnv(logger otelzap.LoggerWithCtx, name string) string {
	value := os.Getenv(name)
	if value == "" {
		logger.Fatal(name + " not found in env")
	}
	return value
}
