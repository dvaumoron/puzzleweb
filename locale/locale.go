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

package locale

import (
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
)

const langName = "lang"

var matcher language.Matcher

func InitAvailableLanguages(availableLanguages []language.Tag) {
	matcher = language.NewMatcher(availableLanguages)
}

func GetText(key string, c *gin.Context) string {
	lang, err := c.Cookie(langName)
	if err != nil {
		tag, _ := language.MatchStrings(matcher, c.GetHeader("Accept-Language"))
		SetLangCookie(c, tag.String())
	}
	return getText(key, lang)
}

func SetLangCookie(c *gin.Context, lang string) {
	c.SetCookie(
		langName, lang, config.SessionTimeOut,
		"/", config.Domain, false, false,
	)
}

func getText(key, lang string) string {
	return "" // TODO
}
