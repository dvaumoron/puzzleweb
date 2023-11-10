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
	"context"
	"errors"
	"strings"

	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/common/config"
	"github.com/dvaumoron/puzzleweb/locale"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

const settingsName = "Settings"

var errWrongLang = errors.New(common.WrongLangKey)

type SettingsManager struct {
	config.SettingsConfig
	InitSettings  func(*gin.Context) map[string]string
	CheckSettings func(map[string]string, *gin.Context) error
}

func NewSettingsManager(settingsConfig config.SettingsConfig) *SettingsManager {
	return &SettingsManager{SettingsConfig: settingsConfig, InitSettings: initSettings, CheckSettings: checkSettings}
}

func initSettings(c *gin.Context) map[string]string {
	return map[string]string{locale.LangName: GetLocalesManager(c).GetLang(c)}
}

func checkSettings(settings map[string]string, c *gin.Context) error {
	askedLang := settings[locale.LangName]
	lang := GetLocalesManager(c).SetLangCookie(askedLang, c)
	settings[locale.LangName] = lang
	if lang != askedLang {
		return errWrongLang
	}
	return nil
}

func (m *SettingsManager) Get(ctx context.Context, userId uint64, c *gin.Context) map[string]string {
	userSettings := c.GetStringMapString(settingsName)
	if len(userSettings) != 0 {
		return userSettings
	}

	userSettings, err := m.Service.Get(ctx, userId)
	if err != nil {
		m.LoggerGetter.Logger(ctx).Warn("Failed to retrieve user settings", zap.Error(err))
	}

	if len(userSettings) == 0 {
		userSettings = m.InitSettings(c)
		err = m.Service.Update(ctx, userId, userSettings)
		if err != nil {
			m.LoggerGetter.Logger(ctx).Warn("Failed to create user settings", zap.Error(err))
		}
	}
	c.Set(settingsName, userSettings)
	return userSettings
}

func (m *SettingsManager) Update(ctx context.Context, userId uint64, settings map[string]string) error {
	return m.Service.Update(ctx, userId, settings)
}

type settingsWidget struct {
	editHandler gin.HandlerFunc
	saveHandler gin.HandlerFunc
}

func (w settingsWidget) LoadInto(router gin.IRouter) {
	router.GET("/", w.editHandler)
	router.POST("/save", w.saveHandler)
}

func newSettingsPage(settingsConfig config.ServiceConfig[*SettingsManager]) Page {
	settingsManager := settingsConfig.Service

	p := MakeHiddenPage("settings")
	p.Widget = settingsWidget{
		editHandler: CreateTemplate(func(data gin.H, c *gin.Context) (string, string) {
			logger := GetLogger(c)
			userId, _ := data[common.UserIdName].(uint64)
			if userId == 0 {
				return "", common.DefaultErrorRedirect(logger, unknownUserKey)
			}

			data["Settings"] = settingsManager.Get(c.Request.Context(), userId, c)
			return "settings/edit", ""
		}),
		saveHandler: common.CreateRedirect(func(c *gin.Context) string {
			logger := GetLogger(c)
			userId := GetSessionUserId(c)
			if userId == 0 {
				return common.DefaultErrorRedirect(logger, unknownUserKey)
			}

			settings := c.PostFormMap("settings")
			err := settingsManager.CheckSettings(settings, c)
			if err == nil {
				err = settingsManager.Update(c.Request.Context(), userId, settings)
			}

			var targetBuilder strings.Builder
			targetBuilder.WriteString("/settings")
			if err != nil {
				common.WriteError(&targetBuilder, logger, err.Error())
			}
			return targetBuilder.String()
		}),
	}
	return p
}
