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
	"strings"

	pb "github.com/dvaumoron/puzzlerightservice"
	"github.com/dvaumoron/puzzleweb"
	"github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/dvaumoron/puzzleweb/session"
	"github.com/gin-gonic/gin"
)

const roleNameName = "RoleName"
const groupName = "Group"
const groupsName = "Groups"
const viewAdminName = "ViewAdmin"

const (
	accessKey = "AccessLabel"
	createKey = "CreateLabel"
	updateKey = "UpdateLabel"
	deleteKey = "DeleteLabel"
)

var errBadName = errors.New("ErrorBadRoleName")

var actionToKey = [4]string{accessKey, createKey, updateKey, deleteKey}

type GroupDisplay struct {
	Id           uint64
	Name         string
	DisplayName  string
	Roles        []RoleDisplay
	AddableRoles []RoleDisplay
}

type RoleDisplay struct {
	Name    string
	Actions []string
}

func MakeRoleDisplay(role service.Role, c *gin.Context) RoleDisplay {
	return RoleDisplay{Name: role.Name, Actions: displayActions(role.Actions, c)}
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

type sortableRoles []RoleDisplay

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

func AddAdminPage(site *puzzleweb.Site, adminConfig config.AdminConfig, args ...string) {
	logger := adminConfig.Logger
	adminService := adminConfig.Service
	userService := adminConfig.UserService
	profileService := adminConfig.ProfileService
	defaultPageSize := adminConfig.PageSize

	indexTmpl := "admin/index.html"
	listUserTmpl := "admin/user/list.html"
	viewUserTmpl := "admin/user/view.html"
	editUserTmpl := "admin/user/edit.html"
	listRoleTmpl := "admin/role/list.html"
	editRoleTmpl := "admin/role/edit.html"
	switch len(args) {
	default:
		logger.Info("AddAdminPage should be called with 3 to 9 arguments.")
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

	p := puzzleweb.MakeHiddenPage("admin")
	p.Widget = adminWidget{
		displayHandler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			viewAdmin, _ := data[viewAdminName].(bool)
			if !viewAdmin {
				return "", common.DefaultErrorRedirect(common.ErrNotAuthorized.Error())
			}
			return indexTmpl, ""
		}),
		listUserHandler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			viewAdmin, _ := data[viewAdminName].(bool)
			if !viewAdmin {
				return "", common.DefaultErrorRedirect(common.ErrNotAuthorized.Error())
			}

			pageNumber, start, end, filter := common.GetPagination(c, defaultPageSize)

			total, users, err := userService.ListUsers(start, end, filter)
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			common.InitPagination(data, filter, pageNumber, end, total)
			data["Users"] = users
			data[common.BaseUrlName] = common.GetBaseUrl(1, c)
			common.InitNoELementMsg(data, len(users), c)
			return listUserTmpl, ""
		}),
		viewUserHandler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			adminId := session.GetUserId(logger, c)
			userId := common.GetRequestedUserId(logger, c)
			if userId == 0 {
				return "", common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}

			roles, err := adminService.GetUserRoles(adminId, userId)
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			users, err := userService.GetUsers([]uint64{userId})
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			updateRight := adminService.AuthQuery(adminId, service.AdminGroupId, pb.RightAction_UPDATE) == nil

			user := users[userId]
			data[common.BaseUrlName] = common.GetBaseUrl(2, c)
			data[common.UserIdName] = userId
			data[common.UserLoginName] = user.Login
			data[common.RegistredAtName] = user.RegistredAt
			data[common.AllowedToUpdateName] = updateRight
			data[groupsName] = DisplayGroups(roles, c)
			return viewUserTmpl, ""
		}),
		editUserHandler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			adminId := session.GetUserId(logger, c)
			userId := common.GetRequestedUserId(logger, c)
			if userId == 0 {
				return "", common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}

			allRoles, err := adminService.GetAllRoles(adminId)
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			userRoles, err := adminService.GetUserRoles(adminId, userId)
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			userIdToLogin, err := userService.GetUsers([]uint64{userId})
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			data[common.BaseUrlName] = common.GetBaseUrl(2, c)
			data[common.UserIdName] = userId
			data[common.UserLoginName] = userIdToLogin[userId].Login
			data[groupsName] = displayEditGroups(userRoles, allRoles, c)
			return editUserTmpl, ""
		}),
		saveUserHandler: common.CreateRedirect(func(c *gin.Context) string {
			userId := common.GetRequestedUserId(logger, c)
			err := common.ErrTechnical
			if userId != 0 {
				rolesStr := c.PostFormArray("roles")
				roles := make([]service.Role, 0, len(rolesStr))
				for _, roleStr := range rolesStr {
					splitted := strings.Split(roleStr, "/")
					if len(splitted) > 1 {
						roles = append(roles, service.Role{Name: splitted[0], GroupName: splitted[1]})
					}
				}
				err = adminService.UpdateUser(session.GetUserId(logger, c), userId, roles)
			}

			targetBuilder := userListUrlBuilder(c)
			if err != nil {
				common.WriteError(targetBuilder, err.Error())
			}
			return targetBuilder.String()
		}),
		deleteUserHandler: common.CreateRedirect(func(c *gin.Context) string {
			userId := common.GetRequestedUserId(logger, c)
			err := common.ErrTechnical
			if userId != 0 {
				// an empty slice delete the user right
				// only the first service call do a right check
				err = adminService.UpdateUser(session.GetUserId(logger, c), userId, []service.Role{})
				if err == nil {
					err = profileService.Delete(userId)
					if err == nil {
						err = userService.Delete(userId)
					}
				}
			}

			targetBuilder := userListUrlBuilder(c)
			if err != nil {
				common.WriteError(targetBuilder, err.Error())
			}
			return targetBuilder.String()
		}),
		listRoleHandler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			allRoles, err := adminService.GetAllRoles(session.GetUserId(logger, c))
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			data[common.BaseUrlName] = common.GetBaseUrl(1, c)
			data[groupsName] = DisplayGroups(allRoles, c)
			return listRoleTmpl, ""
		}),
		editRoleHandler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			roleName := c.PostForm(roleNameName)
			group := c.PostForm(groupName)

			data[common.BaseUrlName] = common.GetBaseUrl(1, c)
			data[roleNameName] = roleName
			data[groupName] = group

			if roleName != "new" {
				actions, err := adminService.GetActions(session.GetUserId(logger, c), roleName, group)
				if err != nil {
					return "", common.DefaultErrorRedirect(err.Error())
				}

				actionSet := common.MakeSet(actions)
				setActionChecked(data, actionSet, pb.RightAction_ACCESS, "Access")
				setActionChecked(data, actionSet, pb.RightAction_CREATE, "Create")
				setActionChecked(data, actionSet, pb.RightAction_UPDATE, "Update")
				setActionChecked(data, actionSet, pb.RightAction_DELETE, "Delete")
			}

			return editRoleTmpl, ""
		}),
		saveRoleHandler: common.CreateRedirect(func(c *gin.Context) string {
			roleName := c.PostForm(roleNameName)
			err := errBadName
			if roleName != "new" {
				group := c.PostForm(groupName)
				actions := make([]pb.RightAction, 0, 4)
				for _, actionStr := range c.PostFormArray("actions") {
					var action pb.RightAction
					switch actionStr {
					case "access":
						action = pb.RightAction_ACCESS
					case "create":
						action = pb.RightAction_CREATE
					case "update":
						action = pb.RightAction_UPDATE
					case "delete":
						action = pb.RightAction_DELETE
					}
					actions = append(actions, action)
				}
				err = adminService.UpdateRole(session.GetUserId(logger, c), service.Role{
					Name: roleName, GroupName: group, Actions: actions,
				})
			}

			var targetBuilder strings.Builder
			targetBuilder.WriteString(common.GetBaseUrl(1, c))
			targetBuilder.WriteString("list")
			if err != nil {
				common.WriteError(&targetBuilder, err.Error())
			}
			return targetBuilder.String()
		}),
	}

	site.AddDefaultData(func(data gin.H, c *gin.Context) {
		data[viewAdminName] = adminService.AuthQuery(
			session.GetUserId(logger, c), service.AdminGroupId, pb.RightAction_ACCESS,
		) == nil
	})

	site.AddPage(p)
}

