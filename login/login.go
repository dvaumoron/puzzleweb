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
	"github.com/gin-gonic/gin"
)

type loginWidget struct {
	tmplName string
}

const LoginName = "login"
const UserIdName = "userId"

func (w *loginWidget) LoadInto(router gin.IRouter) {
	const redirectName = "redirect"
	const prevUrlWithErrorName = "prevUrlWithError"
	router.GET("/", puzzleweb.CreateTemplateHandler(w.tmplName, func(data gin.H, c *gin.Context) {
		if errorMsg := c.Query("error"); errorMsg != "" {
			data["errorMsg"] = errorMsg
		}

		redirectUrl := c.Query(redirectName)
		if redirectUrl == "" {
			redirectUrl = "/"
		}
		data[redirectName] = redirectUrl

		currentUrl := c.Request.URL
		var errorKey string
		if len(currentUrl.Query()) == 0 {
			errorKey = "?error="
		} else {
			errorKey = "&error="
		}
		data[prevUrlWithErrorName] = currentUrl.String() + errorKey
	}))
	router.POST("/submit", puzzleweb.CreateRedirectHandler(func(c *gin.Context) string {
		login := c.PostForm(LoginName)
		password := c.PostForm("password")
		register := c.PostForm("register") == "true"

		id, success, err := client.VerifyOrRegister(login, password, register)
		var errorMsg string
		if err != nil {
			errorMsg = err.Error()
		} else if !success {
			errorMsg = locale.GetText("wrong.login", c)
		}

		var target string
		if errorMsg == "" {
			session := puzzleweb.GetSession(c)
			session.Store(LoginName, login)
			session.Store(UserIdName, fmt.Sprint(id))
			target = c.PostForm(redirectName)
		} else {
			target = c.PostForm(prevUrlWithErrorName) + url.QueryEscape(errorMsg)
		}
		return target
	}))
	router.GET("/logout", puzzleweb.CreateRedirectHandler(func(c *gin.Context) string {
		session := puzzleweb.GetSession(c)
		session.Delete(LoginName)
		session.Delete(UserIdName)
		target := c.Query(redirectName)
		if target == "" {
			target = "/"
		}
		return target
	}))
}

func wrapInitData(loginUrl string, logoutUrl string, idf puzzleweb.InitDataFunc) puzzleweb.InitDataFunc {
	return func(c *gin.Context) gin.H {
		data := idf(c)
		escapedUrl := url.QueryEscape(c.Request.URL.Path)
		if login := puzzleweb.GetSession(c).Load(LoginName); login == "" {
			data["loginUrl"] = loginUrl + escapedUrl
		} else {
			data[LoginName] = login
			data["logoutUrl"] = logoutUrl + escapedUrl
		}
		return data
	}
}

func AddLoginPage(site *puzzleweb.Site, name string, tmplName string) {
	p := puzzleweb.NewHiddenPage(name)
	p.Widget = &loginWidget{tmplName: tmplName}

	baseUrl := "/" + name
	loginUrl := baseUrl + "?redirect="
	logoutUrl := baseUrl + "/logout?redirect="
	site.InitData = wrapInitData(loginUrl, logoutUrl, site.InitData)

	site.AddPage(p)
}
