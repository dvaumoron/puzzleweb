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
package common

import (
	"errors"
	"net/url"
	"strings"

	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/dvaumoron/puzzleweb/log"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const ErrorMsgName = "ErrorMsg"
const QueryError = "?error="
const WrongLangKey = "wrong.lang"
const NoElementKey = "no.element"
const UnknownUser = "error.unknown.user"

// error displayed to user
var ErrNotAuthorized = errors.New("error.not.authorized")
var ErrTechnical = errors.New("error.technical.problem")
var ErrUpdate = errors.New("error.update")

func LogOriginalError(err error) {
	log.Logger.Warn("Original error.", zap.Error(err))
}

func WriteError(urlBuilder *strings.Builder, errMsg string, c *gin.Context) {
	urlBuilder.WriteString(QueryError)
	urlBuilder.WriteString(url.QueryEscape(locale.GetText(errMsg, c)))
}

func DefaultErrorRedirect(errMsg string, c *gin.Context) string {
	return "/?error=" + url.QueryEscape(locale.GetText(errMsg, c))
}
