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
	"errors"
	"sort"
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

const userLoginName = "UserLogin"
const roleNameName = "RoleName"
const groupName = "Group"
const groupsName = "Groups"
const usersName = "Users"

const (
	accessKey = "access.label"
	createKey = "create.label"
	updateKey = "update.label"
	deleteKey = "delete.label"
)

var errorBadName = errors.New("error.bad.role.name")

var actionToKey = [4]string{accessKey, createKey, updateKey, deleteKey}

type GroupDisplay struct {
	Id           uint64
	Name         string
	DisplayName  string
	Roles        []*RoleDisplay
	AddableRoles []*RoleDisplay
}

type RoleDisplay struct {
	Name    string
	Actions []string
}

func NewRoleDisplay(role *client.Role, c *gin.Context) *RoleDisplay {
	return &RoleDisplay{Name: role.Name, Actions: displayActions(role.Actions, c)}
}

type sortableGroups []*GroupDisplay

func (s sortableGroups) Len() int {
	return len(s)
}

func (s sortableGroups) Less(i, j int) bool {
	return s[i].Id < s[j].Id
}

func (s sortableGroups) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type sortableRoles []*RoleDisplay

func (s sortableRoles) Len() int {
	return len(s)
}

func (s sortableRoles) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

func (s sortableRoles) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

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
		rolesStr := c.PostFormArray("roles")
		roles := make([]*client.Role, 0, len(rolesStr))
		for _, roleStr := range rolesStr {
			splitted := strings.Split(roleStr, "/")
			if len(splitted) > 1 {
				roles = append(roles, &client.Role{
					Name: splitted[0], Group: splitted[1],
				})
			}
		}
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
	roleName := c.PostForm(roleNameName)
	err := errorBadName
	if roleName != "new" {
		group := c.PostForm(groupName)
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
		err = client.UpdateRole(adminId, &client.Role{Name: roleName, Group: group, Actions: actions})
	}
	var targetBuilder strings.Builder
	targetBuilder.WriteString(common.GetBaseUrl(2, c))
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
	router.POST("/role/save", saveRoleHanler)
}

func AddAdminPage(site *puzzleweb.Site, args ...string) {
	indexTmpl := "admin/index.html"
	listUserTmpl := "admin/user/list.html"
	viewUserTmpl := "admin/user/view.html"
	editUserTmpl := "admin/user/edit.html"
	listRoleTmpl := "admin/role/list.html"
	editRoleTmpl := "admin/role/edit.html"
	switch len(args) {
	default:
		log.Logger.Info("AddAdminPage should be called with 1 to 7 arguments.")
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

	p := puzzleweb.NewHiddenPage("admin")
	p.Widget = &adminWidget{
		displayHanler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			err := client.AuthQuery(session.GetUserId(c), client.AdminGroupId, client.ActionAccess)

			redirect := ""
			if err != nil {
				redirect = common.DefaultErrorRedirect(err.Error(), c)
			}
			return indexTmpl, redirect
		}),
		listUserHanler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			adminId := session.GetUserId(c)
			pageNumber, _ := strconv.ParseUint(c.Query("pageNumber"), 10, 64)
			pageSize, _ := strconv.ParseUint(c.Query("pageSize"), 10, 64)
			if pageSize == 0 {
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

				if err == nil {
					data["Total"] = total
					data[usersName] = users
					data[common.BaseUrlName] = common.GetBaseUrl(1, c)
					if size := len(users); size == 0 {
						data[common.ErrorMsgName] = locale.GetText(common.NoElementKey, c)
					}
				}
			}

			redirect := ""
			if err != nil {
				redirect = common.DefaultErrorRedirect(err.Error(), c)
			}
			return listUserTmpl, redirect
		}),
		viewUserHanler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			adminId := session.GetUserId(c)
			userId, err := strconv.ParseUint(c.Param(common.UserIdName), 10, 64)
			if err == nil {
				var roles []*client.Role
				roles, err = client.GetUserRoles(adminId, userId)
				if err == nil {
					var userIdToLogin map[uint64]string
					userIdToLogin, err = loginclient.GetLogins([]uint64{userId})
					data[common.BaseUrlName] = common.GetBaseUrl(2, c)
					data[common.UserIdName] = userId
					data[userLoginName] = userIdToLogin[userId]
					data["IsAdmin"] = adminId != userId
					data[groupsName] = displayGroups(roles, c)
				}
			}

			redirect := ""
			if err != nil {
				redirect = common.DefaultErrorRedirect(err.Error(), c)
			}
			return viewUserTmpl, redirect
		}),
		editUserHanler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			adminId := session.GetUserId(c)
			userId, err := strconv.ParseUint(c.Param(common.UserIdName), 10, 64)
			if err == nil {
				var allRoles []*client.Role
				allRoles, err = client.GetAllRoles(adminId)
				if err == nil {
					var userRoles []*client.Role
					userRoles, err = client.GetUserRoles(adminId, userId)
					if err == nil {
						var userIdToLogin map[uint64]string
						userIdToLogin, err = loginclient.GetLogins([]uint64{userId})
						data[common.BaseUrlName] = common.GetBaseUrl(2, c)
						data[common.UserIdName] = userId
						data[userLoginName] = userIdToLogin[userId]
						data[groupsName] = displayEditGroups(userRoles, allRoles, c)
					}
				}
			}

			redirect := ""
			if err != nil {
				redirect = common.DefaultErrorRedirect(err.Error(), c)
			}
			return editUserTmpl, redirect
		}),
		listRoleHanler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			adminId := session.GetUserId(c)
			allRoles, err := client.GetAllRoles(adminId)
			if err == nil {
				data[common.BaseUrlName] = common.GetBaseUrl(1, c)
				data[groupsName] = displayGroups(allRoles, c)
			}

			redirect := ""
			if err != nil {
				redirect = common.DefaultErrorRedirect(err.Error(), c)
			}
			return listRoleTmpl, redirect
		}),
		editRoleHanler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			adminId := session.GetUserId(c)
			roleName := c.PostForm(roleNameName)
			group := c.PostForm(groupName)

			data[common.BaseUrlName] = common.GetBaseUrl(1, c)
			data[roleNameName] = roleName
			data[groupName] = group

			var err error
			if roleName != "new" {
				var actions []pb.RightAction
				actions, err = client.GetActions(adminId, roleName, group)
				if err == nil {
					actionSet := common.MakeSet(actions...)
					setActionChecked(data, actionSet, client.ActionAccess, "Access")
					setActionChecked(data, actionSet, client.ActionCreate, "Create")
					setActionChecked(data, actionSet, client.ActionUpdate, "Update")
					setActionChecked(data, actionSet, client.ActionDelete, "Delete")
				}
			}

			redirect := ""
			if err != nil {
				redirect = common.DefaultErrorRedirect(err.Error(), c)
			}
			return editRoleTmpl, redirect
		}),
	}

	site.AddPage(p)
}

