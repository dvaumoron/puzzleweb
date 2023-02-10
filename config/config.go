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

	"github.com/dvaumoron/puzzlesaltclient"
	adminclient "github.com/dvaumoron/puzzleweb/admin/client"
	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	blogclient "github.com/dvaumoron/puzzleweb/blog/client"
	blogservice "github.com/dvaumoron/puzzleweb/blog/service"
	forumclient "github.com/dvaumoron/puzzleweb/forum/client"
	forumservice "github.com/dvaumoron/puzzleweb/forum/service"
	"github.com/dvaumoron/puzzleweb/log"
	loginclient "github.com/dvaumoron/puzzleweb/login/client"
	loginservice "github.com/dvaumoron/puzzleweb/login/service"
	markdownclient "github.com/dvaumoron/puzzleweb/markdown/client"
	markdownservice "github.com/dvaumoron/puzzleweb/markdown/service"
	profileclient "github.com/dvaumoron/puzzleweb/profile/client"
	profileservice "github.com/dvaumoron/puzzleweb/profile/service"
	sessionclient "github.com/dvaumoron/puzzleweb/session/client"
	sessionservice "github.com/dvaumoron/puzzleweb/session/service"
	wikiclient "github.com/dvaumoron/puzzleweb/wiki/client"
	wikiservice "github.com/dvaumoron/puzzleweb/wiki/service"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

const defaultSessionTimeOut = 1200

type LocalesConfig struct {
	Logger         *zap.Logger
	Domain         string
	SessionTimeOut int
	Path           string
	AllLang        []string
}

type ServiceConfig[ServiceType any] struct {
	Logger  *zap.Logger
	Service ServiceType
}

type ServiceExtConfig[ServiceType any] struct {
	Logger  *zap.Logger
	Service ServiceType
	Ext     string
}

type SessionConfig struct {
	ServiceConfig[sessionservice.SessionService]
	Domain  string
	TimeOut int
}

type SiteConfig struct {
	ServiceConfig[sessionservice.SessionService]
	PictureService profileservice.PictureService

	Domain         string
	Port           string
	SessionTimeOut int
	StaticPath     string
}

func (sc *SiteConfig) ExtractSessionConfig() SessionConfig {
	return SessionConfig{
		ServiceConfig: sc.ServiceConfig, Domain: sc.Domain, TimeOut: sc.SessionTimeOut,
	}
}

type AdminConfig struct {
	ServiceExtConfig[adminservice.AdminService]
	UserService    loginservice.AdvancedUserService
	ProfileService profileservice.AdvancedProfileService
	PageSize       uint64
}

type ProfileConfig struct {
	ServiceExtConfig[profileservice.AdvancedProfileService]
	AdminService adminservice.AdminService
	LoginService loginservice.LoginService
}

type BlogConfig struct {
	ServiceConfig[blogservice.BlogService]
	MarkdownService markdownservice.MarkdownService
	CommentService  forumservice.CommentService
	PageSize        uint64
	ExtractSize     uint64
	Args            []string
}

type ForumConfig struct {
	ServiceConfig[forumservice.ForumService]
	PageSize uint64
	Args     []string
}

type WikiConfig struct {
	ServiceConfig[wikiservice.WikiService]
	MarkdownService markdownservice.MarkdownService
	Args            []string
}

