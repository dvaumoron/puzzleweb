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
package builder

import (
	"github.com/dvaumoron/puzzleweb"
	"github.com/dvaumoron/puzzleweb/admin"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/dvaumoron/puzzleweb/login"
	"github.com/dvaumoron/puzzleweb/profile"
	"github.com/dvaumoron/puzzleweb/settings"
)

func BuildDefaultSite() (*puzzleweb.Site, *config.GlobalConfig) {
	globalConfig := config.LoadDefault()
	localesManager := locale.NewManager(globalConfig.ExtractLocalesConfig())
	settingsManager := settings.NewManager(globalConfig.ExtractSettingsConfig())

	site := puzzleweb.NewSite(globalConfig.ExtractAuthConfig(), localesManager)

	login.AddLoginPage(site, globalConfig.ExtractLoginConfig(), settingsManager)
	admin.AddAdminPage(site, globalConfig.ExtractAdminConfig())
	profile.AddProfilePage(site, globalConfig.ExtractProfileConfig())
	settings.AddSettingsPage(site, config.ServiceConfig[*settings.SettingsManager]{
		Logger: globalConfig.Logger, Service: settingsManager,
	})

	return site, globalConfig
}
