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
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/gin-gonic/gin"
)

type Widget interface {
	LoadInto(*Site)
}

type Site struct {
	Engine gin.Engine
	Root   PageTree
}

func CreateSite() *Site {
	engine := gin.Default()

	engine.LoadHTMLGlob(config.TemplatePath + "/*.html")

	engine.Static("/static", config.StaticPath)

	engine.Use(sessionCookie, manageSession)

	root := MakePageTree("root")

	engine.Use(initAriane(&root))

	return &Site{Engine: *engine, Root: root}
}

func (site *Site) Run() error {
	return site.Engine.Run(":" + config.Port)
}
