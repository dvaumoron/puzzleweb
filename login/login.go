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
package login

import (
	"fmt"
	"net/url"

	"github.com/dvaumoron/puzzleweb"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/dvaumoron/puzzleweb/login/service"
	"github.com/dvaumoron/puzzleweb/settings"
	"github.com/gin-gonic/gin"
)

const loginUrlName = "LoginUrl"
const prevUrlWithErrorName = "PrevUrlWithError"

type loginWidget struct {
	displayHandler gin.HandlerFunc
	submitHandler  gin.HandlerFunc
	logoutHandler  gin.HandlerFunc
}

func (w loginWidget) LoadInto(router gin.IRouter) {
	router.GET("/", w.displayHandler)
	router.POST("/submit", w.submitHandler)
	router.GET("/logout", w.logoutHandler)
}

func AddLoginPage(site *puzzleweb.Site, loginConfig config.ServiceExtConfig[service.LoginService], settingsManager *settings.SettingsManager) {
	loginService := loginConfig.Service

	tmpl := "login" + loginConfig.Ext

	p := puzzleweb.MakeHiddenPage("login")
	p.Widget = loginWidget{
		displayHandler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			data[common.RedirectName] = c.Query(common.RedirectName)

			currentUrl := c.Request.URL
			var errorKey string
			if len(currentUrl.Query()) == 0 {
				errorKey = common.QueryError
			} else {
				errorKey = "&error="
			}
			data[prevUrlWithErrorName] = currentUrl.String() + errorKey

			// To hide the connection link
			delete(data, loginUrlName)

			return tmpl, ""
		}),
		submitHandler: common.CreateRedirect(func(c *gin.Context) string {
			login := c.PostForm(common.LoginName)
			password := c.PostForm(common.PasswordName)
			register := c.PostForm("Register") == "true"

			success := true
			var userId uint64
			var err error
			if register {
				if c.PostForm("ConfirmPassword") != password {
					return c.PostForm(prevUrlWithErrorName) + "WrongConfirmPassword"
				}
				success, userId, err = loginService.Register(login, password)
			} else {
				success, userId, err = loginService.Verify(login, password)
			}

			errorMsg := ""
			if err != nil {
				errorMsg = err.Error()
			} else if !success {
				if register {
					errorMsg = "ExistingLogin"
				} else {
					errorMsg = "WrongLogin"
				}
			}

			if errorMsg != "" {
				return c.PostForm(prevUrlWithErrorName) + url.QueryEscape(errorMsg)
			}

			s := puzzleweb.GetSession(c)
			s.Store(common.LoginName, login)
			s.Store(common.UserIdName, fmt.Sprint(userId))

			puzzleweb.GetLocalesManager(c).SetLangCookie(c, settingsManager.Get(userId, c)[locale.LangName])

			return c.PostForm(common.RedirectName)
		}),
		logoutHandler: common.CreateRedirect(func(c *gin.Context) string {
			s := puzzleweb.GetSession(c)
			s.Delete(common.LoginName)
			s.Delete(common.UserIdName)
			return c.Query(common.RedirectName)
		}),
	}

	site.AddDefaultData(func(data gin.H, c *gin.Context) {
		escapedUrl := url.QueryEscape(c.Request.URL.Path)
		if login := puzzleweb.GetSession(c).Load(common.LoginName); login == "" {
			data[loginUrlName] = "/login?redirect=" + escapedUrl
		} else {
			data[common.LoginName] = login
			data[loginUrlName] = "/login/logout?redirect=" + escapedUrl
		}
	})

	site.AddPage(p)
}