func DisplayGroups(roles []service.Role, c *gin.Context) []*GroupDisplay {
	nameToGroup := map[string]*GroupDisplay{}
	populateGroup(nameToGroup, roles, c, rolesAppender)
	return sortGroups(nameToGroup)
}

func populateGroup(nameToGroup map[string]*GroupDisplay, roles []service.Role, c *gin.Context, appender func(*GroupDisplay, service.Role, *gin.Context)) {
	for _, role := range roles {
		groupName := role.GroupName
		group := nameToGroup[groupName]
		if group == nil {
			group = &GroupDisplay{
				Id: role.GroupId, Name: groupName,
				DisplayName: locale.GetText("GroupLabel"+locale.CamelCase(groupName), c),
			}
			nameToGroup[groupName] = group
		}
		appender(group, role, c)
	}
}

func rolesAppender(group *GroupDisplay, role service.Role, c *gin.Context) {
	group.Roles = append(group.Roles, MakeRoleDisplay(role, c))
}

// convert a RightAction slice in a displayable string slice,
// always in the same order : access, create, update, delete
func displayActions(actions []pb.RightAction, c *gin.Context) []string {
	actionSet := common.MakeSet(actions)
	res := make([]string, len(actions))
	if actionSet.Contains(pb.RightAction_ACCESS) {
		res = append(res, locale.GetText(accessKey, c))
	}
	if actionSet.Contains(pb.RightAction_CREATE) {
		res = append(res, locale.GetText(createKey, c))
	}
	if actionSet.Contains(pb.RightAction_UPDATE) {
		res = append(res, locale.GetText(updateKey, c))
	}
	if actionSet.Contains(pb.RightAction_DELETE) {
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

func displayEditGroups(userRoles []service.Role, allRoles []service.Role, c *gin.Context) []*GroupDisplay {
	nameToGroup := map[string]*GroupDisplay{}
	populateGroup(nameToGroup, userRoles, c, rolesAppender)
	populateGroup(nameToGroup, allRoles, c, addableRolesAppender)
	return sortGroups(nameToGroup)
}

func addableRolesAppender(group *GroupDisplay, role service.Role, c *gin.Context) {
	group.AddableRoles = append(group.AddableRoles, MakeRoleDisplay(role, c))
}

func setActionChecked(data gin.H, actionSet common.Set[pb.RightAction], toTest pb.RightAction, name string) {
	if actionSet.Contains(toTest) {
		data[name] = true
	}
}

func userListUrlBuilder(c *gin.Context) *strings.Builder {
	targetBuilder := new(strings.Builder)
	// no need to erase and rewrite "user/"
	targetBuilder.WriteString(common.GetBaseUrl(2, c))
	targetBuilder.WriteString("list")
	return targetBuilder
}
