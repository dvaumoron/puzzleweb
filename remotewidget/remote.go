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
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/common/config"
	puzzleweb "github.com/dvaumoron/puzzleweb/core"
	widgetservice "github.com/dvaumoron/puzzleweb/remotewidget/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const initMsg = "Failed to init remote widget"

var (
	errSessionCast      = errors.New("cannot cast returned session")
	errSessionFieldCast = errors.New("cannot cast field in returned session")
)

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

func MakeRemotePage(pageName string, initCtx context.Context, remoteConfig config.RemoteWidgetConfig) (puzzleweb.Page, bool) {
	widgetService := remoteConfig.Service
	actions, err := widgetService.GetDesc(initCtx)
	if err != nil {
		remoteConfig.Logger.Error(initMsg, zap.Error(err))
		return puzzleweb.Page{}, false
	}

	handlers := make([]handlerDesc, 0, len(actions))
	for _, action := range actions {
		httpMethod := action.Kind
		pathKeys := extractKeysFromPath(action.Path)
		queryKeys := extractQueryKeys(action.QueryNames)
		var handler gin.HandlerFunc
		switch httpMethod {
		case http.MethodGet, http.MethodHead, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace:
			dataAdder := func(data gin.H, c *gin.Context) {
				retrieveContextData(pathKeys, queryKeys, data, c)
			}
			handler = createHandler(action.Name, dataAdder, widgetService)
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			dataAdder := func(data gin.H, c *gin.Context) {
				data[widgetservice.FormKey] = c.PostFormMap(widgetservice.FormKey)
				retrieveContextData(pathKeys, queryKeys, data, c)
			}
			handler = createHandler(action.Name, dataAdder, widgetService)
		case widgetservice.RawResult:
			httpMethod = http.MethodGet
			handler = createRawHandler(action.Name, pathKeys, queryKeys, widgetService)
		default:
			remoteConfig.Logger.Error(initMsg, zap.String("unknownActionKind", httpMethod))
			return puzzleweb.Page{}, false
		}
		handlers = append(handlers, handlerDesc{httpMethod: httpMethod, path: action.Path, handler: handler})
	}

	p := puzzleweb.MakePage(pageName)
	p.Widget = remoteWidget{handlers: handlers}
	return p, true
}

func extractKeysFromPath(path string) [][2]string {
	splitted := strings.Split(path, "/")
	keys := make([][2]string, 0, len(splitted))
	for _, part := range splitted {
		if len(part) != 0 {
			if firstChar := part[0]; firstChar == ':' || firstChar == '*' {
				key := part[1:]
				keys = append(keys, [2]string{widgetservice.PathKeySlash + key, key})
			}
		}
	}
	return keys
}

func extractQueryKeys(names []string) [][2]string {
	keys := make([][2]string, 0, len(names))
	for _, name := range names {
		if key := strings.TrimSpace(name); len(key) != 0 {
			keys = append(keys, [2]string{widgetservice.QueryKeySlash + key, key})
		}
	}
	return keys
}

func retrieveContextData(pathKeys [][2]string, queryKeys [][2]string, data gin.H, c *gin.Context) {
	for _, key := range pathKeys {
		data[key[0]] = c.Param(key[1])
	}
	for _, key := range queryKeys {
		data[key[0]] = c.Query(key[1])
	}
	data[puzzleweb.SessionName] = puzzleweb.GetSession(c).AsMap()
}

func readFiles(c *gin.Context) (map[string][]byte, error) {
	files := map[string][]byte{}
	fileList := c.PostForm("fileList")
	if len(fileList) == 0 {
		return files, nil
	}

	for _, name := range strings.Split(fileList, ",") {
		if trimmed := strings.TrimSpace(name); len(trimmed) != 0 {
			if err := readFile(trimmed, files, c); err != nil {
				return nil, err
			}
		}
	}
	return files, nil
}

func readFile(name string, files map[string][]byte, c *gin.Context) error {
	header, err := c.FormFile(name)
	if err != nil {
		return nil // ignore non existing file here (widget should handle)
	}

	file, err := header.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil || len(fileData) == 0 {
		return err
	}
	files[name] = fileData
	return nil
}

func createHandler(actionName string, dataAdder common.DataAdder, widgetService widgetservice.WidgetService) gin.HandlerFunc {
	return puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
		logger := puzzleweb.GetLogger(c)
		dataAdder(data, c)
		files, err := readFiles(c)
		if err != nil {
			logger.Error("Failed to retrieve post file", zap.Error(err))
			return "", common.DefaultErrorRedirect(logger, common.ErrorTechnicalKey)
		}
		redirect, templateName, resData, err := widgetService.Process(c.Request.Context(), actionName, data, files)
		if err != nil {
			return "", common.DefaultErrorRedirect(logger, err.Error())
		}
		if redirect != "" {
			return "", redirect
		}

		if err = updateDataAndSession(data, resData, c); err != nil {
			logger.Error("Failed to unmarshal json from remote widget", zap.Error(err))
			return "", common.DefaultErrorRedirect(logger, common.ErrorTechnicalKey)
		}
		return templateName, ""
	})
}

func updateDataAndSession(data gin.H, resData []byte, c *gin.Context) error {
	if len(resData) == 0 {
		return nil
	}

	var newData gin.H
	if err := json.Unmarshal(resData, &newData); err != nil {
		return err
	}

	for key, value := range newData {
		data[key] = value
	}

	sessionMap, update := newData[puzzleweb.SessionName]
	if !update {
		return nil
	}

	castedMap, ok := sessionMap.(map[string]any)
	if !ok {
		return errSessionCast
	}

	session := puzzleweb.GetSession(c)
	for key, value := range castedMap {
		valueStr, ok := value.(string)
		if !ok {
			return errSessionFieldCast
		}
		session.Store(key, valueStr)
	}
	return nil
}

func createRawHandler(actionName string, pathKeys [][2]string, queryKeys [][2]string, widgetService widgetservice.WidgetService) gin.HandlerFunc {
	return func(c *gin.Context) {
		data := gin.H{}
		retrieveContextData(pathKeys, queryKeys, data, c)
		_, _, resData, err := widgetService.Process(c.Request.Context(), actionName, data, map[string][]byte{})
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Data(http.StatusOK, http.DetectContentType(resData), resData)
	}
}
