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

package puzzleweb

import (
	"cmp"
	"errors"
	"slices"
	"strings"

	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/gin-gonic/gin"
)

const (
	roleNameName  = "RoleName"
	groupName     = "Group"
	groupsName    = "Groups"
	viewAdminName = "ViewAdmin"

	accessKey = "AccessLabel"
	createKey = "CreateLabel"
	updateKey = "UpdateLabel"
	deleteKey = "DeleteLabel"
)

var errBadName = errors.New("ErrorBadRoleName")

type GroupDisplay struct {
	Id           uint64
	Name         string
	DisplayName  string
	Roles        []RoleDisplay
	AddableRoles []RoleDisplay
}

func NewGroupDisplay(id uint64, name string) *GroupDisplay {
	return &GroupDisplay{Id: id, Name: name, DisplayName: getGroupDisplayNameKey(name)}
}

type RoleDisplay struct {
	Name    string
	Actions []string
}

func MakeRoleDisplay(role adminservice.Role) RoleDisplay {
	return RoleDisplay{Name: role.Name, Actions: displayActions(role.Actions)}
}

func cmpGroupAsc(a *GroupDisplay, b *GroupDisplay) int {
	return cmp.Compare(a.Id, b.Id)
}

func cmpRoleAsc(a RoleDisplay, b RoleDisplay) int {
	return cmp.Compare(a.Name, b.Name)
}

type adminWidget struct {
	displayHandler    gin.HandlerFunc
	listUserHandler   gin.HandlerFunc
	viewUserHandler   gin.HandlerFunc
	editUserHandler   gin.HandlerFunc
	saveUserHandler   gin.HandlerFunc
	deleteUserHandler gin.HandlerFunc
	listRoleHandler   gin.HandlerFunc
	editRoleHandler   gin.HandlerFunc
	saveRoleHandler   gin.HandlerFunc
}

func (w adminWidget) LoadInto(router gin.IRouter) {
	router.GET("/", w.displayHandler)
	router.GET("/user/list", w.listUserHandler)
	router.GET("/user/view/:UserId", w.viewUserHandler)
	router.GET("/user/edit/:UserId", w.editUserHandler)
	router.POST("/user/save/:UserId", w.saveUserHandler)
	router.GET("/user/delete/:UserId", w.deleteUserHandler)
	router.GET("/role/list", w.listRoleHandler)
	router.GET("/role/edit/:RoleName/:Group", w.editRoleHandler)
	router.POST("/role/save", w.saveRoleHandler)
}

