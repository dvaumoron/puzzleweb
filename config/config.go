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
	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	blogservice "github.com/dvaumoron/puzzleweb/blog/service"
	forumservice "github.com/dvaumoron/puzzleweb/forum/service"
	loginservice "github.com/dvaumoron/puzzleweb/login/service"
	markdownservice "github.com/dvaumoron/puzzleweb/markdown/service"
	profileservice "github.com/dvaumoron/puzzleweb/profile/service"
	sessionservice "github.com/dvaumoron/puzzleweb/session/service"
	wikiservice "github.com/dvaumoron/puzzleweb/wiki/service"
	"go.uber.org/zap"
)

type BaseConfig interface {
	GetLogger() *zap.Logger
	GetTemplatesExt() string
}

type LocalesConfig struct {
	Logger         *zap.Logger
	Domain         string
	SessionTimeOut int
	Path           string
	PasswordRules  map[string]string
}

type ServiceConfig[ServiceType any] struct {
	Logger  *zap.Logger
	Service ServiceType
	Ext     string
}

func MakeServiceConfig[ServiceType any](c BaseConfig, service ServiceType) ServiceConfig[ServiceType] {
	return ServiceConfig[ServiceType]{Logger: c.GetLogger(), Service: service, Ext: c.GetTemplatesExt()}
}

func (c *ServiceConfig[ServiceType]) GetLogger() *zap.Logger {
	return c.Logger
}

func (c *ServiceConfig[ServiceType]) GetTemplatesExt() string {
	return c.Ext
}

type SessionConfig struct {
	ServiceConfig[sessionservice.SessionService]
	Domain  string
	TimeOut int
}

type SiteConfig struct {
	ServiceConfig[sessionservice.SessionService]
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
