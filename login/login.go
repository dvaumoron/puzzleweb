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
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/dvaumoron/puzzleweb/log"
	"github.com/dvaumoron/puzzleweb/login/client"
	"github.com/dvaumoron/puzzleweb/session"
	settingsclient "github.com/dvaumoron/puzzleweb/settings/client"
	"github.com/gin-gonic/gin"
)

const loginName = "Login"
const loginUrlName = "LoginUrl"
const prevUrlWithErrorName = "PrevUrlWithError"

type loginWidget struct {
	displayHandler gin.HandlerFunc
}

var submitHandler = common.CreateRedirect(func(c *gin.Context) string {
	login := c.PostForm(loginName)
	password := c.PostForm("Password")
	register := c.PostForm("Register") == "true"

	success, userId, err := client.VerifyOrRegister(login, password, register)
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	} else if !success {
		if register {
			errorMsg = locale.GetText("existing.login", c)
		} else {
			errorMsg = locale.GetText("wrong.login", c)
		}
	}

	if errorMsg != "" {
		return c.PostForm(prevUrlWithErrorName) + url.QueryEscape(errorMsg)
	}

	s := session.Get(c)
	s.Store(loginName, login)
	s.Store(common.UserIdName, fmt.Sprint(userId))

	locale.SetLangCookie(c, settingsclient.Get(userId, c)[locale.LangName])

	return c.PostForm(common.RedirectName)
})

var logoutHandler = common.CreateRedirect(func(c *gin.Context) string {
	s := session.Get(c)
	s.Delete(loginName)
	s.Delete(common.UserIdName)
	return c.Query(common.RedirectName)
})

func (w *loginWidget) LoadInto(router gin.IRouter) {
	router.GET("/", w.displayHandler)
	router.POST("/submit", submitHandler)
	router.GET("/logout", logoutHandler)
}

func loginData(data gin.H, c *gin.Context) {
	escapedUrl := url.QueryEscape(c.Request.URL.Path)
	if login := session.Get(c).Load(loginName); login == "" {
		data[loginUrlName] = "/login?redirect=" + escapedUrl
	} else {
		data[loginName] = login
		data[loginUrlName] = "/login/logout?redirect=" + escapedUrl
	}
}

func AddLoginPage(site *puzzleweb.Site, args ...string) {
	size := len(args)
	tmpl := "login.html"
	if size != 0 && args[0] != "" {
		tmpl = args[0]
	}
	if size > 1 {
		log.Logger.Info("AddLoginPage should be called with 2 or 3 arguments.")
	}

	p := puzzleweb.NewHiddenPage("login")
	p.Widget = &loginWidget{
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
	}

	site.AddDefaultData(loginData)

	site.AddPage(p)
}