func newAdminPage(adminConfig config.AdminConfig) Page {
	adminService := adminConfig.Service
	userService := adminConfig.UserService
	profileService := adminConfig.ProfileService
	defaultPageSize := adminConfig.PageSize

	p := MakeHiddenPage("admin")
	p.Widget = adminWidget{
		displayHandler: CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			viewAdmin, _ := data[viewAdminName].(bool)
			if !viewAdmin {
				return "", common.DefaultErrorRedirect(GetLogger(c), common.ErrorNotAuthorizedKey)
			}
			return "admin/index", ""
		}),
		listUserHandler: CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			logger := GetLogger(c)
			viewAdmin, _ := data[viewAdminName].(bool)
			if !viewAdmin {
				return "", common.DefaultErrorRedirect(logger, common.ErrorNotAuthorizedKey)
			}

			pageNumber, start, end, filter := common.GetPagination(defaultPageSize, c)

			total, users, err := userService.ListUsers(c.Request.Context(), start, end, filter)
			if err != nil {
				return "", common.DefaultErrorRedirect(logger, err.Error())
			}

			common.InitPagination(data, filter, pageNumber, end, total)
			data["Users"] = users
			InitNoELementMsg(data, len(users), c)
			return "admin/user/list", ""
		}),
		viewUserHandler: CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			logger := GetLogger(c)
			adminId, _ := data[common.UserIdName].(uint64)
			userId := GetRequestedUserId(c)
			if userId == 0 {
				return "", common.DefaultErrorRedirect(logger, common.ErrorTechnicalKey)
			}

			ctx := c.Request.Context()
			updateRight, groups, err := adminService.ViewUserRoles(ctx, adminId, userId)
			if err != nil {
				return "", common.DefaultErrorRedirect(logger, err.Error())
			}

			users, err := userService.GetUsers(ctx, []uint64{userId})
			if err != nil {
				return "", common.DefaultErrorRedirect(logger, err.Error())
			}

			user := users[userId]
			data[common.ViewedUserName] = user
			data[common.AllowedToUpdateName] = updateRight
			data[groupsName] = displayGroups(groups)
			return "admin/user/view", ""
		}),
		editUserHandler: CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			logger := GetLogger(c)
			adminId, _ := data[common.UserIdName].(uint64)
			userId := GetRequestedUserId(c)
			if userId == 0 {
				return "", common.DefaultErrorRedirect(logger, common.ErrorTechnicalKey)
			}

			ctx := c.Request.Context()
			userRoles, allRoles, err := adminService.EditUserRoles(ctx, adminId, userId)
			if err != nil {
				return "", common.DefaultErrorRedirect(logger, err.Error())
			}

			userIdToLogin, err := userService.GetUsers(ctx, []uint64{userId})
			if err != nil {
				return "", common.DefaultErrorRedirect(logger, err.Error())
			}

			data[common.ViewedUserName] = userIdToLogin[userId]
			data[groupsName] = displayEditGroups(userRoles, allRoles)
			return "admin/user/edit", ""
		}),
		saveUserHandler: common.CreateRedirect(func(c *gin.Context) string {
			userId := GetRequestedUserId(c)
			err := common.ErrTechnical
			if userId != 0 {
				rolesStr := c.PostFormArray("roles")
				nameToGroup := make(map[string]adminservice.Group, len(rolesStr))
				for _, roleStr := range rolesStr {
					splitted := strings.Split(roleStr, "/")
					if len(splitted) > 1 {
						groupName := splitted[1]
						group, ok := nameToGroup[groupName]
						if !ok {
							group = adminservice.Group{Name: groupName}
						}
						group.Roles = append(group.Roles, adminservice.Role{Name: splitted[0]})
						nameToGroup[groupName] = group
					}
				}
				err = adminService.UpdateUser(c.Request.Context(), GetSessionUserId(c), userId, common.MapToValueSlice(nameToGroup))
			}

			targetBuilder := userListUrlBuilder()
			if err != nil {
				common.WriteError(targetBuilder, GetLogger(c), err.Error())
			}
			return targetBuilder.String()
		}),
		deleteUserHandler: common.CreateRedirect(func(c *gin.Context) string {
			userId := GetRequestedUserId(c)
			err := common.ErrTechnical
			if userId != 0 {
				// an empty slice delete the user right
				// only the first service call do a right check
				ctx := c.Request.Context()
				err = adminService.UpdateUser(ctx, GetSessionUserId(c), userId, []adminservice.Group{})
				if err == nil {
					err = profileService.Delete(ctx, userId)
					if err == nil {
						err = userService.Delete(ctx, userId)
					}
				}
			}

			targetBuilder := userListUrlBuilder()
			if err != nil {
				common.WriteError(targetBuilder, GetLogger(c), err.Error())
			}
			return targetBuilder.String()
		}),
		listRoleHandler: CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			adminId, _ := data[common.UserIdName].(uint64)
			allGroups, err := adminService.GetAllGroups(c.Request.Context(), adminId)
			if err != nil {
				return "", common.DefaultErrorRedirect(GetLogger(c), err.Error())
			}
			data[groupsName] = displayGroups(allGroups)
			return "admin/role/list", ""
		}),
		editRoleHandler: CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			roleName := c.Param(roleNameName)
			group := c.Param(groupName)

			data[roleNameName] = roleName
			data[groupName] = group
			data["GroupDisplayName"] = getGroupDisplayNameKey(group)

			if roleName != "new" {
				adminId, _ := data[common.UserIdName].(uint64)
				actions, err := adminService.GetActions(c.Request.Context(), adminId, roleName, group)
				if err != nil {
					return "", common.DefaultErrorRedirect(GetLogger(c), err.Error())
				}

				actionSet := common.MakeSet(actions)
				setActionChecked(data, actionSet, adminservice.ActionAccess, "Access")
				setActionChecked(data, actionSet, adminservice.ActionCreate, "Create")
				setActionChecked(data, actionSet, adminservice.ActionUpdate, "Update")
				setActionChecked(data, actionSet, adminservice.ActionDelete, "Delete")
			}

			return "admin/role/edit", ""
		}),
		saveRoleHandler: common.CreateRedirect(func(c *gin.Context) string {
			roleName := c.PostForm(roleNameName)
			err := common.ErrBadRoleName
			if roleName != "new" {
				group := c.PostForm(groupName)
				actions := c.PostFormArray("actions")
				err = adminService.UpdateRole(c.Request.Context(), GetSessionUserId(c), roleName, group, actions)
			}

			var targetBuilder strings.Builder
			targetBuilder.WriteString("/admin/role/list")
			if err != nil {
				common.WriteError(&targetBuilder, GetLogger(c), err.Error())
			}
			return targetBuilder.String()
		}),
	}
	return p
}

