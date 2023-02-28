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

func (sc ServiceExtConfig[ServiceType]) ExtractServiceConfig() ServiceConfig[ServiceType] {
	return ServiceConfig[ServiceType]{Logger: sc.Logger, Service: sc.Service}
}

type SessionConfig struct {
	ServiceConfig[sessionservice.SessionService]
	Domain  string
	TimeOut int
}

type SiteConfig struct {
	ServiceConfig[sessionservice.SessionService]
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
