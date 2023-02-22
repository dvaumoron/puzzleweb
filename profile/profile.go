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
package profile

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	"github.com/dvaumoron/puzzleweb"
	"github.com/dvaumoron/puzzleweb/admin"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/gin-gonic/gin"
)

var errWrongConfirm = errors.New(common.WrongConfirmPassword)

type profileWidget struct {
	viewHandler           gin.HandlerFunc
	editHandler           gin.HandlerFunc
	saveHandler           gin.HandlerFunc
	changeLoginHandler    gin.HandlerFunc
	changePasswordHandler gin.HandlerFunc
}

func (w profileWidget) LoadInto(router gin.IRouter) {
	router.GET("/view/:UserId", w.viewHandler)
	router.GET("/edit", w.editHandler)
	router.POST("/save", w.saveHandler)
	router.POST("/changeLogin", w.changeLoginHandler)
	router.POST("/changePassword", w.changePasswordHandler)
}

func AddProfilePage(site *puzzleweb.Site, profileConfig config.ProfileConfig) {
	logger := profileConfig.Logger
	profileService := profileConfig.Service
	adminService := profileConfig.AdminService
	loginService := profileConfig.LoginService

	ext := profileConfig.Ext
	viewTmpl := "profile/view" + ext
	editTmpl := "profile/edit" + ext

	p := puzzleweb.MakeHiddenPage("profile")
	p.Widget = profileWidget{
		viewHandler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			viewedUserId := common.GetRequestedUserId(logger, c)
			if viewedUserId == 0 {
				return "", common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}

			currentUserId := puzzleweb.GetSessionUserId(c)
			updateRight := viewedUserId == currentUserId
			if !updateRight {
				if err := profileService.ViewRight(currentUserId); err != nil {
					return "", common.DefaultErrorRedirect(err.Error())
				}
			}

			profiles, err := profileService.GetProfiles([]uint64{viewedUserId})
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			roles, err := adminService.GetUserRoles(currentUserId, viewedUserId)
			// ignore ErrNotAuthorized
			if err == common.ErrTechnical {
				return "", common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}
			if err == nil {
				data["UserRight"] = admin.DisplayGroups(roles, puzzleweb.GetMessages(c))
			}

			userProfile := profiles[viewedUserId]
			data[common.AllowedToUpdateName] = updateRight
			data[common.ViewedUserName] = userProfile
			return viewTmpl, ""
		}),
		editHandler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			userId := puzzleweb.GetSessionUserId(c)
			if userId == 0 {
				return "", common.DefaultErrorRedirect(common.UnknownUserKey)
			}

			profiles, err := profileService.GetProfiles([]uint64{userId})
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			userProfile := profiles[userId]
			data[common.ViewedUserName] = userProfile
			return editTmpl, ""
		}),
		saveHandler: common.CreateRedirect(func(c *gin.Context) string {
			userId := puzzleweb.GetSessionUserId(c)
			if userId == 0 {
				return common.DefaultErrorRedirect(common.UnknownUserKey)
			}

			desc := c.PostForm("userDesc")
			info := c.PostFormMap("userInfo")

			picture, err := c.FormFile("picture")
			if err != nil {
				common.LogOriginalError(logger, err)
				return common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}

			if picture != nil {
				var pictureFile multipart.File
				pictureFile, err = picture.Open()
				if err != nil {
					common.LogOriginalError(logger, err)
					return common.DefaultErrorRedirect(common.ErrTechnical.Error())
				}
				defer pictureFile.Close()

				var pictureData []byte
				pictureData, err = io.ReadAll(pictureFile)
				if err != nil {
					common.LogOriginalError(logger, err)
					return common.DefaultErrorRedirect(common.ErrTechnical.Error())
				}

				err = profileService.UpdatePicture(userId, pictureData)
			}

			if err == nil {
				err = profileService.UpdateProfile(userId, desc, info)
			}

			targetBuilder := profileUrlBuilder(userId)
			if err != nil {
				common.WriteError(targetBuilder, err.Error())
			}
			return targetBuilder.String()
		}),
		changeLoginHandler: common.CreateRedirect(func(c *gin.Context) string {
			userId := puzzleweb.GetSessionUserId(c)
			if userId == 0 {
				return common.DefaultErrorRedirect(common.UnknownUserKey)
			}

			session := puzzleweb.GetSession(c)
			oldLogin := session.Load(common.LoginName)
			newLogin := c.PostForm(common.LoginName)
			password := c.PostForm(common.PasswordName)

			err := loginService.ChangeLogin(userId, oldLogin, newLogin, password)

			targetBuilder := profileUrlBuilder(userId)
			if err != nil {
				session.Store(common.LoginName, newLogin)

				common.WriteError(targetBuilder, err.Error())
			}
			return targetBuilder.String()
		}),
		changePasswordHandler: common.CreateRedirect(func(c *gin.Context) string {
			userId := puzzleweb.GetSessionUserId(c)
			if userId == 0 {
				return common.DefaultErrorRedirect(common.UnknownUserKey)
			}

			login := puzzleweb.GetSession(c).Load(common.LoginName)
			oldPassword := c.PostForm("oldPassword")
			newPassword := c.PostForm("newPassword")
			confirmPassword := c.PostForm(common.ConfirmPasswordName)

			err := errWrongConfirm
			if newPassword == confirmPassword {
				// todo check password strength

				err = loginService.ChangePassword(userId, login, oldPassword, newPassword)
			}

			targetBuilder := profileUrlBuilder(userId)
			if err != nil {
				common.WriteError(targetBuilder, err.Error())
			}
			return targetBuilder.String()
		}),
	}

	site.AddPage(p)
}

func profileUrlBuilder(userId uint64) *strings.Builder {
	targetBuilder := new(strings.Builder)
	targetBuilder.WriteString("/profile/view/")
	targetBuilder.WriteString(fmt.Sprint(userId))
	return targetBuilder
}
