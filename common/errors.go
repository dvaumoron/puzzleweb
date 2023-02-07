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
	"strings"

	"github.com/dvaumoron/puzzleweb/log"
	"go.uber.org/zap"
)

const ErrorMsgName = "ErrorMsg"
const QueryError = "?error="

const NoElementKey = "NoElement"
const WrongLangKey = "WrongLang"
const UnknownUserKey = "ErrorUnknownUser"

// error displayed to user
var ErrNotAuthorized = errors.New("ErrorNotAuthorized")
var ErrTechnical = errors.New("ErrorTechnicalProblem")
var ErrUpdate = errors.New("ErrorUpdate")

func LogOriginalError(err error) {
	log.Logger.Warn("Original error.", zap.Error(err))
}

func WriteError(urlBuilder *strings.Builder, errorMsg string) {
	urlBuilder.WriteString(QueryError)
	urlBuilder.WriteString(errorMsg)
}

func DefaultErrorRedirect(errorMsg string) string {
	return "/?error=" + errorMsg
}
