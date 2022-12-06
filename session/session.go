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

package session

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/log"
	"github.com/dvaumoron/puzzleweb/session/client"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const cookieName = "pw_session_id"

func getSessionId(c *gin.Context) (uint64, error) {
	var sessionId uint64
	cookie, err := c.Cookie(cookieName)
	if err == nil {
		sessionId, err = strconv.ParseUint(cookie, 10, 0)
		if err != nil {
			sessionId, err = generateSessionCookie(c)
		}
	} else {
		sessionId, err = generateSessionCookie(c)
	}
	return sessionId, err
}

func generateSessionCookie(c *gin.Context) (uint64, error) {
	sessionId, err := client.Generate()
	if err == nil {
		c.SetCookie(
			cookieName, fmt.Sprint(sessionId),
			config.SessionTimeOut, "/",
			config.Domain, true, true,
		)
	}
	return sessionId, err
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

const sessionName = "session"

func Manage(c *gin.Context) {
	const sessionIdName = "sessionId"

	sessionId, err := getSessionId(c)
	if err == nil {
		session, err := client.GetInfo(sessionId)
		var change bool
		if change = err != nil; change {
			log.Logger.Warn("Failed to retrieve session.",
				zap.Uint64(sessionIdName, sessionId),
				zap.Error(err),
			)
			session = map[string]string{}
		}

		c.Set(sessionName, &SessionWrapper{session: session, change: change})
		c.Next()

		if sw := Get(c); sw.change {
			err = client.UpdateInfo(sessionId, sw.session)
			if err != nil {
				log.Logger.Warn("Failed to save session.",
					zap.Uint64(sessionIdName, sessionId),
					zap.Error(err),
				)

			}
		}
	} else {
		c.AbortWithError(http.StatusInternalServerError, err)
	}
}

func Get(c *gin.Context) *SessionWrapper {
	var swptTyped *SessionWrapper
	swpt, ok := c.Get(sessionName)
	if ok {
		swptTyped = swpt.(*SessionWrapper)
	} else {
		log.Logger.Warn("There is no session in context.")
		swptTyped = &SessionWrapper{session: map[string]string{}, change: true}
		c.Set(sessionName, swptTyped)
	}
	return swptTyped
}
