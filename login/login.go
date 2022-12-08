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
	"github.com/dvaumoron/puzzleweb/log"
	"github.com/dvaumoron/puzzleweb/login/client"
	"github.com/dvaumoron/puzzleweb/session"
	"github.com/dvaumoron/puzzleweb/settings"
	"github.com/gin-gonic/gin"
)

type loginWidget struct {
	tmpl string
}

const LoginName = "Login"
const UserIdName = "UserId"
const loginUrlName = "LogintUrl"

func (w *loginWidget) LoadInto(router gin.IRouter) {
	const prevUrlWithErrorName = "PrevUrlWithError"
	router.GET("/", puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
		data[puzzleweb.RedirectName] = c.Query(puzzleweb.RedirectName)

		currentUrl := c.Request.URL
		var errorKey string
		if len(currentUrl.Query()) == 0 {
			errorKey = puzzleweb.QueryError
		} else {
			errorKey = "&error="
		}
		data[prevUrlWithErrorName] = currentUrl.String() + errorKey

		data["LoginLabel"] = locale.GetText("login.label", c)
		data["PasswordLabel"] = locale.GetText("password.label", c)
		data["RegisterLinkName"] = locale.GetText("register.link.name", c)

		// To hide the connection link
		delete(data, loginUrlName)

		return w.tmpl, ""
	}))
	router.POST("/submit", puzzleweb.CreateRedirect(func(c *gin.Context) string {
		login := c.PostForm(LoginName)
		password := c.PostForm("Password")
		register := c.PostForm("Register") == "true"

		userId, success, err := client.VerifyOrRegister(login, password, register)
		errorMsg := ""
		if err != nil {
			errorMsg = err.Error()
		} else if !success {
			errorMsg = locale.GetText("wrong.login", c)
		}

		target := ""
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
	return func(data gin.H, c *gin.Context) {
		escapedUrl := url.QueryEscape(c.Request.URL.Path)
		if login := session.Get(c).Load(LoginName); login == "" {
			data[loginLinkName] = locale.GetText("login.link.name", c)
			data[loginUrlName] = loginUrl + escapedUrl
		} else {
			data["Welcome"] = locale.GetText("welcome", c)
			data[LoginName] = login
			data[loginLinkName] = locale.GetText("logout.link.name", c)
			data[loginUrlName] = logoutUrl + escapedUrl
		}
	}
}

func AddLoginPage(site *puzzleweb.Site, name string, args ...string) {
	tmpl := ""
	if size := len(args); size == 0 {
		tmpl = "login.html"
	} else {
		if tmpl = args[0]; tmpl == "" {
			tmpl = "login.html"
		}
		if size > 1 {
			log.Logger.Info("AddLoginPage should be called with 2 or 3 arguments.")
		}
	}

	p := puzzleweb.NewHiddenPage(name)
	p.Widget = &loginWidget{tmpl: tmpl}

	baseUrl := "/" + name
	loginUrl := baseUrl + "?redirect="
	logoutUrl := baseUrl + "/logout?redirect="
	site.AddDefaultData(loginData(loginUrl, logoutUrl))

	site.AddPage(p)
}
