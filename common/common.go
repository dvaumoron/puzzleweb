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
package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const RedirectName = "Redirect"
const UserIdName = "UserId"
const BaseUrlName = "BaseUrl"

type DataAdder func(gin.H, *gin.Context)
type Redirecter func(*gin.Context) string
type TemplateRedirecter func(gin.H, *gin.Context) (string, string)

func GetCurrentUrl(c *gin.Context) string {
	path := c.Request.URL.Path
	if path[len(path)-1] != '/' {
		path += "/"
	}
	return path
}

func GetBaseUrl(levelToErase uint8, c *gin.Context) string {
	res := GetCurrentUrl(c)
	i := len(res) - 2
	var count uint8
	for count < levelToErase {
		if res[i] == '/' {
			count++
		}
		i--
	}
	return res[:i+1]
}

func checkTarget(target string) string {
	if target == "" {
		target = "/"
	}
	return target
}

func CreateRedirect(redirecter Redirecter) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Redirect(http.StatusFound, checkTarget(redirecter(c)))
	}
}

func CreateRedirectString(target string) gin.HandlerFunc {
	target = checkTarget(target)
	return func(c *gin.Context) {
		c.Redirect(http.StatusFound, target)
	}
}
