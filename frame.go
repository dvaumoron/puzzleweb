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
	"context"
	_ "embed"
	"os"
	"strings"

	"github.com/dvaumoron/puzzleweb/common/build"
	"github.com/dvaumoron/puzzleweb/common/config"
	"github.com/dvaumoron/puzzleweb/common/config/parser"
	puzzleweb "github.com/dvaumoron/puzzleweb/core"
	"go.uber.org/zap"
)

//go:embed version.txt
var version string

func main() {
	confPath := "frame.hcl"
	if len(os.Args) > 1 {
		confPath = os.Args[1]
	}

	parsedConfig, err := parser.ParseConfig(confPath)
	globalConfig, initSpan := config.Init(config.WebKey, version, parsedConfig, err)
	site, ok := puzzleweb.BuildDefaultSite(globalConfig)
	if !ok {
		return
	}

	// create group for permissions
	rightClient := globalConfig.RightClient
	for _, group := range parsedConfig.PermissionGroups {
		if !rightClient.RegisterGroup(group.Id, group.Name) {
			return
		}
	}

	logger := globalConfig.Logger
	for _, pageGroup := range parsedConfig.StaticPages {
		if !site.AddStaticPages(pageGroup) {
			logger.Error("Failure during static pages creation")
			return
		}
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
				logger.Error("Failed to retrieve parentPage", zap.String("emplacement", emplacement))
				continue
			}
		}

		widgetPage, add := build.MakeWidgetPage(name, globalConfig.InitCtx, globalConfig, widgets[widgetPageConfig.WidgetRef])
		if add {
			if nested {
				if !parentPage.AddSubPage(widgetPage) {
					logger.Error("Only static page can have sub page")
					return
				}
			} else {
				site.AddPage(widgetPage)
			}
		}
	}

	initSpan.End()

	loggerGetter, tracerProvider, tracer := globalConfig.LoggerGetter, globalConfig.TracerProvider, globalConfig.Tracer
	defer func() {
		ctx := context.Background()
		if err := tracerProvider.Shutdown(ctx); err != nil {
			ctx, stopSpan := tracer.Start(ctx, "shutdown")
			loggerGetter.Logger(ctx).Warn("Failed to shutdown trace provider", zap.Error(err))
			stopSpan.End()
		}
	}()

	siteConfig := globalConfig.ExtractSiteConfig()
	// emptying data no longer useful for GC cleaning
	globalConfig = nil

	if err := site.Run(siteConfig); err != nil {
		logger.Error("Failed to serve", zap.Error(err))
	}
}
