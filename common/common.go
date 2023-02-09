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
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const RedirectName = "Redirect"
const BaseUrlName = "BaseUrl"
const AllowedToCreateName = "AllowedToCreate"
const AllowedToUpdateName = "AllowedToUpdate"
const AllowedToDeleteName = "AllowedToDelete"

const PasswordName = "Password"

const UserIdName = "UserId"
const LoginName = "Login"         // current connected user
const UserLoginName = "UserLogin" // viewed user
const RegistredAtName = "RegistredAt"
const UserDescName = "UserDesc"

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
	i := len(res) - 1
	var count uint8
	for count < levelToErase {
		i--
		if res[i] == '/' {
			count++
		}
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

func MapToValueSlice[K comparable, V any](objects map[K]V) []V {
	res := make([]V, 0, len(objects))
	for _, object := range objects {
		res = append(res, object)
	}
	return res
}

func GetRequestedUserId(logger *zap.Logger, c *gin.Context) uint64 {
	userId, err := strconv.ParseUint(c.Param(UserIdName), 10, 64)
	if err != nil {
		logger.Warn("Failed to parse userId from request.", zap.Error(err))
	}
	return userId
}

func GetPagination(c *gin.Context, defaultPageSize uint64) (uint64, uint64, uint64, string) {
	pageNumber, _ := strconv.ParseUint(c.Query("pageNumber"), 10, 64)
	if pageNumber == 0 {
		pageNumber = 1
	}
	pageSize, _ := strconv.ParseUint(c.Query("pageSize"), 10, 64)
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	filter := c.Query("filter")

	start := (pageNumber - 1) * pageSize
	end := start + pageSize

	return pageNumber, start, end, filter
}

func InitPagination(data gin.H, filter string, pageNumber uint64, end uint64, total uint64) {
	data["Filter"] = filter
	if pageNumber != 1 {
		data["PreviousPageNumber"] = pageNumber - 1
	}
	if end < total {
		data["NextPageNumber"] = pageNumber + 1
	}
	data["Total"] = total
}
