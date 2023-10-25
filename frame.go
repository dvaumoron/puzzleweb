/*
 *
 * Copyright 2023 puzzleframe authors.
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
	"strconv"

	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/blog"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/forum"
	puzzleweb "github.com/dvaumoron/puzzleweb/main"
	"github.com/dvaumoron/puzzleweb/remotewidget"
	"github.com/dvaumoron/puzzleweb/wiki"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
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

	frameConfigBody, err := os.ReadFile(os.Getenv("FRAME_CONFIG_PATH"))
	if err != nil {
		ctxLogger.Fatal("Failed to read frame configuration file", zap.Error(err))
	}

	// TODO use HCL instead ? both options ?
	var frameConfig map[string]any
	if err = yaml.Unmarshal(frameConfigBody, &frameConfig); err != nil {
		ctxLogger.Fatal("Failed to parse frame configuration", zap.Error(err))
	}

	// create group for permissions
	for _, group := range asSlice("permissionGroups", frameConfig["permissionGroups"], ctxLogger) {
		castedGroup := asMap("permissionGroup", group, ctxLogger)
		rightClient.RegisterGroup(
			asUint64("permissionGroup.id", castedGroup["id"], ctxLogger),
			asString("permissionGroup.name", castedGroup["name"], ctxLogger),
		)
	}

	site.AddPage(puzzleweb.MakeHiddenStaticPage(globalConfig.Tracer, notFound, adminservice.PublicGroupId, notFound))

	for _, pageGroup := range asSlice("pageGroups", frameConfig["pageGroups"], ctxLogger) {
		castedPageGroup := asMap("pageGroup", pageGroup, ctxLogger)
		site.AddStaticPages(
			globalConfig.CtxLogger,
			asUint64("pageGroup.id", castedPageGroup["id"], ctxLogger),
			asStringSlice("pageGroup.pages", castedPageGroup["pages"], ctxLogger),
		)
	}

	widgets := asMap("widgets", frameConfig["widgets"], ctxLogger)
	for _, widgetPageConfig := range asSlice("widgetPages", frameConfig["widgetPages"], ctxLogger) {
		castedWidgetPage := asMap("widgetPage", widgetPageConfig, ctxLogger)
		emplacement := asString("widgetPage.emplacement", castedWidgetPage["emplacement"], ctxLogger)
		ok := false
		var parentPage puzzleweb.Page
		if emplacement != "" {
			parentPage, ok = site.GetPageWithPath(emplacement)
			if !ok {
				ctxLogger.Fatal("Failed to retrive parentPage", zap.String("emplacement", emplacement))
			}
		}

		widgetPage := makeWidgetPage(
			asString("widgetPage.name", castedWidgetPage["name"], ctxLogger), globalConfig,
			widgets[asString("widgetPage.widgetRef", castedWidgetPage["widgetRef"], ctxLogger)],
		)

		if ok {
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

func makeWidgetPage(pageName string, globalConfig *config.GlobalConfig, widgetConfig any) puzzleweb.Page {
	ctxLogger := globalConfig.CtxLogger
	castedConfig := asMap("widget", widgetConfig, ctxLogger)

	switch kind := asString("widget.kind", castedConfig["kind"], ctxLogger); kind {
	case "forum":
		forumId := asUint64("widget.forumId", castedConfig["forumId"], ctxLogger)
		groupId := asUint64("widget.groupId", castedConfig["groupId"], ctxLogger)
		args := asStringSlice("widget.templates", castedConfig["templates"], ctxLogger)
		return forum.MakeForumPage(pageName, globalConfig.CreateForumConfig(forumId, groupId, args...))
	case "blog":
		blogId := asUint64("widget.blogId", castedConfig["blogId"], ctxLogger)
		groupId := asUint64("widget.groupId", castedConfig["groupId"], ctxLogger)
		args := asStringSlice("widget.templates", castedConfig["templates"], ctxLogger)
		return blog.MakeBlogPage(pageName, globalConfig.CreateBlogConfig(blogId, groupId, args...))
	case "wiki":
		wikiId := asUint64("widget.wikiId", castedConfig["wikiId"], ctxLogger)
		groupId := asUint64("widget.groupId", castedConfig["groupId"], ctxLogger)
		args := asStringSlice("widget.templates", castedConfig["templates"], ctxLogger)
		return wiki.MakeWikiPage(pageName, globalConfig.CreateWikiConfig(wikiId, groupId, args...))
	case "remote":
		serviceAddr := asString("widget.serviceAddr", castedConfig["serviceAddr"], ctxLogger)
		widgetName := asString("widget.widgetName", castedConfig["widgetName"], ctxLogger)
		objectId := asUint64("widget.objectId", castedConfig["objectId"], ctxLogger)
		groupId := asUint64("widget.groupId", castedConfig["groupId"], ctxLogger)
		return remotewidget.MakeRemotePage(
			pageName, ctxLogger, widgetName, globalConfig.CreateWidgetConfig(serviceAddr, objectId, groupId),
		)
	default:
		globalConfig.CtxLogger.Fatal("Widget kind unknown ", zap.String("widgetKind", kind))
	}
	return puzzleweb.Page{} // unreachable
}

func asUint64(name string, value any, ctxLogger otelzap.LoggerWithCtx) uint64 {
	if value == nil {
		return 0
	}
	switch casted := value.(type) {
	case uint:
		return uint64(casted)
	case uint8:
		return uint64(casted)
	case uint16:
		return uint64(casted)
	case uint32:
		return uint64(casted)
	case uint64:
		return uint64(casted)
	case int:
		return uint64(casted)
	case int8:
		return uint64(casted)
	case int16:
		return uint64(casted)
	case int32:
		return uint64(casted)
	case int64:
		return uint64(casted)
	case float32:
		return uint64(casted)
	case float64:
		return uint64(casted)
	case string:
		i, err := strconv.ParseUint(casted, 10, 64)
		if err != nil {
			ctxLogger.Fatal("Failed to parse value", zap.String(valueName, name), zap.Error(err))
		}
		return i
	default:
		ctxLogger.Fatal(castMsg, zap.String(valueName, name))
	}
	return 0 // unreachable
}

func asMap(name string, value any, ctxLogger otelzap.LoggerWithCtx) map[string]any {
	if value == nil {
		return nil
	}
	m, ok := value.(map[string]any)
	if !ok {
		ctxLogger.Fatal(castMsg, zap.String("valueName", name))
	}
	return m
}

func asSlice(name string, value any, ctxLogger otelzap.LoggerWithCtx) []any {
	if value == nil {
		return nil
	}
	s, ok := value.([]any)
	if !ok {
		ctxLogger.Fatal(castMsg, zap.String(valueName, name))
	}
	return s
}

func asString(name string, value any, ctxLogger otelzap.LoggerWithCtx) string {
	if value == nil {
		return ""
	}
	s, ok := value.(string)
	if !ok {
		ctxLogger.Fatal(castMsg, zap.String(valueName, name))
	}
	return s
}

func asStringSlice(name string, value any, ctxLogger otelzap.LoggerWithCtx) []string {
	s := asSlice(name, value, ctxLogger)
	s2 := make([]string, 0, len(s))
	for _, innerValue := range s {
		s2 = append(s2, asString(name, innerValue, ctxLogger))
	}
	return s2
}
