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
package forum

import (
	"github.com/dvaumoron/puzzleweb"
	"github.com/gin-gonic/gin"
)

type forumWidget struct {
	listThreadHandler    gin.HandlerFunc
	createThreadHandler  gin.HandlerFunc
	saveThreadHandler    gin.HandlerFunc
	deleteThreadHandler  gin.HandlerFunc
	viewThreadHandler    gin.HandlerFunc
	saveMessageHandler   gin.HandlerFunc
	deleteMessageHandler gin.HandlerFunc
}

func (w *forumWidget) LoadInto(router gin.IRouter) {
	router.GET("/", w.listThreadHandler)
	router.GET("/create/", w.createThreadHandler)
	router.POST("/save/", w.saveThreadHandler)
	router.GET("/delete/:threadId", w.deleteThreadHandler)
	router.GET("/view/:threadId", w.viewThreadHandler)
	router.POST("/message/save/:threadId", w.saveMessageHandler)
	router.GET("/message/delete/:threadId/:messageId", w.deleteMessageHandler)
}

func NewForumPage(forumName string, groupId uint64, forumId uint64, args ...string) *puzzleweb.Page {
	// TODO
	p := puzzleweb.NewPage(forumName)
	p.Widget = &forumWidget{}
	return p
}
