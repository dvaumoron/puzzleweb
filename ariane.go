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

	"github.com/dvaumoron/puzzleweb/errors"
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/gin-gonic/gin"
)

type PageDesc struct {
	Name string
	Url  string
}

func getPageTitle(name string, c *gin.Context) string {
	return locale.GetText("page.title."+name, c)
}

func GetCurrentUrl(c *gin.Context) string {
	path := c.Request.URL.Path
	if path[len(path)-1] != '/' {
		path += "/"
	}
	return path
}

func extractAriane(splittedPath []string, c *gin.Context) []PageDesc {
	pageDescs := make([]PageDesc, 0, len(splittedPath))
	var urlBuilder strings.Builder
	for _, name := range splittedPath {
		urlBuilder.WriteString("/")
		urlBuilder.WriteString(name)
		pageDescs = append(pageDescs,
			PageDesc{
				Name: getPageTitle(name, c),
				Url:  urlBuilder.String(),
			},
		)
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
		"PageTitle":  getPageTitle(page.name, c),
		"CurrentUrl": GetCurrentUrl(c),
		"Ariane":     extractAriane(path, c),
		"SubPages":   page.extractSubPageNames(c),
	}
	if errorMsg := c.Query("error"); errorMsg != "" {
		data[errors.Msg] = errorMsg
	}
	for _, adder := range site.adders {
		adder(data, c)
	}
	return data
}
