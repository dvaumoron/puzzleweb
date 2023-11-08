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

	"github.com/dvaumoron/puzzleweb/common/build"
	"github.com/dvaumoron/puzzleweb/common/config"
	"github.com/dvaumoron/puzzleweb/common/config/parser"
	puzzleweb "github.com/dvaumoron/puzzleweb/core"
	"go.uber.org/zap"
)

const notFound = "notFound"

//go:embed version.txt
var version string

func main() {
	confPath := "frame.hcl"
	if len(os.Args) > 1 {
		confPath = os.Args[1]
	}

	parsedConfig, err := parser.ParseConfig(confPath)
	site, globalConfig, initSpan := puzzleweb.BuildDefaultSite(config.WebKey, version, parsedConfig, err)
	logger := globalConfig.Logger
	rightClient := globalConfig.RightClient

	// create group for permissions
	for _, group := range parsedConfig.PermissionGroups {
		rightClient.RegisterGroup(group.Id, group.Name)
	}

	for _, pageGroup := range parsedConfig.StaticPages {
		site.AddStaticPages(pageGroup)
	}

	widgets := parsedConfig.WidgetsAsMap()
	for _, widgetPageConfig := range parsedConfig.WidgetPages {
		name := widgetPageConfig.Path
		nested := false
		var parentPage puzzleweb.Page
		if index := strings.LastIndex(name, "/"); index != -1 {
			emplacement := name[:index]
			name = name[index+1:]
			parentPage, nested = site.GetPageWithPath(emplacement)
			if !nested {
				logger.Fatal("Failed to retrive parentPage", zap.String("emplacement", emplacement))
			}
		}

		widgetPage := build.MakeWidgetPage(name, globalConfig.InitCtx, logger, globalConfig, widgets[widgetPageConfig.WidgetRef])

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
