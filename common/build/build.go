package build

import (
	"context"

	"github.com/dvaumoron/puzzleweb/blog"
	"github.com/dvaumoron/puzzleweb/common/config"
	"github.com/dvaumoron/puzzleweb/common/config/parser"
	puzzleweb "github.com/dvaumoron/puzzleweb/core"
	"github.com/dvaumoron/puzzleweb/forum"
	"github.com/dvaumoron/puzzleweb/remotewidget"
	"github.com/dvaumoron/puzzleweb/wiki"
	"go.uber.org/zap"
)

func MakeWidgetPage(pageName string, initCtx context.Context, configBuilder config.WidgetConfigBuilder, widgetConfig parser.WidgetConfig) (puzzleweb.Page, bool) {
	switch kind := widgetConfig.Kind; kind {
	case "forum":
		if forumConfig, ok := configBuilder.CreateForumConfig(widgetConfig); ok {
			return forum.MakeForumPage(pageName, forumConfig), true
		}
	case "blog":
		if blogConfig, ok := configBuilder.CreateBlogConfig(widgetConfig); ok {
			return blog.MakeBlogPage(pageName, blogConfig), true
		}
	case "wiki":
		if wikiConfig, ok := configBuilder.CreateWikiConfig(widgetConfig); ok {
			return wiki.MakeWikiPage(pageName, wikiConfig), true
		}
	default:
		if remoteConfig, ok := configBuilder.CreateWidgetConfig(widgetConfig); ok {
			return remotewidget.MakeRemotePage(pageName, initCtx, remoteConfig)
		}
		configBuilder.GetLogger().Error("Widget kind unknown ", zap.String("widgetKind", kind))
	}
	return puzzleweb.Page{}, false
}