func getGroupDisplayNameKey(name string) string {
	return "GroupLabel" + locale.CamelCase(name)
}

func displayGroups(groups []adminservice.Group) []*GroupDisplay {
	nameToGroup := map[string]*GroupDisplay{}
	populateGroup(nameToGroup, groups, rolesAppender)
	return sortGroups(nameToGroup)
}

func populateGroup(nameToGroup map[string]*GroupDisplay, groups []adminservice.Group, appender func(*GroupDisplay, adminservice.Role)) {
	for _, group := range groups {
		groupDisplay := nameToGroup[group.Name]
		if groupDisplay == nil {
			groupDisplay = NewGroupDisplay(group.Id, group.Name)
			nameToGroup[group.Name] = groupDisplay
		}
		for _, role := range group.Roles {
			appender(groupDisplay, role)
		}
	}
}

func rolesAppender(group *GroupDisplay, role adminservice.Role) {
	group.Roles = append(group.Roles, MakeRoleDisplay(role))
}

// convert a string slice of codes in a displayable key slice,
// always in the same order : access, create, update, delete
func displayActions(actions []string) []string {
	actionSet := common.MakeSet(actions)
	res := make([]string, 0, len(actionSet))
	if actionSet.Contains(adminservice.ActionAccess) {
		res = append(res, accessKey)
	}
	if actionSet.Contains(adminservice.ActionCreate) {
		res = append(res, createKey)
	}
	if actionSet.Contains(adminservice.ActionUpdate) {
		res = append(res, updateKey)
	}
	if actionSet.Contains(adminservice.ActionDelete) {
		res = append(res, deleteKey)
	}
	return res
}

func sortGroups(nameToGroup map[string]*GroupDisplay) []*GroupDisplay {
	groupRoles := common.MapToValueSlice(nameToGroup)
	slices.SortFunc(groupRoles, cmpGroupAsc)
	for _, group := range groupRoles {
		slices.SortFunc(group.Roles, cmpRoleAsc)
		slices.SortFunc(group.AddableRoles, cmpRoleAsc)
	}
	return groupRoles
}

func displayEditGroups(userRoles []adminservice.Group, allRoles []adminservice.Group) []*GroupDisplay {
	nameToGroup := map[string]*GroupDisplay{}
	populateGroup(nameToGroup, userRoles, rolesAppender)
	populateGroup(nameToGroup, allRoles, addableRolesAppender)
	return sortGroups(nameToGroup)
}

func addableRolesAppender(group *GroupDisplay, role adminservice.Role) {
	// check if the user already have this role
	contains := slices.ContainsFunc(group.Roles, func(roleDisplay RoleDisplay) bool {
		return roleDisplay.Name == role.Name
	})
	// no duplicate
	if !contains {
		group.AddableRoles = append(group.AddableRoles, MakeRoleDisplay(role))
	}
}

func setActionChecked(data gin.H, actionSet common.Set[string], toTest string, name string) {
	if actionSet.Contains(toTest) {
		data[name] = true
	}
}

func userListUrlBuilder() *strings.Builder {
	targetBuilder := new(strings.Builder)
	targetBuilder.WriteString("/admin/user/list")
	return targetBuilder
}
