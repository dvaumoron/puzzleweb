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
	"math/big"
	"net/http"

	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/sessionclient"
	"github.com/gin-gonic/gin"
)

const sessionIdName = "sessionId"
const cookieName = "pw_session_id"

func sessionCookie(c *gin.Context) {
	var sessionId uint64
	cookie, err := c.Cookie(cookieName)
	if err == nil {
		var id_bytes []byte
		id_bytes, err = base64.StdEncoding.DecodeString(cookie)
		if err == nil {
			sessionId = new(big.Int).SetBytes(id_bytes).Uint64()
		} else {
			sessionId = generateSessionCookie(c)
		}
	} else {
		sessionId = generateSessionCookie(c)
	}

	c.Set(sessionIdName, sessionId)
}

func generateSessionCookie(c *gin.Context) uint64 {
	sessionId, err := sessionclient.Generate()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}

	cookie := base64.StdEncoding.EncodeToString(big.NewInt(int64(sessionId)).Bytes())
	c.SetCookie(cookieName, cookie, config.SessionTimeOut, "/", config.Domain, true, true)

	return sessionId
}