func displayGroups(roles []*client.Role, c *gin.Context) []*GroupDisplay {
	nameToGroup := map[string]*GroupDisplay{}
	populateGroup(nameToGroup, roles, c, rolesAppender)
	return sortGroups(nameToGroup)
}

func populateGroup(nameToGroup map[string]*GroupDisplay, roles []*client.Role, c *gin.Context, appender func(*GroupDisplay, *client.Role, *gin.Context)) {
	for _, role := range roles {
		groupName := role.Group
		group := nameToGroup[groupName]
		if group == nil {
			group = &GroupDisplay{
				Id:          client.GetGroupId(groupName),
				Name:        groupName,
				DisplayName: locale.GetText("group.label."+groupName, c),
			}
			nameToGroup[groupName] = group
		}
		appender(group, role, c)
	}
}

func rolesAppender(group *GroupDisplay, role *client.Role, c *gin.Context) {
	group.Roles = append(group.Roles, NewRoleDisplay(role, c))
}

// convert a RightAction slice in a displayable string slice,
// always in the same order : access, create, update, delete
func displayActions(actions []pb.RightAction, c *gin.Context) []string {
	actionSet := common.MakeSet(actions...)
	res := make([]string, len(actions))
	if actionSet.Contains(client.ActionAccess) {
		res = append(res, locale.GetText(accessKey, c))
	}
	if actionSet.Contains(client.ActionCreate) {
		res = append(res, locale.GetText(createKey, c))
	}
	if actionSet.Contains(client.ActionUpdate) {
		res = append(res, locale.GetText(updateKey, c))
	}
	if actionSet.Contains(client.ActionDelete) {
		res = append(res, locale.GetText(deleteKey, c))
	}
	return res
}

func sortGroups(nameToGroup map[string]*GroupDisplay) []*GroupDisplay {
	groupRoles := common.MapToValueSlice(nameToGroup)
	sort.Sort(sortableGroups(groupRoles))
	for _, group := range groupRoles {
		sort.Sort(sortableRoles(group.Roles))
		sort.Sort(sortableRoles(group.AddableRoles))
	}
	return groupRoles
}

func displayEditGroups(userRoles []*client.Role, allRoles []*client.Role, c *gin.Context) []*GroupDisplay {
	nameToGroup := map[string]*GroupDisplay{}
	populateGroup(nameToGroup, userRoles, c, rolesAppender)
	populateGroup(nameToGroup, allRoles, c, addableRolesAppender)
	return sortGroups(nameToGroup)
}

func addableRolesAppender(group *GroupDisplay, role *client.Role, c *gin.Context) {
	group.AddableRoles = append(group.AddableRoles, NewRoleDisplay(role, c))
}

func setActionChecked(data gin.H, actionSet common.Set[pb.RightAction], toTest pb.RightAction, name string) {
	if actionSet.Contains(toTest) {
		data[name] = true
	}
}
