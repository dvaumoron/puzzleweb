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
package errors

import (
	"errors"
	"net/url"

	"github.com/dvaumoron/puzzleweb/locale"
	"github.com/gin-gonic/gin"
)

const Msg = "ErrorMsg"
const QueryError = "?error="
const WrongLang = "wrong.lang"
const NoElement = "no.element"

var ErrorNotAuthorized = errors.New("error.not.authorized")
var ErrorTechnical = errors.New("error.technical.problem")
var ErrorUpdate = errors.New("error.update")

func DefaultErrorRedirect(errMsg string, c *gin.Context) string {
	return "/?error=" + url.QueryEscape(locale.GetText(errMsg, c))
}
