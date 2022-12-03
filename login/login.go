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
	"net/http"

	"github.com/dvaumoron/puzzleweb"
	"github.com/dvaumoron/puzzleweb/login/client"
	"github.com/gin-gonic/gin"
)

type LoginPage struct {
	tmplName string
}

const LoginName = "login"
const UserIdName = "userId"

func (p *LoginPage) LoadInto(router gin.IRouter) {
	router.GET("/", puzzleweb.CreateHandlerFunc(p.tmplName, func(data gin.H, c *gin.Context) {
		if c.Query("error") != "" {
			data["msg"] = ""
		}
		data["redirect"] = c.Query("redirect")
		url := c.Request.URL
		var errorKey string
		if len(url.Query()) == 0 {
			errorKey = "?error="
		} else {
			errorKey = "&error="
		}
		data["prevError"] = url.String() + errorKey
	}))
	router.POST("/submit", func(c *gin.Context) {
		login := c.PostForm(LoginName)
		password := c.PostForm("password")

		id, success, err := client.Validate(login, password)
		var errorMsg string
		if err != nil {
			errorMsg = err.Error()
		} else if !success {
			errorMsg = ""
		}

		var target string
		if errorMsg == "" {
			session := puzzleweb.GetSession(c)
			session.Store(LoginName, login)
			session.Store(UserIdName, fmt.Sprint(id))
			target = c.PostForm("redirect")
		} else {
			target = c.PostForm("prevError")
		}

		c.Redirect(http.StatusFound, target)
	})
}
