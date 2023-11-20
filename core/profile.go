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
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/common/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type profileWidget struct {
	defaultHandler        gin.HandlerFunc
	viewHandler           gin.HandlerFunc
	linkHandler           gin.HandlerFunc
	editHandler           gin.HandlerFunc
	saveHandler           gin.HandlerFunc
	changeLoginHandler    gin.HandlerFunc
	changePasswordHandler gin.HandlerFunc
	pictureHandler        gin.HandlerFunc
}

func defaultRedirecter(c *gin.Context) string {
	userId := GetSessionUserId(c)
	if userId == 0 {
		return common.DefaultErrorRedirect(GetLogger(c), unknownUserKey)
	}
	return profileUrlBuilder(userId).String()
}

func (w profileWidget) LoadInto(router gin.IRouter) {
	router.GET("/", w.defaultHandler)
	router.GET("/view/:UserId", w.viewHandler)
	router.GET("/link/:Login", w.linkHandler)
	router.GET("/edit", w.editHandler)
	router.POST("/save", w.saveHandler)
	router.POST("/changeLogin", w.changeLoginHandler)
	router.POST("/changePassword", w.changePasswordHandler)
	router.GET("/picture/:UserId", w.pictureHandler)
}

func newProfilePage(profileConfig config.ProfileConfig) Page {
	profileService := profileConfig.Service
	adminService := profileConfig.AdminService
	loginService := profileConfig.LoginService

	p := MakeHiddenPage("profile")
	p.Widget = profileWidget{
		defaultHandler: common.CreateRedirect(defaultRedirecter),
		viewHandler: CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			logger := GetLogger(c)
			ctx := c.Request.Context()
			viewedUserId := GetRequestedUserId(c)
			if viewedUserId == 0 {
				return "", common.DefaultErrorRedirect(logger, common.ErrorTechnicalKey)
			}

			currentUserId, _ := data[common.UserIdName].(uint64)
			updateRight := viewedUserId == currentUserId
			if !updateRight {
				if err := profileService.ViewRight(ctx, currentUserId); err != nil {
					return "", common.DefaultErrorRedirect(logger, err.Error())
				}
			}

			profiles, err := profileService.GetProfiles(ctx, []uint64{viewedUserId})
			if err != nil {
				return "", common.DefaultErrorRedirect(logger, err.Error())
			}

			userRoles, err := adminService.GetUserRoles(ctx, currentUserId, viewedUserId)
			// ignore ErrNotAuthorized
			if err == common.ErrTechnical {
				return "", common.DefaultErrorRedirect(logger, common.ErrorTechnicalKey)
			}
			if err == nil {
				data["UserRight"] = displayGroups(userRoles)
			}

			userProfile := profiles[viewedUserId]
			data[common.AllowedToUpdateName] = updateRight
			data[common.ViewedUserName] = userProfile
			return "profile/view", ""
		}),
		linkHandler: CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			if viewedUserLogin := c.Param(loginName); viewedUserLogin != "" {
				// use 0, 1 because we just need the first result
				nb, list, err := loginService.ListUsers(c.Request.Context(), 0, 1, viewedUserLogin)
				if err == nil && nb != 0 {
					data[common.ViewedUserName] = list[0]
				}
			}
			return "profile/link", ""
		}),
		editHandler: CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			logger := GetLogger(c)
			userId, _ := data[common.UserIdName].(uint64)
			if userId == 0 {
				return "", common.DefaultErrorRedirect(logger, unknownUserKey)
			}

			profiles, err := profileService.GetProfiles(c.Request.Context(), []uint64{userId})
			if err != nil {
				return "", common.DefaultErrorRedirect(logger, err.Error())
			}

			userProfile := profiles[userId]
			data[common.ViewedUserName] = userProfile
			return "profile/edit", ""
		}),
		saveHandler: common.CreateRedirect(func(c *gin.Context) string {
			logger := GetLogger(c)
			userId := GetSessionUserId(c)
			if userId == 0 {
				return common.DefaultErrorRedirect(logger, unknownUserKey)
			}

			desc := c.PostForm("userDesc")
			info := c.PostFormMap("userInfo")

			picture, err := c.FormFile("picture")
			if err != nil {
				logger.Error("Failed to retrieve picture file", zap.Error(err))
				return common.DefaultErrorRedirect(logger, common.ErrorTechnicalKey)
			}

			ctx := c.Request.Context()
			if picture != nil {
				var pictureFile multipart.File
				pictureFile, err = picture.Open()
				if err != nil {
					logger.Error("Failed to open retrieve picture file ", zap.Error(err))
					return common.DefaultErrorRedirect(logger, common.ErrorTechnicalKey)
				}
				defer pictureFile.Close()

				var pictureData []byte
				pictureData, err = io.ReadAll(pictureFile)
				if err != nil {
					logger.Error("Failed to read picture file ", zap.Error(err))
					return common.DefaultErrorRedirect(logger, common.ErrorTechnicalKey)
				}

				err = profileService.UpdatePicture(ctx, userId, pictureData)
			}

			if err == nil {
				err = profileService.UpdateProfile(ctx, userId, desc, info)
			}

			targetBuilder := profileUrlBuilder(userId)
			if err != nil {
				common.WriteError(targetBuilder, logger, err.Error())
			}
			return targetBuilder.String()
		}),
		changeLoginHandler: common.CreateRedirect(func(c *gin.Context) string {
			logger := GetLogger(c)
			session := GetSession(c)
			userId := GetSessionUserId(c)
			if userId == 0 {
				return common.DefaultErrorRedirect(logger, unknownUserKey)
			}

			oldLogin := session.Load(loginName)
			newLogin := c.PostForm(loginName)
			password := c.PostForm(passwordName)

			err := common.ErrEmptyLogin
			if newLogin != "" {
				err = loginService.ChangeLogin(c.Request.Context(), userId, oldLogin, newLogin, password)
			}

			targetBuilder := profileUrlBuilder(userId)
			if err == nil {
				session.Store(loginName, newLogin)
			} else {
				common.WriteError(targetBuilder, logger, err.Error())
			}
			return targetBuilder.String()
		}),
		changePasswordHandler: common.CreateRedirect(func(c *gin.Context) string {
			logger := GetLogger(c)
			session := GetSession(c)
			userId := GetSessionUserId(c)
			if userId == 0 {
				return common.DefaultErrorRedirect(logger, unknownUserKey)
			}

			login := session.Load(loginName)
			oldPassword := c.PostForm("oldPassword")
			newPassword := c.PostForm("newPassword")
			confirmPassword := c.PostForm(confirmPasswordName)

			err := common.ErrEmptyPassword
			if newPassword != "" {
				err = common.ErrWrongConfirm
				if newPassword == confirmPassword {
					err = loginService.ChangePassword(c.Request.Context(), userId, login, oldPassword, newPassword)
				}
			}

			targetBuilder := profileUrlBuilder(userId)
			if err != nil {
				common.WriteError(targetBuilder, logger, err.Error())
			}
			return targetBuilder.String()
		}),
		pictureHandler: func(c *gin.Context) {
			userId := GetRequestedUserId(c)
			if userId == 0 {
				c.AbortWithStatus(http.StatusNotFound)
				return
			}

			data := profileService.GetPicture(c.Request.Context(), userId)
			c.Data(http.StatusOK, http.DetectContentType(data), data)
		},
	}

	return p
}

func GetRequestedUserId(c *gin.Context) uint64 {
	userId, err := strconv.ParseUint(c.Param(userIdName), 10, 64)
	if err != nil {
		GetLogger(c).Warn("Failed to parse userId from request", zap.Error(err))
	}
	return userId
}

func profileUrlBuilder(userId uint64) *strings.Builder {
	targetBuilder := new(strings.Builder)
	targetBuilder.WriteString("/profile/view/")
	targetBuilder.WriteString(strconv.FormatUint(userId, 10))
	return targetBuilder
}
