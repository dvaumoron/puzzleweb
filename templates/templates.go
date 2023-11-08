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

package templates

import (
	"context"
	"net/http"

	"github.com/dvaumoron/puzzleweb/config"
	templateservice "github.com/dvaumoron/puzzleweb/templates/service"
	"github.com/gin-gonic/gin/render"
)

type ContextAndData struct {
	Ctx  context.Context
	Data any
}

// match Render interface from gin.
type remoteHTML struct {
	Service      templateservice.TemplateService
	ctx          context.Context
	templateName string
	data         any
}

func (r remoteHTML) Render(w http.ResponseWriter) error {
	r.WriteContentType(w)
	content, err := r.Service.Render(r.ctx, r.templateName, r.data)
	if err != nil {
		return err
	}
	_, err = w.Write(content)
	return err
}

const contentTypeName = "Content-Type"

var htmlContentType = []string{"text/html; charset=utf-8"}

// Writes HTML ContentType.
func (r remoteHTML) WriteContentType(w http.ResponseWriter) {
	header := w.Header()
	if val := header[contentTypeName]; len(val) == 0 {
		header[contentTypeName] = htmlContentType
	}
}

// match HTMLRender interface from gin.
type remoteHTMLRender struct {
	Service templateservice.TemplateService
}

func (r remoteHTMLRender) Instance(name string, dataWithCtx any) render.Render {
	ctxData := dataWithCtx.(ContextAndData)
	return remoteHTML{Service: r.Service, ctx: ctxData.Ctx, templateName: name, data: ctxData.Data}
}

func NewServiceRender(templateConfig config.TemplateConfig) render.HTMLRender {
	return remoteHTMLRender{Service: templateConfig.Service}
}
