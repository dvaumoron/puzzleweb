package build

import (
	"context"
	"strings"

	"github.com/dvaumoron/puzzleweb/blog"
	"github.com/dvaumoron/puzzleweb/common/config"
	"github.com/dvaumoron/puzzleweb/common/config/parser"
	puzzleweb "github.com/dvaumoron/puzzleweb/core"
	"github.com/dvaumoron/puzzleweb/forum"
	"github.com/dvaumoron/puzzleweb/remotewidget"
	"github.com/dvaumoron/puzzleweb/wiki"
	"go.uber.org/zap"
)

type WidgetConfigBuilder interface {
	config.BaseConfig
	MakeWikiConfig(widgetConfig parser.WidgetConfig) (config.WikiConfig, bool)
	MakeForumConfig(widgetConfig parser.WidgetConfig) (config.ForumConfig, bool)
	MakeBlogConfig(widgetConfig parser.WidgetConfig) (config.BlogConfig, bool)
	MakeWidgetConfig(widgetConfig parser.WidgetConfig) (config.RemoteWidgetConfig, bool)
}

func AddWidgetPages(site *puzzleweb.Site, initCtx context.Context, widgetPages []parser.WidgetPageConfig, configBuilder WidgetConfigBuilder, widgets map[string]parser.WidgetConfig) bool {
	for _, widgetPageConfig := range widgetPages {
		name := widgetPageConfig.Path
		nested := false
		var parentPage puzzleweb.Page
		if index := strings.LastIndex(name, "/"); index != -1 {
			emplacement := name[:index]
			name = name[index+1:]
			parentPage, nested = site.GetPageWithPath(emplacement)
			if !nested {
				configBuilder.GetLogger().Error("Failed to retrieve parentPage", zap.String("emplacement", emplacement))
				return false
			}
		}

		widgetPage, add := MakeWidgetPage(name, initCtx, configBuilder, widgets[widgetPageConfig.WidgetRef])
		if add {
			if nested {
				if !parentPage.AddSubPage(widgetPage) {
					configBuilder.GetLogger().Error("Only static page can have sub page")
					return false
				}
			} else {
				site.AddPage(widgetPage)
			}
		}
	}
	return true
}

func MakeWidgetPage(pageName string, initCtx context.Context, configBuilder WidgetConfigBuilder, widgetConfig parser.WidgetConfig) (puzzleweb.Page, bool) {
	switch kind := widgetConfig.Kind; kind {
	case "forum":
		if forumConfig, ok := configBuilder.MakeForumConfig(widgetConfig); ok {
			return forum.MakeForumPage(pageName, forumConfig), true
		}
	case "blog":
		if blogConfig, ok := configBuilder.MakeBlogConfig(widgetConfig); ok {
			return blog.MakeBlogPage(pageName, blogConfig), true
		}
	case "wiki":
		if wikiConfig, ok := configBuilder.MakeWikiConfig(widgetConfig); ok {
			return wiki.MakeWikiPage(pageName, wikiConfig), true
		}
	default:
		if remoteConfig, ok := configBuilder.MakeWidgetConfig(widgetConfig); ok {
			return remotewidget.MakeRemotePage(pageName, initCtx, remoteConfig)
		}
		configBuilder.GetLogger().Error("Widget kind unknown ", zap.String("widgetKind", kind))
	}
	return puzzleweb.Page{}, false
}
