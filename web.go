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
	"net/http"

	"github.com/dvaumoron/puzzleweb/config"
	"github.com/gin-gonic/gin"
)

type Widget interface {
	LoadInto(gin.IRouter)
}

type Site struct {
	Engine  gin.Engine
	Root    PageTree
	Page404 PageTree
}

func CreateSite() *Site {
	engine := gin.Default()

	engine.LoadHTMLGlob(config.TemplatePath + "/*.html")

	engine.Static("/static", config.StaticPath)
	const favicon = "/favicon.ico"
	engine.StaticFile(favicon, config.StaticPath+favicon)

	engine.Use(manageSession)

	root := MakePage("root")
	page404 := MakePage("page404")

	engine.Use(initAriane(&root))

	return &Site{Engine: *engine, Root: root, Page404: page404}
}

func (site *Site) AddPage(page PageTree) {
	site.Root.AddSubPage(page)
}

func (site *Site) Run() error {
	site.Engine.NoRoute()

	return site.Engine.Run(":" + config.Port)
}

func initDisplay(c *gin.Context) gin.H {
	ariane, _ := c.Get(arianeName)
	names, _ := c.Get(subPagesName)
	return gin.H{
		arianeName:   ariane,
		subPagesName: names,
	}
}

type InfoAdder func(gin.H, *gin.Context)

func AddNothing(data gin.H, c *gin.Context) {
}

func CreateHandlerFunc(tmplName string, adder InfoAdder) gin.HandlerFunc {
	return func(c *gin.Context) {
		displayData := initDisplay(c)
		adder(displayData, c)
		c.HTML(http.StatusOK, tmplName, displayData)
	}
}
