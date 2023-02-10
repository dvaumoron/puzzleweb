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

	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const cookieName = "pw_session_id"
const sessionName = "Session"

type sessionManager config.SessionConfig

func makeSessionManager(sessionConfig config.SessionConfig) sessionManager {
	return sessionManager(sessionConfig)
}

func (m sessionManager) getSessionId(c *gin.Context) (uint64, error) {
	cookie, err := c.Cookie(cookieName)
	if err != nil {
		m.Logger.Info("Failed to retrieve session cookie.", zap.Error(err))
		return m.generateSessionCookie(c)
	}
	sessionId, err := strconv.ParseUint(cookie, 10, 64)
	if err != nil {
		m.Logger.Info("Failed to parse session cookie.", zap.Error(err))
		return m.generateSessionCookie(c)
	}
	return sessionId, nil
}

func (m sessionManager) generateSessionCookie(c *gin.Context) (uint64, error) {
	sessionId, err := m.Service.Generate()
	if err == nil {
		c.SetCookie(cookieName, fmt.Sprint(sessionId), m.TimeOut, "/", m.Domain, true, true)
	}
	return sessionId, err
}

type Session struct {
	session map[string]string
	change  bool
}

func (s *Session) Load(key string) string {
	return s.session[key]
}

func (s *Session) Store(key, value string) {
	oldValue := s.session[key]
	if oldValue != value {
		s.session[key] = value
		s.change = true
	}
}

func (s *Session) Delete(key string) {
	_, present := s.session[key]
	if present {
		s.session[key] = "" // to allow a deletion in the service
		s.change = true
	}
}

func (m sessionManager) Manage(c *gin.Context) {
	sessionId, err := m.getSessionId(c)
	if err != nil {
		m.Logger.Error("Failed to generate sessionId.")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	session, err := m.Service.Get(sessionId)
	if err != nil {
		logSessionError(c, "Failed to retrieve session.", sessionId)
		return
	}

	c.Set(sessionName, &Session{session: session}) // change is false (default bool)
	c.Next()

	if s := GetSession(c); s.change {
		if m.Service.Update(sessionId, s.session) != nil {
			logSessionError(c, "Failed to save session.", sessionId)
		}
	}
}

func logSessionError(c *gin.Context, msg string, sessionId uint64) {
	getSite(c).logger.Error(msg, zap.Uint64("sessionId", sessionId))
	c.AbortWithStatus(http.StatusInternalServerError)
}

func GetSession(c *gin.Context) *Session {
	untyped, _ := c.Get(sessionName)
	typed, ok := untyped.(*Session)
	if !ok {
		getSite(c).logger.Error("There is no session in context.")
		typed = &Session{session: map[string]string{}, change: true}
		c.Set(sessionName, typed)
	}
	return typed
}

func GetSessionUserId(c *gin.Context) uint64 {
	userId, err := strconv.ParseUint(GetSession(c).Load(common.UserIdName), 10, 64)
	if err != nil {
		getSite(c).logger.Info("Failed to parse userId from session.", zap.Error(err))
	}
	return userId
}