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
package remotewidget

import (
	"github.com/dvaumoron/puzzleweb"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/gin-gonic/gin"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
)

type handlerDesc struct {
	kind    string
	path    string
	handler gin.HandlerFunc
}

type remoteWidget struct {
	handlers []handlerDesc
}

func (w remoteWidget) LoadInto(router gin.IRouter) {
	for _, desc := range w.handlers {
		router.Handle(desc.kind, desc.path, desc.handler)
	}
}

func NewRemotePage(pageName string, ctxLogger otelzap.LoggerWithCtx, widgetName string, remoteConfig config.WidgetConfig) puzzleweb.Page {
	widgetService := remoteConfig.Service
	actions, err := widgetService.GetDesc(ctxLogger, widgetName)
	if err != nil {
		ctxLogger.Fatal("Failed to init remote widget", zap.Error(err))
	}

	tracer := remoteConfig.Tracer
	widgetNameSlash := widgetName + "/"
	handlers := make([]handlerDesc, 0, len(actions))
	for _, action := range actions {
		actionName := action.Name
		handlers = append(handlers, handlerDesc{kind: action.Kind, path: action.Path, handler: puzzleweb.CreateTemplate(
			tracer, widgetNameSlash+actionName, func(data gin.H, c *gin.Context) (string, string) {
				ctxLogger := puzzleweb.GetLogger(c)
				// TODO init data
				widgetService.Process(ctxLogger, widgetName, actionName, data)
				// TODO
				return "", ""
			},
		)})
	}

	p := puzzleweb.MakePage(pageName)
	p.Widget = remoteWidget{handlers: handlers}
	return p
}
