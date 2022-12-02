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
	"github.com/dvaumoron/puzzleweb/sessionclient"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const SessionName = "session"

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

func manageSession(c *gin.Context) {
	session_id := c.GetUint64(sessionIdName)

	session, err := sessionclient.GetInfo(session_id)
	if err != nil {
		Logger.Warn("failed to retrieve Session",
			zap.Uint64(sessionIdName, session_id),
			zap.Error(err),
		)
		session = map[string]string{}
	}

	sw := SessionWrapper{session: session, change: false}
	c.Set(SessionName, SessionWrapperPt{pt: &sw})

	c.Next()

	if sw.change {
		err = sessionclient.UpdateInfo(session_id, session)
		if err != nil {
			Logger.Warn("failed to save Session",
				zap.Uint64(sessionIdName, session_id),
				zap.Error(err),
			)

		}
	}
}

func GetSession(c *gin.Context) *SessionWrapper {
	swpt, _ := c.Get(SessionName)
	swptTyped, ok := swpt.(SessionWrapperPt)
	if !ok {
		Logger.Warn("there is no Session in Context")
		sw := SessionWrapper{session: map[string]string{}, change: false}
		swptTyped = SessionWrapperPt{pt: &sw}
		c.Set(SessionName, swptTyped)
	}
	return swptTyped.pt
}
