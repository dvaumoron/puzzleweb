package build

import (
	"context"
	"strings"

	"github.com/dvaumoron/puzzleweb/blog"
	"github.com/dvaumoron/puzzleweb/common/config"
	"github.com/dvaumoron/puzzleweb/common/config/parser"
	"github.com/dvaumoron/puzzleweb/common/log"
	puzzleweb "github.com/dvaumoron/puzzleweb/core"
	"github.com/dvaumoron/puzzleweb/forum"
	"github.com/dvaumoron/puzzleweb/remotewidget"
	"github.com/dvaumoron/puzzleweb/wiki"
	"go.uber.org/zap"
)

func MakeWidgetPage(pageName string, initCtx context.Context, logger log.Logger, configBuilder config.WidgetConfigBuilder, widgetConfig parser.WidgetConfig) puzzleweb.Page {
	switch kind := widgetConfig.Kind; kind {
	case "forum":
		return forum.MakeForumPage(pageName, configBuilder.CreateForumConfig(widgetConfig))
	case "blog":
		return blog.MakeBlogPage(pageName, configBuilder.CreateBlogConfig(widgetConfig))
	case "wiki":
		return wiki.MakeWikiPage(pageName, configBuilder.CreateWikiConfig(widgetConfig))
	default:
		if widgetName, ok := strings.CutPrefix(kind, "remote/"); ok {
			return remotewidget.MakeRemotePage(
				pageName, initCtx, logger, widgetName, configBuilder.CreateWidgetConfig(widgetConfig),
			)
		}
		logger.Fatal("Widget kind unknown ", zap.String("widgetKind", kind))
	}
	return puzzleweb.Page{} // unreachable
}
