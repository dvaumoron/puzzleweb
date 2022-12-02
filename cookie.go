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
	"github.com/gin-gonic/gin"
)

func sessionCookie(c *gin.Context) {
	var session_id uint64
	cookie, err := c.Cookie("pw_session_id")
	if err == nil {
		var id_bytes []byte
		id_bytes, err = base64.StdEncoding.DecodeString(cookie)
		if err == nil {
			session_id = new(big.Int).SetBytes(id_bytes).Uint64()
		} else {
			c.AbortWithStatus(http.StatusInternalServerError)
		}
	} else {
		session_id, cookie = generateCookie()
		c.SetCookie("pw_session_id", cookie, config.SessionTimeOut, "/", config.Domain, true, true)
	}

	c.Set("session_id", session_id)
}

func generateCookie() (uint64, string) {
	var session_id uint64
	var cookie = base64.StdEncoding.EncodeToString(big.NewInt(int64(session_id)).Bytes())
	return session_id, cookie
}
