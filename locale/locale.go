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
	"bufio"
	"os"
	"strings"
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
	GetMessages(*gin.Context) map[string]string
}

type localesManager struct {
	logger         *zap.Logger
	AllLang        []string
	DefaultLang    string
	MultipleLang   bool
	matcher        language.Matcher
	messages       map[string]map[string]string
	sessionTimeOut int
	domain         string
}

func NewManager(localesConfig config.LocalesConfig) Manager {
	logger := localesConfig.Logger
	localesPath := localesConfig.Path
	allLang := localesConfig.AllLang
	size := len(allLang)
	if size == 0 {
		logger.Fatal("No locales declared")
	}
	multipleLang := size > 1

	tags := make([]language.Tag, 0, size)
	for _, lang := range allLang {
		tags = append(tags, language.MustParse(lang))
	}
	defaultLang := allLang[0]
	matcher := language.NewMatcher(tags)

	messages := map[string]map[string]string{}
	for _, lang := range allLang {
		messagesLang := map[string]string{}
		messages[lang] = messagesLang

		var pathBuilder strings.Builder
		pathBuilder.WriteString(localesPath)
		pathBuilder.WriteString("/messages_")
		pathBuilder.WriteString(lang)
		pathBuilder.WriteString(".properties")
		path := pathBuilder.String()
		file, err := os.Open(path)
		if err != nil {
			logger.Fatal("Failed to load locale file", zap.String(pathName, path), zap.Error(err))
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) != 0 && line[0] != '#' {
				if equal := strings.Index(line, "="); equal > 0 {
					if key := strings.TrimSpace(line[:equal]); key != "" {
						if value := strings.TrimSpace(line[equal+1:]); value != "" {
							messagesLang[key] = value
						}
					}
				}
			}
		}
		if err = scanner.Err(); err != nil {
			logger.Error("Error reading locale file", zap.String(pathName, path), zap.Error(err))
		}
	}

	messagesDefaultLang := messages[defaultLang]
	for _, lang := range allLang {
		if lang == defaultLang {
			continue
		}
		messagesLang := messages[lang]
		for key, value := range messagesLang {
			if value == "" {
				messagesLang[key] = messagesDefaultLang[key]
			}
		}
	}

	return &localesManager{
		logger: logger, AllLang: allLang, DefaultLang: defaultLang, MultipleLang: multipleLang, matcher: matcher,
		messages: messages, sessionTimeOut: localesConfig.SessionTimeOut, domain: localesConfig.Domain,
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
	m.logger.Info("Asked not declared locale", zap.String("askedLocale", lang))
	return m.DefaultLang
}

func (m *localesManager) setLangCookie(lang string, c *gin.Context) string {
	c.SetCookie(LangName, lang, m.sessionTimeOut, "/", m.domain, false, false)
	return lang
}

func (m *localesManager) SetLangCookie(lang string, c *gin.Context) string {
	return m.setLangCookie(m.CheckLang(lang), c)
}

func (m *localesManager) GetMessages(c *gin.Context) map[string]string {
	return m.messages[m.GetLang(c)]
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
