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
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var errEmptyLogin = errors.New(emptyLoginKey)
var errWrongConfirm = errors.New(wrongConfirmPasswordKey)

type profileWidget struct {
	viewHandler           gin.HandlerFunc
	editHandler           gin.HandlerFunc
	saveHandler           gin.HandlerFunc
	changeLoginHandler    gin.HandlerFunc
	changePasswordHandler gin.HandlerFunc
	pictureHandler        gin.HandlerFunc
}

var defaultHandler = common.CreateRedirect(func(c *gin.Context) string {
	userId := GetSessionUserId(c)
	if userId == 0 {
		return common.DefaultErrorRedirect(unknownUserKey)
	}
	return profileUrlBuilder(userId).String()
})

func (w profileWidget) LoadInto(router gin.IRouter) {
	router.GET("/", defaultHandler)
	router.GET("/view/:UserId", w.viewHandler)
	router.GET("/edit", w.editHandler)
	router.POST("/save", w.saveHandler)
	router.POST("/changeLogin", w.changeLoginHandler)
	router.POST("/changePassword", w.changePasswordHandler)
	router.GET("/picture/:UserId", w.pictureHandler)
}

func NewProfilePage(profileConfig config.ProfileConfig) Page {
	logger := profileConfig.Logger
	profileService := profileConfig.Service
	adminService := profileConfig.AdminService
	loginService := profileConfig.LoginService

	ext := profileConfig.Ext
	viewTmpl := "profile/view" + ext
	editTmpl := "profile/edit" + ext

	p := MakeHiddenPage("profile")
	p.Widget = profileWidget{
		viewHandler: CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			viewedUserId := GetRequestedUserId(logger, c)
			if viewedUserId == 0 {
				return "", common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}

			currentUserId, _ := data[common.IdName].(uint64)
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
				data["UserRight"] = DisplayGroups(roles, GetMessages(c))
			}

			userProfile := profiles[viewedUserId]
			data[common.AllowedToUpdateName] = updateRight
			data[common.ViewedUserName] = userProfile
			return viewTmpl, ""
		}),
		editHandler: CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			userId, _ := data[common.IdName].(uint64)
			if userId == 0 {
				return "", common.DefaultErrorRedirect(unknownUserKey)
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
			userId := GetSessionUserId(c)
			if userId == 0 {
				return common.DefaultErrorRedirect(unknownUserKey)
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
			userId := GetSessionUserId(c)
			if userId == 0 {
				return common.DefaultErrorRedirect(unknownUserKey)
			}

			session := GetSession(c)
			oldLogin := session.Load(loginName)
			newLogin := c.PostForm(loginName)
			password := c.PostForm(passwordName)

			var err error
			if newLogin == "" {
				err = errEmptyLogin
			} else {
				err = loginService.ChangeLogin(userId, oldLogin, newLogin, password)
			}

			targetBuilder := profileUrlBuilder(userId)
			if err == nil {
				session.Store(loginName, newLogin)
			} else {
				common.WriteError(targetBuilder, err.Error())
			}
			return targetBuilder.String()
		}),
		changePasswordHandler: common.CreateRedirect(func(c *gin.Context) string {
			session := GetSession(c)
			userId := extractUserIdFromSession(logger, session)
			if userId == 0 {
				return common.DefaultErrorRedirect(unknownUserKey)
			}

			login := session.Load(loginName)
			oldPassword := c.PostForm("oldPassword")
			newPassword := c.PostForm("newPassword")
			confirmPassword := c.PostForm(confirmPasswordName)

			err := errWrongConfirm
			if newPassword == confirmPassword {
				// TODO check password strength

				err = loginService.ChangePassword(userId, login, oldPassword, newPassword)
			}

			targetBuilder := profileUrlBuilder(userId)
			if err != nil {
				common.WriteError(targetBuilder, err.Error())
			}
			return targetBuilder.String()
		}), pictureHandler: func(c *gin.Context) {
			userId := GetRequestedUserId(logger, c)
			if userId == 0 {
				c.AbortWithStatus(http.StatusNotFound)
				return
			}

			data := profileService.GetPicture(userId)
			c.Data(http.StatusOK, http.DetectContentType(data), data)
		},
	}

	return p
}

func GetRequestedUserId(logger *zap.Logger, c *gin.Context) uint64 {
	userId, err := strconv.ParseUint(c.Param(userIdName), 10, 64)
	if err != nil {
		logger.Warn("Failed to parse userId from request", zap.Error(err))
	}
	return userId
}

func profileUrlBuilder(userId uint64) *strings.Builder {
	targetBuilder := new(strings.Builder)
	targetBuilder.WriteString("/profile/view/")
	targetBuilder.WriteString(fmt.Sprint(userId))
	return targetBuilder
}