type GlobalConfig struct {
	Domain string
	Port   string

	AllLang        []string
	SessionTimeOut int
	DateFormat     string
	PageSize       uint64
	ExtractSize    uint64

	StaticPath    string
	LocalesPath   string
	TemplatesPath string
	TemplatesExt  string

	Logger *zap.Logger

	SessionService  sessionservice.SessionService
	SaltService     loginservice.SaltService
	SettingsService sessionservice.SessionService
	LoginService    loginservice.FullLoginService
	RightClient     adminclient.RightClient

	// lazy service
	ProfileService  profileservice.AdvancedProfileService
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

	domain := retrieveWithDefault("SITE_DOMAIN", "localhost")
	port := retrieveWithDefault("SITE_PORT", "8080")

	confLang := strings.Split(os.Getenv("AVAILABLE_LOCALES"), ",")
	allLang := make([]string, 0, len(confLang))
	for _, s := range confLang {
		allLang = append(allLang, strings.TrimSpace(s))
	}

	sessionTimeOutStr := os.Getenv("SESSION_TIME_OUT")
	if sessionTimeOutStr == "" {
		sessionTimeOut = defaultSessionTimeOut
	} else {
		sessionTimeOut, _ = strconv.Atoi(sessionTimeOutStr)
		if sessionTimeOut == 0 {
			fmt.Println("Failed to parse SESSION_TIME_OUT, using default")
			sessionTimeOut = defaultSessionTimeOut
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
	logger := log.NewLogger(logConfig)

	sessionService := sessionclient.New(requiredFromEnv("SESSION_SERVICE_ADDR"), logger)
	saltService := puzzlesaltclient.Make(requiredFromEnv("SALT_SERVICE_ADDR"))
	settingsService := sessionclient.New(requiredFromEnv("SETTINGS_SERVICE_ADDR"), logger)
	loginService := loginclient.New(requiredFromEnv("LOGIN_SERVICE_ADDR"), logger, dateFormat, saltService)
	rightClient := adminclient.Make(requiredFromEnv("RIGHT_SERVICE_ADDR"), logger)

	return &GlobalConfig{
		Domain: domain, Port: port, AllLang: allLang, SessionTimeOut: sessionTimeOut,
		DateFormat: dateFormat, PageSize: pageSize, ExtractSize: extractSize,

		StaticPath:    retrievePath("STATIC_PATH", "static"),
		LocalesPath:   retrievePath("LOCALES_PATH", "locales"),
		TemplatesPath: retrievePath("TEMPLATES_PATH", "templates"),
		TemplatesExt:  retrieveWithDefault("TEMPLATES_EXT", ".html"),

		Logger:          logger,
		SessionService:  sessionService,
		SaltService:     saltService,
		SettingsService: settingsService,
		LoginService:    loginService,
		RightClient:     rightClient,
	}
}

func (c *GlobalConfig) loadProfile() {
	if c.ProfileService == nil {
		// if not setted in configuration, profile are public
		profileGroupId := retrieveUintWithDefault("PROFILE_GROUP_ID", adminservice.PublicGroupId)
		c.ProfileService = profileclient.New(
			requiredFromEnv("PROFILE_SERVICE_ADDR"), c.Logger, profileGroupId,
			c.LoginService, c.RightClient,
		)
	}
}

func (c *GlobalConfig) loadMarkdown() {
	if c.MarkdownService == nil {
		c.MarkdownService = markdownclient.New(requiredFromEnv("MARKDOWN_SERVICE_ADDR"), c.Logger)
	}
}

func (c *GlobalConfig) loadWiki() {
	if c.WikiServiceAddr == "" {
		c.loadProfile()
		c.loadMarkdown()
		c.WikiServiceAddr = requiredFromEnv("WIKI_SERVICE_ADDR")
	}
}

func (c *GlobalConfig) loadForum() {
	if c.ForumServiceAddr == "" {
		c.loadProfile()
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

func (c *GlobalConfig) ExtractAuthConfig() ServiceConfig[adminservice.AuthService] {
	return ServiceConfig[adminservice.AuthService]{
		Logger: c.Logger, Service: c.RightClient,
	}
}

func (c *GlobalConfig) ExtractLocalesConfig() LocalesConfig {
	return LocalesConfig{
		Logger: c.Logger, Domain: c.Domain, SessionTimeOut: c.SessionTimeOut,
		Path: c.LocalesPath, AllLang: c.AllLang,
	}
}

func (c *GlobalConfig) ExtractSiteConfig() SiteConfig {
	return SiteConfig{
		ServiceConfig: ServiceConfig[sessionservice.SessionService]{
			Logger: c.Logger, Service: c.SessionService,
		},
		PictureService: c.ProfileService, Domain: c.Domain, Port: c.Port,
		SessionTimeOut: c.SessionTimeOut, StaticPath: c.StaticPath,
	}
}

func (c *GlobalConfig) ExtractLoginConfig() ServiceExtConfig[loginservice.LoginService] {
	return ServiceExtConfig[loginservice.LoginService]{
		Logger: c.Logger, Service: c.LoginService, Ext: c.TemplatesExt,
	}
}

func (c *GlobalConfig) ExtractAdminConfig() AdminConfig {
	return AdminConfig{
		ServiceExtConfig: ServiceExtConfig[adminservice.AdminService]{
			Logger: c.Logger, Service: c.RightClient, Ext: c.TemplatesExt,
		},
		UserService: c.LoginService, ProfileService: c.ProfileService, PageSize: c.PageSize,
	}
}

func (c *GlobalConfig) ExtractProfileConfig() ProfileConfig {
	c.loadProfile()
	return ProfileConfig{
		ServiceExtConfig: ServiceExtConfig[profileservice.AdvancedProfileService]{
			Logger: c.Logger, Service: c.ProfileService, Ext: c.TemplatesExt,
		},
		AdminService: c.RightClient, LoginService: c.LoginService,
	}
}

func (c *GlobalConfig) ExtractSettingsConfig() ServiceConfig[sessionservice.SessionService] {
	return ServiceConfig[sessionservice.SessionService]{
		Logger: c.Logger, Service: c.SettingsService,
	}
}

func CreateServiceExtConfig[ServiceType any](c *GlobalConfig, service ServiceType) ServiceExtConfig[ServiceType] {
	return ServiceExtConfig[ServiceType]{
		Logger: c.Logger, Service: service, Ext: c.TemplatesExt,
	}
}

func (c *GlobalConfig) CreateWikiConfig(wikiId uint64, groupId uint64, args ...string) WikiConfig {
	c.loadWiki()
	return WikiConfig{
		ServiceConfig: ServiceConfig[wikiservice.WikiService]{Logger: c.Logger, Service: wikiclient.New(
			c.WikiServiceAddr, c.Logger, wikiId, groupId, c.DateFormat, c.RightClient, c.ProfileService,
		)},
		MarkdownService: c.MarkdownService, Args: args,
	}
}

func (c *GlobalConfig) CreateForumConfig(forumId uint64, groupId uint64, args ...string) ForumConfig {
	c.loadForum()
	return ForumConfig{
		ServiceConfig: ServiceConfig[forumservice.ForumService]{Logger: c.Logger, Service: forumclient.New(
			c.ForumServiceAddr, c.Logger, forumId, groupId, c.DateFormat, c.RightClient, c.ProfileService,
		)},
		PageSize: c.PageSize, Args: args,
	}
}

func (c *GlobalConfig) CreateBlogConfig(blogId uint64, groupId uint64, args ...string) BlogConfig {
	c.loadBlog()
	return BlogConfig{
		ServiceConfig: ServiceConfig[blogservice.BlogService]{Logger: c.Logger, Service: blogclient.New(
			c.BlogServiceAddr, c.Logger, blogId, groupId, c.DateFormat, c.RightClient, c.ProfileService,
		)},
		MarkdownService: c.MarkdownService, CommentService: forumclient.New(
			c.ForumServiceAddr, c.Logger, blogId, groupId, c.DateFormat, c.RightClient, c.ProfileService,
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
