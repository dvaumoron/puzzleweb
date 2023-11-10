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
	"time"

	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	blogservice "github.com/dvaumoron/puzzleweb/blog/service"
	"github.com/dvaumoron/puzzleweb/common/log"
	forumservice "github.com/dvaumoron/puzzleweb/forum/service"
	loginservice "github.com/dvaumoron/puzzleweb/login/service"
	markdownservice "github.com/dvaumoron/puzzleweb/markdown/service"
	profileservice "github.com/dvaumoron/puzzleweb/profile/service"
	widgetservice "github.com/dvaumoron/puzzleweb/remotewidget/service"
	sessionservice "github.com/dvaumoron/puzzleweb/session/service"
	templateservice "github.com/dvaumoron/puzzleweb/templates/service"
	wikiservice "github.com/dvaumoron/puzzleweb/wiki/service"
)

const (
	WebKey = "puzzleWeb"

	DefaultFavicon = "/favicon.ico"
)

type AuthConfig = ServiceConfig[adminservice.AuthService]
type LoginConfig = ServiceConfig[loginservice.LoginService]
type SettingsConfig = ServiceConfig[sessionservice.SessionService]
type TemplateConfig = ServiceConfig[templateservice.TemplateService]
type RemoteWidgetConfig = ServiceConfig[widgetservice.WidgetService]

type BaseConfig interface {
	GetLogger() log.Logger
}

type BaseConfigExtracter interface {
	BaseConfig
	GetLoggerGetter() log.LoggerGetter
	GetServiceTimeOut() time.Duration
	ExtractLocalesConfig() LocalesConfig
	ExtractLoginConfig() LoginConfig
	ExtractAdminConfig() AdminConfig
	ExtractSettingsConfig() SettingsConfig
	ExtractProfileConfig() ProfileConfig
}

type LocalesConfig struct {
	Logger         log.Logger
	LoggerGetter   log.LoggerGetter
	Domain         string
	SessionTimeOut int
	AllLang        []string
}

type ServiceConfig[ServiceType any] struct {
	Logger  log.Logger // for init phase (have the context)
	Service ServiceType
}

func MakeServiceConfig[ServiceType any](c BaseConfig, service ServiceType) ServiceConfig[ServiceType] {
	return ServiceConfig[ServiceType]{Logger: c.GetLogger(), Service: service}
}

func (c *ServiceConfig[ServiceType]) GetLogger() log.Logger {
	return c.Logger
}

type SessionConfig struct {
	ServiceConfig[sessionservice.SessionService]
	Domain  string
	TimeOut int
}

type SiteConfig struct {
	ServiceConfig[sessionservice.SessionService]
	TemplateService    templateservice.TemplateService
	LoggerGetter       log.LoggerGetter
	Domain             string
	Port               string
	SessionTimeOut     int
	MaxMultipartMemory int64
	StaticPath         string
	FaviconPath        string
	Page404Url         string
	LangPicturePaths   map[string]string
}

func (sc *SiteConfig) ExtractSessionConfig() SessionConfig {
	return SessionConfig{
		ServiceConfig: sc.ServiceConfig, Domain: sc.Domain, TimeOut: sc.SessionTimeOut,
	}
}

func (sc *SiteConfig) ExtractTemplateConfig() TemplateConfig {
	return MakeServiceConfig(sc, sc.TemplateService)
}

type AdminConfig struct {
	ServiceConfig[adminservice.AdminService]
	UserService    loginservice.AdvancedUserService
	ProfileService profileservice.AdvancedProfileService
	PageSize       uint64
}

type ProfileConfig struct {
	ServiceConfig[profileservice.AdvancedProfileService]
	AdminService adminservice.AdminService
	LoginService loginservice.LoginService
}

type BlogConfig struct {
	ServiceConfig[blogservice.BlogService]
	MarkdownService markdownservice.MarkdownService
	CommentService  forumservice.CommentService
	Domain          string
	Port            string
	DateFormat      string
	PageSize        uint64
	ExtractSize     uint64
	FeedFormat      string
	FeedSize        uint64
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
