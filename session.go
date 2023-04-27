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
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"

	"github.com/dvaumoron/puzzleweb/config"
	"github.com/gin-gonic/gin"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
)

const cookieName = "pw_session_id"
const sessionName = "Session"

var errDecodeTooShort = errors.New("the result from base64 decoding is too short")

type sessionManager config.SessionConfig

func makeSessionManager(sessionConfig config.SessionConfig) sessionManager {
	return sessionManager(sessionConfig)
}

func (m sessionManager) getSessionId(c *gin.Context) (uint64, error) {
	cookie, err := c.Cookie(cookieName)
	if err != nil {
		m.Logger.Info("Failed to retrieve session cookie", zap.Error(err))
		return m.generateSessionCookie(c)
	}
	sessionId, err := decodeFromBase64(cookie)
	if err != nil {
		m.Logger.Info("Failed to parse session cookie", zap.Error(err))
		return m.generateSessionCookie(c)
	}
	// refreshing cookie
	m.setSessionCookie(sessionId, c)
	return sessionId, nil
}

func (m sessionManager) generateSessionCookie(c *gin.Context) (uint64, error) {
	sessionId, err := m.Service.Generate()
	if err == nil {
		m.setSessionCookie(sessionId, c)
	}
	return sessionId, err
}

func (m sessionManager) setSessionCookie(sessionId uint64, c *gin.Context) {
	c.SetCookie(cookieName, encodeToBase64(sessionId), m.TimeOut, "/", m.Domain, true, true)
}

func encodeToBase64(i uint64) string {
	bs := make([]byte, 8)
	bs[0] = byte(i)
	i >>= 8
	bs[1] = byte(i)
	i >>= 8
	bs[2] = byte(i)
	i >>= 8
	bs[3] = byte(i)
	i >>= 8
	bs[4] = byte(i)
	i >>= 8
	bs[5] = byte(i)
	i >>= 8
	bs[6] = byte(i)
	i >>= 8
	bs[7] = byte(i)
	return base64.StdEncoding.EncodeToString(bs)
}

func decodeFromBase64(s string) (uint64, error) {
	bs, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return 0, err
	}
	if len(bs) < 8 {
		return 0, errDecodeTooShort
	}
	res := uint64(bs[0])
	res |= uint64(bs[1]) << 8
	res |= uint64(bs[2]) << 16
	res |= uint64(bs[3]) << 24
	res |= uint64(bs[4]) << 32
	res |= uint64(bs[5]) << 40
	res |= uint64(bs[6]) << 48
	res |= uint64(bs[7]) << 56
	return res, nil
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
		m.Logger.Error("Failed to generate sessionId")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	session, err := m.Service.Get(sessionId)
	if err != nil {
		logSessionError("Failed to retrieve session", sessionId, c)
		return
	}

	if session == nil {
		session = map[string]string{}
	}

	c.Set(sessionName, &Session{session: session}) // change is false (default bool)
	c.Next()

	if s := GetSession(c); s.change {
		if m.Service.Update(sessionId, s.session) != nil {
			logSessionError("Failed to save session", sessionId, c)
		}
	}
}

func logSessionError(msg string, sessionId uint64, c *gin.Context) {
	getSite(c).logger.Error(msg, zap.Uint64("sessionId", sessionId))
	c.AbortWithStatus(http.StatusInternalServerError)
}

func GetSession(c *gin.Context) *Session {
	untyped, _ := c.Get(sessionName)
	typed, ok := untyped.(*Session)
	if !ok {
		getSite(c).logger.Error("There is no session in context")
		typed = &Session{session: map[string]string{}, change: true}
		c.Set(sessionName, typed)
	}
	return typed
}

func GetSessionUserId(c *gin.Context) uint64 {
	return extractUserIdFromSession(getSite(c).logger, GetSession(c))
}

func extractUserIdFromSession(logger *otelzap.Logger, session *Session) uint64 {
	userId, err := strconv.ParseUint(session.Load(userIdName), 10, 64)
	if err != nil {
		logger.Info("Failed to parse userId from session", zap.Error(err))
	}
	return userId
}
