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
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin/render"
)

// As HTMLProduction from gin, but without the unused Delims.
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

func Load(templatesPath string) render.HTMLRender {
	templatesPath, err := filepath.Abs(templatesPath)
	if err != nil {
		fmt.Println("wrong templatesPath :", err)
		os.Exit(1)
	}
	if last := len(templatesPath) - 1; templatesPath[last] != '/' {
		templatesPath += "/"
	}

	tmpl := template.New("")
	inSize := len(templatesPath)
	err = filepath.WalkDir(templatesPath, func(path string, d fs.DirEntry, err error) error {
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