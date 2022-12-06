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
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/dvaumoron/puzzleweb/log"
	"github.com/dvaumoron/puzzleweb/session/client"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type InitSettingsFunc func(*gin.Context) map[string]string

var InitSettings InitSettingsFunc = initSettings

func initSettings(c *gin.Context) map[string]string {
	return map[string]string{
		locale.LangName: locale.GetLang(c),
	}
}

func Get(userId uint64, c *gin.Context) map[string]string {
	const settingsName = "settings"
	userSettings := c.GetStringMapString(settingsName)
	if len(userSettings) == 0 {
		var err error
		userSettings, err = client.GetSettings(userId)
		if err == nil {
			if len(userSettings) == 0 {
				userSettings = InitSettings(c)
				err = Update(userId, userSettings)
				if err != nil {
					log.Logger.Warn("Failed to create user settings.",
						zap.Error(err),
					)
				}
			}
		} else {
			log.Logger.Warn("Failed to retrieve user settings.",
				zap.Error(err),
			)
			userSettings = InitSettings(c)
		}
		c.Set(settingsName, userSettings)
	}
	return userSettings
}

func Update(id uint64, settings map[string]string) error {
	return client.UpdateSettings(id, settings)
}
