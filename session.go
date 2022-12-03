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
	"fmt"
	"net/http"
	"strconv"

	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/sessionclient"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const cookieName = "pw_session_id"

func getSessionId(c *gin.Context) uint64 {
	var sessionId uint64
	cookie, err := c.Cookie(cookieName)
	if err == nil {
		sessionId, err = strconv.ParseUint(cookie, 10, 0)
		if err != nil {
			sessionId = generateSessionCookie(c)
		}
	} else {
		sessionId = generateSessionCookie(c)
	}

	return sessionId
}

func generateSessionCookie(c *gin.Context) uint64 {
	sessionId, err := sessionclient.Generate()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}

	c.SetCookie(cookieName, fmt.Sprint(sessionId), config.SessionTimeOut, "/", config.Domain, true, true)

	return sessionId
}

type SessionWrapper struct {
	session map[string]string
	change  bool
}

func (sw *SessionWrapper) Load(key string) string {
	return sw.session[key]
}

func (sw *SessionWrapper) Store(key, value string) {
	oldValue := sw.session[key]
	if oldValue != value {
		sw.session[key] = value
		sw.change = true
	}
}

func (sw *SessionWrapper) Delete(key string) {
	_, present := sw.session[key]
	if present {
		delete(sw.session, key)
		sw.change = true
	}
}

type SessionWrapperPt struct {
	pt *SessionWrapper
}

const sessionIdName = "sessionId"
const sessionName = "session"

func manageSession(c *gin.Context) {
	sessionId := getSessionId(c)

	session, err := sessionclient.GetInfo(sessionId)
	var change bool
	if change = err != nil; change {
		Logger.Warn("failed to retrieve Session",
			zap.Uint64(sessionIdName, sessionId),
			zap.Error(err),
		)
		session = map[string]string{}
	}

	sw := SessionWrapper{session: session, change: change}
	c.Set(sessionName, SessionWrapperPt{pt: &sw})

	c.Next()

	if sw.change {
		err = sessionclient.UpdateInfo(sessionId, session)
		if err != nil {
			Logger.Warn("failed to save Session",
				zap.Uint64(sessionIdName, sessionId),
				zap.Error(err),
			)

		}
	}
}

func GetSession(c *gin.Context) *SessionWrapper {
	swpt, _ := c.Get(sessionName)
	swptTyped, ok := swpt.(SessionWrapperPt)
	if !ok {
		Logger.Warn("there is no Session in Context")
		sw := SessionWrapper{session: map[string]string{}, change: true}
		swptTyped = SessionWrapperPt{pt: &sw}
		c.Set(sessionName, swptTyped)
	}
	return swptTyped.pt
}
