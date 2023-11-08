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

	"github.com/dvaumoron/puzzleweb/common/log"
	"go.uber.org/zap"
)

const WrongLangKey = "WrongLang"

const ReportingPlaceName = "reporting_place"

const (
	ErrorKey       = "error"
	errorKeyEq     = ErrorKey + "="
	QueryError     = "?" + errorKeyEq
	AddQueryError  = "&" + errorKeyEq
	PathQueryError = "/" + QueryError
)

// error displayed to user
const (
	ErrorBadRoleNameKey          = "ErrorBadRoleName"
	ErrorBaseVersionKey          = "BaseVersionOutdated"
	ErrorEmptyCommentKey         = "EmptyComment"
	ErrorEmptyLoginKey           = "EmptyLogin"
	ErrorEmptyPasswordKey        = "EmptyPassword"
	ErrorExistingLoginKey        = "ExistingLogin"
	ErrorNotAuthorizedKey        = "ErrorNotAuthorized"
	ErrorTechnicalKey            = "ErrorTechnicalProblem"
	ErrorUpdateKey               = "ErrorUpdate"
	ErrorWeakPasswordKey         = "WeakPassword"
	ErrorWrongConfirmPasswordKey = "WrongConfirmPassword"
	ErrorWrongLangKey            = "WrongLang"
	ErrorWrongLoginKey           = "WrongLogin"
)

const originalErrorMsg = "Original error"

var (
	ErrBadRoleName   = errors.New(ErrorBadRoleNameKey)
	ErrBaseVersion   = errors.New(ErrorBaseVersionKey)
	ErrEmptyComment  = errors.New(ErrorEmptyCommentKey)
	ErrEmptyLogin    = errors.New(ErrorEmptyLoginKey)
	ErrEmptyPassword = errors.New(ErrorEmptyPasswordKey)
	ErrExistingLogin = errors.New(ErrorExistingLoginKey)
	ErrNotAuthorized = errors.New(ErrorNotAuthorizedKey)
	ErrTechnical     = errors.New(ErrorTechnicalKey)
	ErrUpdate        = errors.New(ErrorUpdateKey)
	ErrWeakPassword  = errors.New(ErrorWeakPasswordKey)
	ErrWrongConfirm  = errors.New(ErrorWrongConfirmPasswordKey)
	ErrWrongLogin    = errors.New(ErrorWrongLoginKey)
)

func LogOriginalError(logger log.Logger, err error) {
	logger.Warn(originalErrorMsg, zap.Error(err))
}

func WriteError(urlBuilder *strings.Builder, logger log.Logger, errorMsg string) {
	urlBuilder.WriteString(QueryError)
	urlBuilder.WriteString(FilterErrorMsg(logger, errorMsg))
}

func DefaultErrorRedirect(logger log.Logger, errorMsg string) string {
	return PathQueryError + FilterErrorMsg(logger, errorMsg)
}

func FilterErrorMsg(logger log.Logger, errorMsg string) string {
	if errorMsg == ErrorBadRoleNameKey || errorMsg == ErrorBaseVersionKey || errorMsg == ErrorEmptyCommentKey ||
		errorMsg == ErrorEmptyLoginKey || errorMsg == ErrorEmptyPasswordKey || errorMsg == ErrorExistingLoginKey ||
		errorMsg == ErrorNotAuthorizedKey || errorMsg == ErrorTechnicalKey || errorMsg == ErrorUpdateKey ||
		errorMsg == ErrorWeakPasswordKey || errorMsg == ErrorWrongConfirmPasswordKey || errorMsg == ErrorWrongLangKey ||
		errorMsg == ErrorWrongLoginKey {
		return errorMsg
	}
	logger.Error(originalErrorMsg, zap.String(ErrorKey, errorMsg))
	return ErrorTechnicalKey
}
