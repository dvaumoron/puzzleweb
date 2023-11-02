/*
 *
 * Copyright 2023 puzzleweb authors.
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

package main

import (
	_ "embed"
	"os"
	"strings"

	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/blog"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/config/parser"
	"github.com/dvaumoron/puzzleweb/forum"
	puzzleweb "github.com/dvaumoron/puzzleweb/main"
	"github.com/dvaumoron/puzzleweb/remotewidget"
	"github.com/dvaumoron/puzzleweb/wiki"
	"go.uber.org/zap"
)

const notFound = "notFound"
const castMsg = "Failed to cast value"
const valueName = "valueName"

//go:embed version.txt
var version string

func main() {
	site, globalConfig, initSpan := puzzleweb.BuildDefaultSite(config.WebKey, version)
	ctxLogger := globalConfig.CtxLogger
	rightClient := globalConfig.RightClient

	frameConfig, err := parser.LoadFrameConfig(os.Getenv("FRAME_CONFIG_PATH"))
	if err != nil {
		ctxLogger.Fatal("Failed to read frame configuration file", zap.Error(err))
	}

	// create group for permissions
	for _, group := range frameConfig.PermissionGroups {
		rightClient.RegisterGroup(group.Id, group.Name)
	}

	site.AddPage(puzzleweb.MakeHiddenStaticPage(globalConfig.Tracer, notFound, adminservice.PublicGroupId, notFound))

	for _, pageGroup := range frameConfig.StaticPages {
		site.AddStaticPages(globalConfig.CtxLogger, pageGroup.GroupId, pageGroup.Locations)
	}

	widgets := frameConfig.WidgetsAsMap()
	for _, widgetPageConfig := range frameConfig.WidgetPages {
		name := widgetPageConfig.Path
		nested := false
		var parentPage puzzleweb.Page
		if index := strings.LastIndex(name, "/"); index != -1 {
			emplacement := name[:index]
			name = name[index+1:]
			parentPage, nested = site.GetPageWithPath(emplacement)
			if !nested {
				ctxLogger.Fatal("Failed to retrive parentPage", zap.String("emplacement", emplacement))
			}
		}

		widgetPage := makeWidgetPage(name, globalConfig, widgets[widgetPageConfig.WidgetRef])

		if nested {
			parentPage.AddSubPage(widgetPage)
		} else {
			site.AddPage(widgetPage)
		}
	}

	initSpan.End()

	siteConfig := globalConfig.ExtractSiteConfig()
	// emptying data no longer useful for GC cleaning
	globalConfig = nil

	if err := site.Run(siteConfig); err != nil {
		siteConfig.Logger.Fatal("Failed to serve", zap.Error(err))
	}
}

func makeWidgetPage(pageName string, globalConfig *config.GlobalConfig, widgetConfig parser.WidgetConfig) puzzleweb.Page {
	switch ctxLogger, kind := globalConfig.CtxLogger, widgetConfig.Kind; kind {
	case "forum":
		forumId, groupId := widgetConfig.ObjectId, widgetConfig.GroupId
		args := widgetConfig.Templates
		return forum.MakeForumPage(pageName, globalConfig.CreateForumConfig(forumId, groupId, args...))
	case "blog":
		blogId, groupId := widgetConfig.ObjectId, widgetConfig.GroupId
		args := widgetConfig.Templates
		return blog.MakeBlogPage(pageName, globalConfig.CreateBlogConfig(blogId, groupId, args...))
	case "wiki":
		wikiId, groupId := widgetConfig.ObjectId, widgetConfig.GroupId
		args := widgetConfig.Templates
		return wiki.MakeWikiPage(pageName, globalConfig.CreateWikiConfig(wikiId, groupId, args...))
	default:
		if widgetName, ok := strings.CutPrefix(kind, "remote/"); ok {
			serviceAddr := widgetConfig.ServiceAddr
			objectId, groupId := widgetConfig.ObjectId, widgetConfig.GroupId
			return remotewidget.MakeRemotePage(pageName, ctxLogger, widgetName, globalConfig.CreateWidgetConfig(serviceAddr, objectId, groupId))
		}
		ctxLogger.Fatal("Widget kind unknown ", zap.String("widgetKind", kind))
	}
	return puzzleweb.Page{} // unreachable
}
