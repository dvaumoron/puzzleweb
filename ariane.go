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
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
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

func buildAriane(messages map[string]string, splittedPath []string) []PageDesc {
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

func GetLogger(c *gin.Context) otelzap.LoggerWithCtx {
	return getSite(c).logger.Ctx(c.Request.Context())
}

func GetLocalesManager(c *gin.Context) locale.Manager {
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

func (site *Site) extractArianeInfoFromUrl(url string) (Page, []string) {
	current := site.root
	splitted := strings.Split(url, "/")[1:]
	names := make([]string, 0, len(splitted))
	for _, name := range splitted {
		subPage, ok := current.GetSubPage(name)
		if !ok {
			break
		}
		current = subPage
		names = append(names, name)
	}
	return current, names
}

func (p Page) extractSubPageNames(messages map[string]string, url string, c *gin.Context) []PageDesc {
	sw, ok := p.Widget.(*staticWidget)
	if !ok {
		return nil
	}

	pages := sw.subPages
	size := len(pages)
	if size == 0 {
		return nil
	}

	pageDescs := make([]PageDesc, 0, size)
	for _, page := range pages {
		if page.visible {
			name := page.name
			pageDescs = append(pageDescs, makePageDesc(messages, name, url+name))
		}
	}
	return pageDescs
}

func initData(c *gin.Context) gin.H {
	site := getSite(c)
	logger := site.logger.Ctx(c.Request.Context())
	localesManager := site.localesManager
	messages := localesManager.GetMessages(c)
	currentUrl := common.GetCurrentUrl(c)
	page, path := site.extractArianeInfoFromUrl(currentUrl)
	data := gin.H{
		"PageTitle":  getPageTitle(messages, page.name),
		"CurrentUrl": currentUrl,
		"Ariane":     buildAriane(messages, path),
		"SubPages":   page.extractSubPageNames(messages, currentUrl, c),
		"Messages":   messages,
	}
	if errorKey := c.Query("error"); errorKey != "" {
		data[errorMsgName] = messages[errorKey]
	}
	escapedUrl := url.QueryEscape(c.Request.URL.Path)
	if localesManager.GetMultipleLang() {
		data["LangSelectorUrl"] = "/changeLang?Redirect=" + escapedUrl
		data["AllLang"] = localesManager.GetAllLang()
	}
	session := GetSession(logger, c)
	var currentUserId uint64
	if login := session.Load(loginName); login == "" {
		data[loginUrlName] = "/login?Redirect=" + escapedUrl
	} else {
		currentUserId = extractUserIdFromSession(logger, session)
		data[loginName] = login
		data[common.IdName] = currentUserId
		data[loginUrlName] = "/login/logout?Redirect=" + escapedUrl
	}
	data[viewAdminName] = site.authService.AuthQuery(
		logger, currentUserId, adminservice.AdminGroupId, adminservice.ActionAccess,
	) == nil
	for _, adder := range site.adders {
		adder(data, c)
	}
	return data
}
