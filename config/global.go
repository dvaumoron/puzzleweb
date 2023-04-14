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
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dvaumoron/puzzlesaltclient"
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
	sessionclient "github.com/dvaumoron/puzzleweb/session/client"
	sessionservice "github.com/dvaumoron/puzzleweb/session/service"
	wikiclient "github.com/dvaumoron/puzzleweb/wiki/client"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const defaultSessionTimeOut = 1200
const defaultServiceTimeOut = 5 * time.Second

const DefaultFavicon = "/favicon.ico"

type AuthConfig = ServiceConfig[adminservice.AuthService]
type LoginConfig = ServiceConfig[loginservice.LoginService]
type SettingsConfig = ServiceConfig[sessionservice.SessionService]

type BaseConfigExtracter interface {
	BaseConfig
	ExtractLoginConfig() LoginConfig
	ExtractAdminConfig() AdminConfig
	ExtractProfileConfig() ProfileConfig
}

type GlobalConfig struct {
	Domain string
	Port   string

	PasswordRules      map[string]string
	SessionTimeOut     int
	ServiceTimeOut     time.Duration
	MaxMultipartMemory int64
	DateFormat         string
	PageSize           uint64
	ExtractSize        uint64

	StaticPath    string
	FaviconPath   string
	LocalesPath   string
	TemplatesPath string
	TemplatesExt  string
	Page404Url    string

	Logger           *zap.Logger
	LangPicturePaths map[string]string

	DialOptions     grpc.DialOption
	SessionService  sessionservice.SessionService
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

func LoadDefault() *GlobalConfig {
	if godotenv.Overload() == nil {
		fmt.Println("Loaded .env file")
	}

	var err error
	var logConfig []byte
	var sessionTimeOut int
	var serviceTimeOut time.Duration
	var maxMultipartMemory int64

	domain := retrieveWithDefault("SITE_DOMAIN", "localhost")
	port := retrieveWithDefault("SITE_PORT", "8080")

	sessionTimeOutStr := os.Getenv("SESSION_TIME_OUT")
	if sessionTimeOutStr == "" {
		fmt.Println("SESSION_TIME_OUT not found, using default :", defaultSessionTimeOut)
		sessionTimeOut = defaultSessionTimeOut
	} else {
		sessionTimeOut, _ = strconv.Atoi(sessionTimeOutStr)
		if sessionTimeOut == 0 {
			fmt.Println("Failed to parse SESSION_TIME_OUT, using default :", defaultSessionTimeOut)
			sessionTimeOut = defaultSessionTimeOut
		}
	}

	serviceTimeOutStr := os.Getenv("SERVICE_TIME_OUT")
	if serviceTimeOutStr == "" {
		fmt.Println("SERVICE_TIME_OUT not found, using default :", defaultServiceTimeOut)
		serviceTimeOut = defaultServiceTimeOut
	} else if timeOut, _ := strconv.ParseInt(serviceTimeOutStr, 10, 64); timeOut == 0 {
		fmt.Println("Failed to parse SERVICE_TIME_OUT, using default :", defaultServiceTimeOut)
		serviceTimeOut = defaultServiceTimeOut
	} else {
		serviceTimeOut = time.Duration(timeOut) * time.Second
	}

	maxMultipartMemoryStr := os.Getenv("MAX_MULTIPART_MEMORY")
	if maxMultipartMemoryStr == "" {
		fmt.Println("MAX_MULTIPART_MEMORY not found, using gin default")
	} else {
		maxMultipartMemory, _ = strconv.ParseInt(maxMultipartMemoryStr, 10, 64)
		if maxMultipartMemory == 0 {
			fmt.Println("Failed to parse MAX_MULTIPART_MEMORY, using gin default")
		}
	}

	dateFormat := retrieveWithDefault("DATE_FORMAT", "2/1/2006 15:04:05")
	pageSize := retrieveUintWithDefault("PAGE_SIZE", 20)
	extractSize := retrieveUintWithDefault("EXTRACT_SIZE", 200)

	fileLogConfigPath := os.Getenv("LOG_CONFIG_PATH")
	if fileLogConfigPath != "" {
		logConfig, err = os.ReadFile(fileLogConfigPath)
		if err != nil {
			fmt.Println("Failed to read logging config file :", err)
			logConfig = nil
		}
	}
	logger := newLogger(logConfig)

	dialOptions := grpc.WithTransportCredentials(insecure.NewCredentials())

	sessionService := sessionclient.New(requiredFromEnv("SESSION_SERVICE_ADDR"), dialOptions, serviceTimeOut, logger)
	settingsService := sessionclient.New(requiredFromEnv("SETTINGS_SERVICE_ADDR"), dialOptions, serviceTimeOut, logger)
	strengthService := strengthclient.New(requiredFromEnv("PASSSTRENGTH_SERVICE_ADDR"), dialOptions, serviceTimeOut, logger)
	saltService := puzzlesaltclient.Make(requiredFromEnv("SALT_SERVICE_ADDR"), dialOptions, serviceTimeOut)
	loginService := loginclient.New(
		requiredFromEnv("LOGIN_SERVICE_ADDR"), dialOptions, serviceTimeOut,
		logger, dateFormat, saltService, strengthService,
	)
	rightClient := adminclient.Make(requiredFromEnv("RIGHT_SERVICE_ADDR"), dialOptions, serviceTimeOut, logger)

	staticPath := retrievePath("STATIC_PATH", "static")
	augmentedStaticPath := staticPath + "/"
	faviconPath := os.Getenv("FAVICON_PATH")
	if faviconPath == "" {
		faviconPath = staticPath + DefaultFavicon
		fmt.Println("FAVICON_PATH not found, using default :", faviconPath)
	} else if faviconPath[0] != '/' {
		// user should use absolute path or path relative to STATIC_PATH
		faviconPath = augmentedStaticPath + faviconPath
	}

	defaultPicturePath := retrieveWithDefault("PROFILE_DEFAULT_PICTURE_PATH", staticPath+"/images/unknownuser.png")
	defaultPicture, err := os.ReadFile(defaultPicturePath)
	if err != nil {
		fmt.Println("Can not read :", defaultPicturePath)
		os.Exit(1)
	}

	confLangs := strings.Split(os.Getenv("AVAILABLE_LOCALES"), ",")
	langNumber := len(confLangs)
	allLang := make([]string, 0, langNumber)
	passwordRules := make(map[string]string, langNumber)
	for _, confLang := range confLangs {
		lang := strings.TrimSpace(confLang)
		allLang = append(allLang, lang)
		passwordRule, err := strengthService.GetRules(lang)
		if err != nil {
			fmt.Println("Failed to retrieve password rule for", lang, "locale :", err)
		}
		passwordRules[lang] = passwordRule
	}
	fmt.Println("Declared locales : ", allLang)

	langPicturePaths := make(map[string]string, langNumber)
	confLangPicturePaths := strings.Split(os.Getenv("LOCALE_PICTURE_PATHS"), ",")
	confLangPicturePathsLen := len(confLangPicturePaths)
	for index, lang := range allLang {
		if index >= confLangPicturePathsLen {
			fmt.Println("LOCALE_PICTURE_PATHS have less element than AVAILABLE_LOCALES")
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
	profileGroupId := retrieveUintWithDefault("PROFILE_GROUP_ID", adminservice.PublicGroupId)
	profileService := profileclient.New(
		requiredFromEnv("PROFILE_SERVICE_ADDR"), dialOptions, serviceTimeOut,
		logger, profileGroupId, loginService, rightClient, defaultPicture,
	)

	return &GlobalConfig{
		Domain: domain, Port: port, PasswordRules: passwordRules, SessionTimeOut: sessionTimeOut, ServiceTimeOut: serviceTimeOut,
		MaxMultipartMemory: maxMultipartMemory, DateFormat: dateFormat, PageSize: pageSize, ExtractSize: extractSize,

		StaticPath:    staticPath,
		FaviconPath:   faviconPath,
		LocalesPath:   retrievePath("LOCALES_PATH", "locales"),
		TemplatesPath: retrievePath("TEMPLATES_PATH", "templates"),
		TemplatesExt:  retrieveWithDefault("TEMPLATES_EXT", ".html"),
		Page404Url:    os.Getenv("PAGE_404_URL"),

		Logger:           logger,
		LangPicturePaths: langPicturePaths,
		DialOptions:      dialOptions,
		SessionService:   sessionService,
		SaltService:      saltService,
		SettingsService:  settingsService,
		LoginService:     loginService,
		RightClient:      rightClient,
		ProfileService:   profileService,
	}
}

func (c *GlobalConfig) loadMarkdown() {
	if c.MarkdownService == nil {
		c.MarkdownService = markdownclient.New(
			requiredFromEnv("MARKDOWN_SERVICE_ADDR"), c.DialOptions, c.ServiceTimeOut, c.Logger,
		)
	}
}

func (c *GlobalConfig) loadWiki() {
	if c.WikiServiceAddr == "" {
		c.loadMarkdown()
		c.WikiServiceAddr = requiredFromEnv("WIKI_SERVICE_ADDR")
	}
}

func (c *GlobalConfig) loadForum() {
	if c.ForumServiceAddr == "" {
		c.ForumServiceAddr = requiredFromEnv("FORUM_SERVICE_ADDR")
	}
}

func (c *GlobalConfig) loadBlog() {
	if c.BlogServiceAddr == "" {
		c.loadForum()
		c.loadMarkdown()
		c.BlogServiceAddr = requiredFromEnv("BLOG_SERVICE_ADDR")
	}
}

func (c *GlobalConfig) GetLogger() *zap.Logger {
	return c.Logger
}

func (c *GlobalConfig) GetTemplatesExt() string {
	return c.TemplatesExt
}

func (c *GlobalConfig) ExtractAuthConfig() AuthConfig {
	return MakeServiceConfig[adminservice.AuthService](c, c.RightClient)
}

func (c *GlobalConfig) ExtractLocalesConfig() LocalesConfig {
	return LocalesConfig{
		Logger: c.Logger, Domain: c.Domain, SessionTimeOut: c.SessionTimeOut,
		Path: c.LocalesPath, PasswordRules: c.PasswordRules,
	}
}

func (c *GlobalConfig) ExtractSiteConfig() SiteConfig {
	return SiteConfig{
		ServiceConfig: MakeServiceConfig(c, c.SessionService),
		Domain:        c.Domain, Port: c.Port, SessionTimeOut: c.SessionTimeOut, MaxMultipartMemory: c.MaxMultipartMemory,
		StaticPath: c.StaticPath, FaviconPath: c.FaviconPath, LangPicturePaths: c.LangPicturePaths, Page404Url: c.Page404Url,
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
			c.WikiServiceAddr, c.DialOptions, c.ServiceTimeOut, c.Logger,
			wikiId, groupId, c.DateFormat, c.RightClient, c.ProfileService,
		)),
		MarkdownService: c.MarkdownService, Args: args,
	}
}

func (c *GlobalConfig) CreateForumConfig(forumId uint64, groupId uint64, args ...string) ForumConfig {
	c.loadForum()
	return ForumConfig{
		ServiceConfig: MakeServiceConfig[forumservice.ForumService](c, forumclient.New(
			c.ForumServiceAddr, c.DialOptions, c.ServiceTimeOut, c.Logger,
			forumId, groupId, c.DateFormat, c.RightClient, c.ProfileService,
		)),
		PageSize: c.PageSize, Args: args,
	}
}

func (c *GlobalConfig) CreateBlogConfig(blogId uint64, groupId uint64, args ...string) BlogConfig {
	c.loadBlog()
	return BlogConfig{
		ServiceConfig: MakeServiceConfig(c, blogclient.New(
			c.BlogServiceAddr, c.DialOptions, c.ServiceTimeOut, c.Logger,
			blogId, groupId, c.DateFormat, c.RightClient, c.ProfileService,
		)),
		MarkdownService: c.MarkdownService, CommentService: forumclient.New(
			c.ForumServiceAddr, c.DialOptions, c.ServiceTimeOut, c.Logger,
			blogId, groupId, c.DateFormat, c.RightClient, c.ProfileService,
		),
		PageSize: c.PageSize, ExtractSize: c.ExtractSize, Args: args,
	}
}

func retrieveWithDefault(name string, defaultValue string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	fmt.Println(name, "not found, using default :", defaultValue)
	return defaultValue
}

func retrieveUintWithDefault(name string, defaultValue uint64) uint64 {
	var value uint64
	valueStr := os.Getenv(name)
	if valueStr == "" {
		fmt.Println(name, "not found, using default :", defaultValue)
		return defaultValue
	}
	value, _ = strconv.ParseUint(valueStr, 10, 64)
	if value == 0 {
		fmt.Println("Failed to parse", name, "using default :", defaultValue)
		return defaultValue
	}
	return value
}

func retrievePath(name string, defaultPath string) string {
	if path := os.Getenv(name); path != "" {
		if last := len(path) - 1; path[last] == '/' {
			path = path[:last]
		}
		return path
	}
	fmt.Println(name, "not found, using default :", defaultPath)
	return defaultPath
}

func requiredFromEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		fmt.Println(name, "not found in env")
		os.Exit(1)
	}
	return value
}
