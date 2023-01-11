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
package admin

import (
	"strconv"
	"strings"

	"github.com/dvaumoron/puzzleweb"
	"github.com/dvaumoron/puzzleweb/admin/client"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/dvaumoron/puzzleweb/log"
	"github.com/dvaumoron/puzzleweb/session"
	"github.com/gin-gonic/gin"
)

// TODO
type adminWidget struct {
	displayHanler  gin.HandlerFunc
	listUserHanler gin.HandlerFunc
	viewUserHanler gin.HandlerFunc
	editUserHanler gin.HandlerFunc
	listRoleHanler gin.HandlerFunc
	editRoleHanler gin.HandlerFunc
}

var saveUserHanler = common.CreateRedirect(func(c *gin.Context) string {
	adminId := session.GetUserId(c)
	userId, err := strconv.ParseUint(c.Param(common.UserIdName), 10, 64)
	if err == nil {
		var roles []*client.Role
		// TODO retrieve Role from post
		err = client.UpdateUser(adminId, userId, roles)
	}

	var targetBuilder strings.Builder
	targetBuilder.WriteString(common.GetBaseUrl(3, c))
	targetBuilder.WriteString("user/list")
	if err != nil {
		common.WriteError(&targetBuilder, err.Error(), c)
	}
	return targetBuilder.String()
})

var deleteUserHanler = common.CreateRedirect(func(c *gin.Context) string {
	return ""
})

var saveRoleHanler = common.CreateRedirect(func(c *gin.Context) string {
	return ""
})

func (w *adminWidget) LoadInto(router gin.IRouter) {
	router.GET("/", w.displayHanler)
	router.GET("/user/list", w.listUserHanler)
	router.GET("/user/view/:UserId", w.viewUserHanler)
	router.GET("/user/edit/:UserId", w.editUserHanler)
	router.GET("/user/save/:UserId", saveUserHanler)
	router.GET("/user/delete/:UserId", deleteUserHanler)
	router.GET("/role/list", w.listRoleHanler)
	router.GET("/role/edit/:roleName/:groupId", w.editRoleHanler)
	router.GET("/role/save/:roleName/:groupId", saveRoleHanler)
}

func AddAdminPage(site *puzzleweb.Site, name string, args ...string) {
	indexTmpl := "admin/index.html"
	listUserTmpl := "admin/user/list.html"
	viewUserTmpl := "admin/user/view.html"
	editUserTmpl := "admin/user/edit.html"
	listRoleTmpl := "admin/role/list.html"
	editRoleTmpl := "admin/role/edit.html"
	switch len(args) {
	default:
		log.Logger.Info("AddAdminPage should be called with 2 to 8 arguments.")
		fallthrough
	case 6:
		if args[5] != "" {
			editRoleTmpl = args[5]
		}
		fallthrough
	case 5:
		if args[4] != "" {
			listRoleTmpl = args[4]
		}
		fallthrough
	case 4:
		if args[3] != "" {
			editUserTmpl = args[3]
		}
		fallthrough
	case 3:
		if args[2] != "" {
			viewUserTmpl = args[2]
		}
		fallthrough
	case 2:
		if args[1] != "" {
			listUserTmpl = args[1]
		}
	case 1:
		if args[0] != "" {
			indexTmpl = args[0]
		}
		fallthrough
	case 0:
	}

	p := puzzleweb.NewHiddenPage(name)
	p.Widget = &adminWidget{displayHanler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
		data["UserListTitle"] = locale.GetText("user.list", c)
		data["RoleListTitle"] = locale.GetText("role.list", c)
		return indexTmpl, ""
	})}

	site.AddPage(p)
}
