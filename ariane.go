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
	"net/url"
	"strings"

	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/gin-gonic/gin"
)

const errorMsgName = "ErrorMsg"

type PageDesc struct {
	Name string
	Url  string
}

func makePageDesc(messages map[string]string, name string, url string) PageDesc {
	return PageDesc{Name: getPageTitle(messages, name), Url: url}
}

func getPageTitle(messages map[string]string, name string) string {
	return messages["PageTitle"+locale.CamelCase(name)]
}

func extractAriane(messages map[string]string, splittedPath []string) []PageDesc {
	pageDescs := make([]PageDesc, 0, len(splittedPath))
	var urlBuilder strings.Builder
	for _, name := range splittedPath {
		urlBuilder.WriteString("/")
		urlBuilder.WriteString(name)
		pageDescs = append(pageDescs, makePageDesc(messages, name, urlBuilder.String()))
	}
	return pageDescs
}

func getSite(c *gin.Context) *Site {
	siteUntyped, _ := c.Get(siteName)
	return siteUntyped.(*Site)
}

func GetLocalesManager(c *gin.Context) *locale.LocalesManager {
	return getSite(c).localesManager
}

func GetMessages(c *gin.Context) map[string]string {
	return getSite(c).localesManager.GetMessages(c)
}

func InitNoELementMsg(data gin.H, size int, c *gin.Context) {
	if size == 0 {
		data[errorMsgName] = GetMessages(c)["NoElement"]
	}
}

func initData(c *gin.Context) gin.H {
	site := getSite(c)
	localesManager := site.localesManager
	messages := localesManager.GetMessages(c)
	currentUrl := common.GetCurrentUrl(c)
	page, path := site.root.extractPageAndPath(currentUrl)
	data := gin.H{
		"PageTitle":  getPageTitle(messages, page.name),
		"CurrentUrl": currentUrl,
		"Ariane":     extractAriane(messages, path),
		"SubPages":   page.extractSubPageNames(messages, currentUrl, c),
		"Messages":   messages,
	}
	if errorKey := c.Query("error"); errorKey != "" {
		data[errorMsgName] = messages[errorKey]
	}
	escapedUrl := url.QueryEscape(c.Request.URL.Path)
	if localesManager.MultipleLang {
		data["LangSelectorUrl"] = "/changeLang?Redirect=" + escapedUrl
		data["AllLang"] = localesManager.AllLang
	}
	session := GetSession(c)
	var currentUserId uint64
	if login := session.Load(loginName); login == "" {
		data[loginUrlName] = "/login?Redirect=" + escapedUrl
	} else {
		currentUserId = extractUserIdFromSession(site.logger, session)
		data[loginName] = login
		data[common.IdName] = currentUserId
		data[loginUrlName] = "/login/logout?Redirect=" + escapedUrl
	}
	data[viewAdminName] = site.authService.AuthQuery(
		currentUserId, adminservice.AdminGroupId, adminservice.ActionAccess,
	) == nil
	for _, adder := range site.adders {
		adder(data, c)
	}
	return data
}
