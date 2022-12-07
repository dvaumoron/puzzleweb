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
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/dvaumoron/puzzleweb/login/client"
	"github.com/dvaumoron/puzzleweb/session"
	"github.com/dvaumoron/puzzleweb/settings"
	"github.com/gin-gonic/gin"
)

type loginWidget struct {
	tmplName string
}

const LoginName = "Login"
const UserIdName = "userId"

func (w *loginWidget) LoadInto(router gin.IRouter) {
	const prevUrlWithErrorName = "prevUrlWithError"
	router.GET("/", puzzleweb.CreateDirectTemplate(w.tmplName, func(data gin.H, c *gin.Context) {
		if errorMsg := c.Query("error"); errorMsg != "" {
			data["errorMsg"] = errorMsg
		}

		data[puzzleweb.RedirectName] = c.Query(puzzleweb.RedirectName)

		currentUrl := c.Request.URL
		var errorKey string
		if len(currentUrl.Query()) == 0 {
			errorKey = "?error="
		} else {
			errorKey = "&error="
		}
		data[prevUrlWithErrorName] = currentUrl.String() + errorKey

		data["loginLabel"] = locale.GetText("login.label", c)
		data["passwordLabel"] = locale.GetText("password.label", c)
	}))
	router.POST("/submit", puzzleweb.CreateRedirect(func(c *gin.Context) string {
		login := c.PostForm(LoginName)
		password := c.PostForm("password")
		register := c.PostForm("register") == "true"

		userId, success, err := client.VerifyOrRegister(login, password, register)
		var errorMsg string
		if err != nil {
			errorMsg = err.Error()
		} else if !success {
			errorMsg = locale.GetText("wrong.login", c)
		}

		var target string
		if errorMsg == "" {
			session := session.Get(c)
			session.Store(LoginName, login)
			session.Store(UserIdName, fmt.Sprint(userId))

			locale.SetLangCookie(c, settings.Get(userId, c)[locale.LangName])

			target = c.PostForm(puzzleweb.RedirectName)
		} else {
			target = c.PostForm(prevUrlWithErrorName) + url.QueryEscape(errorMsg)
		}
		return target
	}))
	router.GET("/logout", puzzleweb.CreateRedirect(func(c *gin.Context) string {
		session := session.Get(c)
		session.Delete(LoginName)
		session.Delete(UserIdName)
		return c.Query(puzzleweb.RedirectName)
	}))
}

func loginData(loginUrl string, logoutUrl string) puzzleweb.DataAdder {
	const loginLinkName = "LoginLinkName"
	const loginUrlName = "LogintUrl"
	return func(data gin.H, c *gin.Context) {
		escapedUrl := url.QueryEscape(c.Request.URL.Path)
		if login := session.Get(c).Load(LoginName); login == "" {
			data[loginLinkName] = locale.GetText("login.link.name", c)
			data[loginUrlName] = loginUrl + escapedUrl
		} else {
			data[LoginName] = login
			data[loginLinkName] = locale.GetText("logout.link.name", c)
			data[loginUrlName] = logoutUrl + escapedUrl
		}
	}
}

func AddLoginPage(site *puzzleweb.Site, name string, tmplName string) {
	p := puzzleweb.NewHiddenPage(name)
	p.Widget = &loginWidget{tmplName: tmplName}

	baseUrl := "/" + name
	loginUrl := baseUrl + "?redirect="
	logoutUrl := baseUrl + "/logout?redirect="
	site.AddDefaultData(loginData(loginUrl, logoutUrl))

	site.AddPage(p)
}
