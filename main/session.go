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
const SessionName = "Session"

var errDecodeTooShort = errors.New("the result from base64 decoding is too short")

type sessionManager config.SessionConfig

func makeSessionManager(sessionConfig config.SessionConfig) sessionManager {
	return sessionManager(sessionConfig)
}

func (m sessionManager) getSessionId(logger otelzap.LoggerWithCtx, c *gin.Context) (uint64, error) {
	cookie, err := c.Cookie(cookieName)
	if err != nil {
		logger.Info("Failed to retrieve session cookie", zap.Error(err))
		return m.generateSessionCookie(logger, c)
	}
	sessionId, err := decodeFromBase64(cookie)
	if err != nil {
		logger.Info("Failed to parse session cookie", zap.Error(err))
		return m.generateSessionCookie(logger, c)
	}
	// refreshing cookie
	m.setSessionCookie(sessionId, c)
	return sessionId, nil
}

func (m sessionManager) generateSessionCookie(logger otelzap.LoggerWithCtx, c *gin.Context) (uint64, error) {
	sessionId, err := m.Service.Generate(logger)
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

func (s *Session) Store(key string, value string) {
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

// Writing in the returned map will not be saved.
func (s *Session) AsMap() map[string]string {
	return s.session
}

func (m sessionManager) manage(c *gin.Context) {
	logger := GetLogger(c)
	sessionId, err := m.getSessionId(logger, c)
	if err != nil {
		logger.Error("Failed to generate sessionId")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	session, err := m.Service.Get(logger, sessionId)
	if err != nil {
		logSessionError(logger, "Failed to retrieve session", sessionId, c)
		return
	}

	if session == nil {
		session = map[string]string{}
	}

	c.Set(SessionName, &Session{session: session}) // change is false (default bool)
	c.Next()

	if s := GetSession(c); s.change {
		if m.Service.Update(logger, sessionId, s.session) != nil {
			logSessionError(logger, "Failed to save session", sessionId, c)
		}
	}
}

func logSessionError(logger otelzap.LoggerWithCtx, msg string, sessionId uint64, c *gin.Context) {
	logger.WithOptions(zap.AddCallerSkip(1)).Error(msg, zap.Uint64("sessionId", sessionId))
	c.AbortWithStatus(http.StatusInternalServerError)
}

func GetSession(c *gin.Context) *Session {
	untyped, _ := c.Get(SessionName)
	typed, ok := untyped.(*Session)
	if !ok {
		GetLogger(c).Error("There is no session in context")
		typed = &Session{session: map[string]string{}, change: true}
		c.Set(SessionName, typed)
	}
	return typed
}

func GetSessionUserId(c *gin.Context) uint64 {
	userId, err := strconv.ParseUint(GetSession(c).Load(userIdName), 10, 64)
	if err == nil {
		GetLogger(c).Debug("userId parsed from session", zap.Uint64(userIdName, userId))
	} else {
		GetLogger(c).Info("Failed to parse userId from session", zap.Error(err))
	}
	return userId
}
