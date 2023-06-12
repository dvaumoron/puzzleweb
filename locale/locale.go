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
	"unicode"

	"github.com/dvaumoron/puzzleweb/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/text/language"
)

const LangName = "lang"
const pathName = "Path"

type Manager interface {
	GetDefaultLang() string
	GetAllLang() []string
	GetMultipleLang() bool
	GetLang(*gin.Context) string
	CheckLang(string) string
	SetLangCookie(string, *gin.Context) string
}

type localesManager struct {
	config.LocalesConfig
	DefaultLang  string
	MultipleLang bool
	matcher      language.Matcher
}

func NewManager(localesConfig config.LocalesConfig) Manager {
	allLang := localesConfig.AllLang
	size := len(allLang)
	if size == 0 {
		localesConfig.Logger.Fatal("No locales declared")
	}

	tags := make([]language.Tag, 0, size)
	for _, lang := range allLang {
		tags = append(tags, language.MustParse(lang))
	}

	return &localesManager{
		LocalesConfig: localesConfig, DefaultLang: allLang[0], MultipleLang: size > 1, matcher: language.NewMatcher(tags),
	}
}

func (m *localesManager) GetDefaultLang() string {
	return m.DefaultLang
}

func (m *localesManager) GetAllLang() []string {
	return m.AllLang
}

func (m *localesManager) GetMultipleLang() bool {
	return m.MultipleLang
}

func (m *localesManager) GetLang(c *gin.Context) string {
	lang, err := c.Cookie(LangName)
	if err != nil {
		tag, _ := language.MatchStrings(m.matcher, c.GetHeader("Accept-Language"))
		return m.setLangCookie(tag.String(), c)
	}
	// check & refresh cookie
	return m.SetLangCookie(lang, c)
}

func (m *localesManager) CheckLang(lang string) string {
	for _, l := range m.AllLang {
		if lang == l {
			return lang
		}
	}
	m.Logger.Info("Asked not declared locale", zap.String("askedLocale", lang))
	return m.DefaultLang
}

func (m *localesManager) setLangCookie(lang string, c *gin.Context) string {
	c.SetCookie(LangName, lang, m.SessionTimeOut, "/", m.Domain, false, false)
	return lang
}

func (m *localesManager) SetLangCookie(lang string, c *gin.Context) string {
	return m.setLangCookie(m.CheckLang(lang), c)
}

func CamelCase(word string) string {
	if word == "" {
		return ""
	}

	first := true
	chars := make([]rune, 0, len(word))
	for _, char := range word {
		if first {
			first = false
			char = unicode.ToTitle(char)
		}
		chars = append(chars, char)
	}
	return string(chars)
}
