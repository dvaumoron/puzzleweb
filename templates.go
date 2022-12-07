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
package puzzleweb

import (
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/dvaumoron/puzzleweb/config"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
)

var templatesRender puzzleHTMLRender = loadTemplates()

type puzzleHTMLRender struct {
	templates *template.Template
}

func (r puzzleHTMLRender) Instance(name string, data any) render.Render {
	return render.HTML{
		Template: r.templates,
		Name:     name,
		Data:     data,
	}
}

func loadTemplates() puzzleHTMLRender {
	tmpl := template.New("")
	inSize := len(config.TemplatesPath) + 1
	err := filepath.WalkDir(config.TemplatesPath+"/", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			name := path[inSize:]
			if name[len(name)-5:] == ".html" {
				var data []byte
				data, err = os.ReadFile(path)
				if err == nil {
					_, err = tmpl.New(name).Parse(string(data))
				}
			}
		}
		return err
	})

	if err != nil {
		panic(err)
	}
	return puzzleHTMLRender{templates: tmpl}
}

type DataAdder func(gin.H, *gin.Context)

func CreateDirectTemplate(tmplName string, adder DataAdder) gin.HandlerFunc {
	return func(c *gin.Context) {
		data := initData(c)
		adder(data, c)
		c.HTML(http.StatusOK, tmplName, data)
	}
}

type DataRedirecter func(gin.H, *gin.Context) string

func CreateTemplate(redirecter DataRedirecter) gin.HandlerFunc {
	return func(c *gin.Context) {
		data := initData(c)
		c.HTML(http.StatusOK, redirecter(data, c), data)
	}
}
