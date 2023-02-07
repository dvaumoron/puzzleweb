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
	rightclient "github.com/dvaumoron/puzzleweb/admin/client"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/log"
	loginclient "github.com/dvaumoron/puzzleweb/login/client"
	"github.com/dvaumoron/puzzleweb/profile/client"
	"github.com/dvaumoron/puzzleweb/session"
	"github.com/gin-gonic/gin"
)

const userInfoName = "UserInfo"

var errWrongConfirm = errors.New("ErrorWrongConfirmPassword")

type profileWidget struct {
	viewHandler gin.HandlerFunc
	editHandler gin.HandlerFunc
}

var saveHandler = common.CreateRedirect(func(c *gin.Context) string {
	userId := session.GetUserId(c)
	if userId == 0 {
		return common.DefaultErrorRedirect(common.UnknownUserKey)
	}

	desc := c.PostForm(common.UserDescName)
	info := c.PostFormMap(userInfoName)

	picture, err := c.FormFile("picture")
	if err != nil {
		common.LogOriginalError(err)
		return common.DefaultErrorRedirect(common.ErrTechnical.Error())
	}

	if picture != nil {
		var pictureFile multipart.File
		pictureFile, err = picture.Open()
		if err != nil {
			common.LogOriginalError(err)
			return common.DefaultErrorRedirect(common.ErrTechnical.Error())
		}
		defer pictureFile.Close()

		var pictureData []byte
		pictureData, err = io.ReadAll(pictureFile)
		if err != nil {
			common.LogOriginalError(err)
			return common.DefaultErrorRedirect(common.ErrTechnical.Error())
		}

		err = client.UpdatePicture(userId, pictureData)
	}

	if err == nil {
		err = client.UpdateProfile(userId, desc, info)
	}

	targetBuilder := profileUrlBuilder(c, userId)
	if err != nil {
		common.WriteError(targetBuilder, err.Error())
	}
	return targetBuilder.String()
})

var changeLoginHandler = common.CreateRedirect(func(c *gin.Context) string {
	userId := session.GetUserId(c)
	if userId == 0 {
		return common.DefaultErrorRedirect(common.UnknownUserKey)
	}

	session := session.Get(c)
	oldLogin := session.Load(common.LoginName)
	newLogin := c.PostForm(common.LoginName)
	password := c.PostForm(common.PasswordName)

	err := loginclient.ChangeLogin(userId, oldLogin, newLogin, password)

	targetBuilder := profileUrlBuilder(c, userId)
	if err != nil {
		session.Store(common.LoginName, newLogin)

		common.WriteError(targetBuilder, err.Error())
	}
	return targetBuilder.String()
})

var changePasswordHandler = common.CreateRedirect(func(c *gin.Context) string {
	userId := session.GetUserId(c)
	if userId == 0 {
		return common.DefaultErrorRedirect(common.UnknownUserKey)
	}

	login := session.Get(c).Load(common.LoginName)
	oldPassword := c.PostForm("oldPassword")
	newPassword := c.PostForm("newPassword")
	confirmPassword := c.PostForm("confirmPassword")

	err := errWrongConfirm
	if newPassword == confirmPassword {
		err = loginclient.ChangePassword(userId, login, oldPassword, newPassword)
	}

	targetBuilder := profileUrlBuilder(c, userId)
	if err != nil {
		common.WriteError(targetBuilder, err.Error())
	}
	return targetBuilder.String()
})

func (w *profileWidget) LoadInto(router gin.IRouter) {
	router.GET("/view/:UserId", w.viewHandler)
	router.GET("/edit", w.editHandler)
	router.POST("/save", saveHandler)
	router.POST("/changeLogin", changeLoginHandler)
	router.POST("/changePassword", changePasswordHandler)
}

func AddProfilePage(site *puzzleweb.Site, groupId uint64, args ...string) {
	config.Shared.LoadProfile()

	viewTmpl := "profile/view.html"
	editTmpl := "profile/edit.html"
	switch len(args) {
	default:
		log.Logger.Info("AddProfilePage should be called with 2 to 4 arguments.")
		fallthrough
	case 2:
		if args[1] != "" {
			editTmpl = args[1]
		}
		fallthrough
	case 1:
		if args[0] != "" {
			viewTmpl = args[0]
		}
		fallthrough
	case 0:
	}

	p := puzzleweb.NewHiddenPage("profile")
	p.Widget = &profileWidget{
		viewHandler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			userId, err := common.GetRequestedUserId(c)
			if err != nil {
				common.LogOriginalError(err)
				return "", common.DefaultErrorRedirect(common.ErrTechnical.Error())
			}

			if currentUserId := session.GetUserId(c); userId != currentUserId {
				err = rightclient.AuthQuery(currentUserId, groupId, rightclient.ActionAccess)
				if err != nil {
					return "", common.DefaultErrorRedirect(err.Error())
				}
			}

			profiles, err := client.GetProfiles([]uint64{userId})
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			userProfile := profiles[userId]
			data[common.UserIdName] = userId
			data[common.UserLoginName] = userProfile.Login
			data[common.RegistredAtName] = userProfile.RegistredAt
			data[common.UserDescName] = userProfile.Desc
			data[userInfoName] = userProfile.Info
			return viewTmpl, ""
		}),
		editHandler: puzzleweb.CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			userId := session.GetUserId(c)
			if userId == 0 {
				return "", common.DefaultErrorRedirect(common.UnknownUserKey)
			}

			profiles, err := client.GetProfiles([]uint64{userId})
			if err != nil {
				return "", common.DefaultErrorRedirect(err.Error())
			}

			userProfile := profiles[userId]
			data[common.UserIdName] = userId
			data[common.UserDescName] = userProfile.Desc
			data[userInfoName] = userProfile.Info
			return editTmpl, ""
		}),
	}

	site.AddPage(p)
}

func profileUrlBuilder(c *gin.Context, userId uint64) *strings.Builder {
	targetBuilder := new(strings.Builder)
	targetBuilder.WriteString(common.GetBaseUrl(1, c))
	targetBuilder.WriteString("view/")
	targetBuilder.WriteString(fmt.Sprint(userId))
	return targetBuilder
}
