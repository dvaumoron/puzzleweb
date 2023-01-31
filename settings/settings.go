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
	"github.com/gin-gonic/gin"
)

type settingsWidget struct {
	editHandler gin.HandlerFunc
}

// TODO
var saveHandler gin.HandlerFunc

func (w *settingsWidget) LoadInto(router gin.IRouter) {
	router.GET("/edit/", w.editHandler)
	router.POST("/save/", saveHandler)
}

func AddSettingsPage(site *puzzleweb.Site, args ...string) {
	config.Shared.LoadSettings()

	// TODO
	p := puzzleweb.NewHiddenPage("settings")
	p.Widget = &settingsWidget{}

	site.AddPage(p)
}
