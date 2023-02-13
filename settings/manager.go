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
package settings

import (
	"github.com/dvaumoron/puzzleweb"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/dvaumoron/puzzleweb/session/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const settingsName = "Settings"

type InitSettingsFunc func(*gin.Context) map[string]string

type SettingsManager struct {
	config.ServiceConfig[service.SessionService]
	InitSettings InitSettingsFunc
}

func NewManager(settingsConfig config.ServiceConfig[service.SessionService]) *SettingsManager {
	return &SettingsManager{ServiceConfig: settingsConfig, InitSettings: initSettings}
}

func initSettings(c *gin.Context) map[string]string {
	return map[string]string{locale.LangName: puzzleweb.GetLocalesManager(c).GetLang(c)}
}

func (m *SettingsManager) Get(userId uint64, c *gin.Context) map[string]string {
	userSettings := c.GetStringMapString(settingsName)
	if len(userSettings) != 0 {
		return userSettings
	}

	userSettings, err := m.Service.Get(userId)
	if err != nil {
		m.Logger.Warn("Failed to retrieve user settings", zap.Error(err))
	}

	if len(userSettings) == 0 {
		userSettings = m.InitSettings(c)
		err = m.Service.Update(userId, userSettings)
		if err != nil {
			m.Logger.Warn("Failed to create user settings", zap.Error(err))
		}
	}
	c.Set(settingsName, userSettings)
	return userSettings
}

func (m *SettingsManager) Update(userId uint64, settings map[string]string) error {
	return m.Service.Update(userId, settings)
}
