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

	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/gin-gonic/gin"
)

type PageDesc struct {
	Name string
	Url  string
}

func extractAriane(splittedPath []string) []PageDesc {
	pageDescs := make([]PageDesc, 0, len(splittedPath))
	var urlBuilder strings.Builder
	for _, name := range splittedPath {
		urlBuilder.WriteString("/")
		urlBuilder.WriteString(name)
		pageDescs = append(pageDescs, PageDesc{Name: name, Url: urlBuilder.String()})
	}
	return pageDescs
}

func getSite(c *gin.Context) *Site {
	siteAny, _ := c.Get(siteName)
	return siteAny.(*Site)
}

func initData(c *gin.Context) gin.H {
	site := getSite(c)
	page, path := site.root.extractPageAndPath(c.Request.URL.Path)
	data := gin.H{
		"PageTitle": locale.GetText("page.title."+page.name, c),
		"Ariane":    extractAriane(path),
		"SubPages":  page.extractSubPageNames(),
	}
	for _, adder := range site.adders {
		adder(data, c)
	}
	return data
}
