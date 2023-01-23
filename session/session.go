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

	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/log"
	"github.com/dvaumoron/puzzleweb/session/client"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const cookieName = "pw_session_id"
const sessionName = "Session"

func getSessionId(c *gin.Context) (uint64, error) {
	cookie, err := c.Cookie(cookieName)
	if err != nil {
		return generateSessionCookie(c)
	}
	sessionId, err := strconv.ParseUint(cookie, 10, 64)
	if err != nil {
		log.Logger.Info("Failed to parse session cookie.", zap.Error(err))
		return generateSessionCookie(c)
	}
	return sessionId, nil
}

func generateSessionCookie(c *gin.Context) (uint64, error) {
	sessionId, err := client.Generate()
	if err == nil {
		c.SetCookie(
			cookieName, fmt.Sprint(sessionId), config.SessionTimeOut,
			"/", config.Domain, true, true,
		)
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

func Manage(c *gin.Context) {
	sessionId, err := getSessionId(c)
	if err != nil {
		log.Logger.Error("Failed to generate sessionId.", zap.Error(err))
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	session, err := client.GetSession(sessionId)
	if err != nil {
		logSessionError(c, "Failed to retrieve session.", sessionId, err)
		return
	}

	c.Set(sessionName, &Session{session: session, change: false})
	c.Next()

	if s := Get(c); s.change {
		if err := client.UpdateSession(sessionId, s.session); err != nil {
			logSessionError(c, "Failed to save session.", sessionId, err)
		}
	}
}

func logSessionError(c *gin.Context, msg string, sessionId uint64, err error) {
	log.Logger.Error(msg, zap.Uint64("sessionId", sessionId), zap.Error(err))
	c.AbortWithError(http.StatusInternalServerError, err)
}

func Get(c *gin.Context) *Session {
	var typed *Session
	untyped, _ := c.Get(sessionName)
	typed, ok := untyped.(*Session)
	if !ok {
		log.Logger.Error("There is no session in context.")
		typed = &Session{session: map[string]string{}, change: true}
		c.Set(sessionName, typed)
	}
	return typed
}

func GetUserId(c *gin.Context) uint64 {
	userIdStr := Get(c).Load(common.UserIdName)
	userId, err := strconv.ParseUint(userIdStr, 10, 64)
	if err != nil {
		log.Logger.Info("Failed to parse userId.", zap.Error(err))
		return 0
	}
	return userId
}
