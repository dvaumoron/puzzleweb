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
	"encoding/json"
	"net/http"
	"strings"

	"github.com/dvaumoron/puzzleweb"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/remotewidget/service"
	"github.com/gin-gonic/gin"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const formKey = "formData"
const pathKeySlash = "pathData/"

type handlerDesc struct {
	httpMethod string
	path       string
	handler    gin.HandlerFunc
}

type remoteWidget struct {
	handlers []handlerDesc
}

func (w remoteWidget) LoadInto(router gin.IRouter) {
	for _, desc := range w.handlers {
		router.Handle(desc.httpMethod, desc.path, desc.handler)
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
		httpMethod := action.Kind
		actionName := action.Name
		actionPath := action.Path
		var handler gin.HandlerFunc
		switch httpMethod {
		case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace:
			keys := extractKeysFromPath(actionPath)
			dataAdder := func(data gin.H, c *gin.Context) {
				extractPathData(keys, data, c)
			}
			handler = createHandler(tracer, widgetNameSlash+actionName, widgetName, actionName, dataAdder, widgetService)
		case http.MethodPost:
			keys := extractKeysFromPath(actionPath)
			dataAdder := func(data gin.H, c *gin.Context) {
				data[formKey] = c.PostFormMap(formKey)
				extractPathData(keys, data, c)
			}
			handler = createHandler(tracer, widgetNameSlash+actionName, widgetName, actionName, dataAdder, widgetService)
		case service.RawResult:
			httpMethod = http.MethodGet
			keys := extractKeysFromPath(actionPath)
			handler = func(c *gin.Context) {
				ctxLogger := puzzleweb.GetLogger(c)
				data := gin.H{}
				extractPathData(keys, data, c)
				_, _, resData, err := widgetService.Process(ctxLogger, widgetName, actionName, data)
				if err != nil {
					c.AbortWithStatus(http.StatusInternalServerError)
					return
				}
				c.Data(http.StatusOK, http.DetectContentType(resData), resData)
			}
		default:
			ctxLogger.Fatal("Failed to init remote widget", zap.String("unknownActionKind", httpMethod))
		}
		handlers = append(handlers, handlerDesc{httpMethod: httpMethod, path: actionPath, handler: handler})
	}

	p := puzzleweb.MakePage(pageName)
	p.Widget = remoteWidget{handlers: handlers}
	return p
}

func extractKeysFromPath(path string) [][2]string {
	splitted := strings.Split(path, "/")
	keys := make([][2]string, 0, len(splitted))
	for _, part := range splitted {
		if part[0] == ':' {
			key := part[1:]
			keys = append(keys, [2]string{pathKeySlash + key, key})
		}
	}
	return keys
}

func extractPathData(keys [][2]string, data gin.H, c *gin.Context) {
	for _, key := range keys {
		data[key[0]] = c.Param(key[1])
	}
}

func createHandler(tracer trace.Tracer, spanName string, widgetName string, actionName string, dataAdder common.DataAdder, widgetService service.WidgetService) gin.HandlerFunc {
	return puzzleweb.CreateTemplate(tracer, spanName, func(data gin.H, c *gin.Context) (string, string) {
		ctxLogger := puzzleweb.GetLogger(c)
		dataAdder(data, c)
		redirect, templateName, resData, err := widgetService.Process(ctxLogger, widgetName, actionName, data)
		if err != nil {
			return "", common.DefaultErrorRedirect(err.Error())
		}
		if redirect != "" {
			return "", redirect
		}

		if updateData(ctxLogger, data, resData) {
			return templateName, ""
		}
		return "", common.DefaultErrorRedirect(common.ErrTechnical.Error())
	})
}

func updateData(ctxLogger otelzap.LoggerWithCtx, data gin.H, resData []byte) bool {
	var newData gin.H
	if err := json.Unmarshal(resData, &newData); err != nil {
		ctxLogger.Error("Failed to unmarshal json from remote widget", zap.Error(err))
		return false
	}
	for key, value := range newData {
		data[key] = value
	}
	return true
}
