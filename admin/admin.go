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
	"fmt"
	"strconv"
	"strings"

	pb "github.com/dvaumoron/puzzlerightservice"
	"github.com/dvaumoron/puzzleweb"
	"github.com/dvaumoron/puzzleweb/admin/client"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/dvaumoron/puzzleweb/log"
	loginclient "github.com/dvaumoron/puzzleweb/login/client"
	profileclient "github.com/dvaumoron/puzzleweb/profile/client"
	"github.com/dvaumoron/puzzleweb/session"
	"github.com/gin-gonic/gin"
)

const RoleNameName = "RoleName"
const GroupName = "Group"
const UsersName = "Users"

type UserDisplay struct {
	Id           string
	Login        string
	BaseUrl      string
	ViewLinkName string
}

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
		// TODO retrieve roles from post
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
	adminId := session.GetUserId(c)
	userId, err := strconv.ParseUint(c.Param(common.UserIdName), 10, 64)
	if err == nil {
		// an empty slice delete the user right
		// only the first service call do a right check
		err = client.UpdateUser(adminId, userId, []*client.Role{})
		if err == nil {
			err = profileclient.Delete(userId)
			if err == nil {
				err = loginclient.DeleteUser(userId)
			}
		}
	}

	var targetBuilder strings.Builder
	targetBuilder.WriteString(common.GetBaseUrl(3, c))
	targetBuilder.WriteString("user/list")
	if err != nil {
		common.WriteError(&targetBuilder, err.Error(), c)
	}
	return targetBuilder.String()
})

var saveRoleHanler = common.CreateRedirect(func(c *gin.Context) string {
	adminId := session.GetUserId(c)
	roleName := c.Param(RoleNameName)
	group := c.Param(GroupName)
	actions := make([]pb.RightAction, 0, 4)
	for _, actionStr := range c.PostFormArray("actions") {
		var action pb.RightAction
		switch actionStr {
		case "access":
			action = client.ActionAccess
		case "create":
			action = client.ActionCreate
		case "update":
			action = client.ActionUpdate
		case "delete":
			action = client.ActionDelete
		}
		actions = append(actions, action)
	}
	role := &client.Role{Name: roleName, Group: group, Actions: actions}
	err := client.UpdateRole(adminId, role)

	var targetBuilder strings.Builder
	targetBuilder.WriteString(common.GetBaseUrl(4, c))
	targetBuilder.WriteString("role/list")
	if err != nil {
		common.WriteError(&targetBuilder, err.Error(), c)
	}
	return targetBuilder.String()
})

func (w *adminWidget) LoadInto(router gin.IRouter) {
	router.GET("/", w.displayHanler)
	router.GET("/user/list", w.listUserHanler)
	router.GET("/user/view/:UserId", w.viewUserHanler)
	router.GET("/user/edit/:UserId", w.editUserHanler)
	router.POST("/user/save/:UserId", saveUserHanler)
	router.GET("/user/delete/:UserId", deleteUserHanler)
	router.GET("/role/list", w.listRoleHanler)
	router.GET("/role/edit/:RoleName/:Group", w.editRoleHanler)
	router.POST("/role/save/:RoleName/:Group", saveRoleHanler)
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
	p.Widget = &adminWidget{
		displayHanler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			err := client.AuthQuery(session.GetUserId(c), client.AdminGroupId, client.ActionAccess)

			redirect := ""
			if err == nil {
				data["UserListTitle"] = locale.GetText("user.list", c)
				data["RoleListTitle"] = locale.GetText("role.list", c)
			} else {
				redirect = common.DefaultErrorRedirect(err.Error(), c)
			}
			return indexTmpl, redirect
		}),
		listUserHanler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			adminId := session.GetUserId(c)
			pageNumber, _ := strconv.ParseUint(c.Query("pageNumber"), 10, 64)
			pageSize, parseErr := strconv.ParseUint(c.Query("pageSize"), 10, 64)
			if parseErr != nil {
				pageSize = config.PageSize
			}
			filter := c.Query("filter")

			err := client.AuthQuery(adminId, client.AdminGroupId, client.ActionAccess)

			if err == nil {
				var total uint64
				var users []*loginclient.User

				start := pageNumber * pageSize
				end := start + pageSize
				total, users, err = loginclient.GetUsers(start, end, filter)

				data["UserListTitle"] = locale.GetText("user.list", c)
				if err == nil {
					data["Total"] = total
					if size := len(users); size == 0 {
						data[common.ErrorMsgName] = locale.GetText(common.NoElement, c)
						data[UsersName] = users
					} else {
						viewLinkName := locale.GetText("view.link.name", c)

						baseUrl := common.GetBaseUrl(2, c)
						converted := make([]*UserDisplay, 0, size)
						for _, user := range users {
							converted = append(converted, &UserDisplay{
								Id: fmt.Sprint(user.Id), Login: user.Login,
								BaseUrl: baseUrl, ViewLinkName: viewLinkName,
							})
						}
						data[UsersName] = converted
					}
				}
			}

			redirect := ""
			if err != nil {
				redirect = common.DefaultErrorRedirect(err.Error(), c)
			}
			return listUserTmpl, redirect
		}),
	}

	site.AddPage(p)
}
