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
	"strings"

	"github.com/gin-gonic/gin"
)

const arianeName = "ariane"
const subPagesName = "subPages"

type PageDesc struct {
	Name string
	Url  string
}

func extractAriane(path []string) []PageDesc {
	pageDescs := make([]PageDesc, 0, len(path))
	var url strings.Builder
	for _, name := range path {
		url.WriteString("/")
		url.WriteString(name)
		pageDescs = append(pageDescs, PageDesc{Name: name, Url: url.String()})
	}
	return pageDescs
}

func extractSubPageNames(pt *PageTree) []string {
	pages := pt.SubPages
	var names []string

	if len(pages) == 0 {
		names = make([]string, 0)
	} else {
		names = make([]string, 0, len(pages))
		for _, page := range pages {
			names = append(names, page.Name)
		}
	}

	return names
}

func initAriane(root *PageTree) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, path := extractPageAndPath(root, c.Request.URL.Path)
		c.Set(arianeName, extractAriane(path))
		c.Set(subPagesName, extractSubPageNames(page))
	}
}
