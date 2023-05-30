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
const initMsg = "Failed to init remote widget"

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

func MakeRemotePage(pageName string, ctxLogger otelzap.LoggerWithCtx, widgetName string, remoteConfig config.WidgetConfig) puzzleweb.Page {
	widgetService := remoteConfig.Service
	actions, err := widgetService.GetDesc(ctxLogger, widgetName)
	if err != nil {
		ctxLogger.Fatal(initMsg, zap.Error(err))
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
			pathKeys := extractKeysFromPath(actionPath)
			dataAdder := func(data gin.H, c *gin.Context) {
				retrievePathAndSessionData(pathKeys, data, c)
			}
			handler = createHandler(tracer, widgetNameSlash+actionName, widgetName, actionName, dataAdder, widgetService)
		case http.MethodPost:
			pathKeys := extractKeysFromPath(actionPath)
			dataAdder := func(data gin.H, c *gin.Context) {
				data[formKey] = c.PostFormMap(formKey)
				retrievePathAndSessionData(pathKeys, data, c)
			}
			handler = createHandler(tracer, widgetNameSlash+actionName, widgetName, actionName, dataAdder, widgetService)
		case service.RawResult:
			httpMethod = http.MethodGet
			pathKeys := extractKeysFromPath(actionPath)
			handler = func(c *gin.Context) {
				ctxLogger := puzzleweb.GetLogger(c)
				data := gin.H{}
				retrievePathAndSessionData(pathKeys, data, c)
				files := readFiles(c)
				_, _, resData, err := widgetService.Process(ctxLogger, widgetName, actionName, data, files)
				if err != nil {
					c.AbortWithStatus(http.StatusInternalServerError)
					return
				}
				c.Data(http.StatusOK, http.DetectContentType(resData), resData)
			}
		default:
			ctxLogger.Fatal(initMsg, zap.String("unknownActionKind", httpMethod))
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
		if len(part) != 0 && part[0] == ':' {
			key := part[1:]
			keys = append(keys, [2]string{pathKeySlash + key, key})
		}
	}
	return keys
}

func retrievePathAndSessionData(pathKeys [][2]string, data gin.H, c *gin.Context) {
	for _, key := range pathKeys {
		data[key[0]] = c.Param(key[1])
	}
	data[puzzleweb.SessionName] = puzzleweb.GetSession(c).AsMap()
}

func readFiles(c *gin.Context) map[string][]byte {
	fileList := strings.Split(c.Param("fileList"), ",")
	files := map[string][]byte{}
	for _, name := range fileList {
		readFile(name, files, c)
	}
	return files
}

func readFile(name string, files map[string][]byte, c *gin.Context) {
	// TODO
}

func createHandler(tracer trace.Tracer, spanName string, widgetName string, actionName string, dataAdder common.DataAdder, widgetService service.WidgetService) gin.HandlerFunc {
	return puzzleweb.CreateTemplate(tracer, spanName, func(data gin.H, c *gin.Context) (string, string) {
		ctxLogger := puzzleweb.GetLogger(c)
		dataAdder(data, c)
		files := readFiles(c)
		redirect, templateName, resData, err := widgetService.Process(ctxLogger, widgetName, actionName, data, files)
		if err != nil {
			return "", common.DefaultErrorRedirect(err.Error())
		}
		if redirect != "" {
			return "", redirect
		}

		if updateData(ctxLogger, data, resData, c) {
			return templateName, ""
		}
		return "", common.DefaultErrorRedirect(common.ErrorTechnicalKey)
	})
}

func updateData(ctxLogger otelzap.LoggerWithCtx, data gin.H, resData []byte, c *gin.Context) bool {
	var newData gin.H
	if err := json.Unmarshal(resData, &newData); err != nil {
		ctxLogger.Error("Failed to unmarshal json from remote widget", zap.Error(err))
		return false
	}
	for key, value := range newData {
		data[key] = value
	}
	sessionMap, sessionUpdate := newData[puzzleweb.SessionName]
	if sessionUpdate {
		casted, ok := sessionMap.(map[string]string)
		if ok {
			session := puzzleweb.GetSession(c)
			for key, value := range casted {
				session.Store(key, value)
			}
		}
	}
	return true
}
