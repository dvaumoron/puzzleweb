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

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/text/language"
)

const LangName = "lang"
const pathName = "Path"

type LocalesManager struct {
	logger         *zap.Logger
	tags           []language.Tag
	matcher        language.Matcher
	AllLang        []string
	DefaultLang    string
	MultipleLang   bool
	messages       map[string]map[string]string
	sessionTimeOut int
	domain         string
}

func NewManager(logger *zap.Logger, sessionTimeOut int, domain string) *LocalesManager {
	return &LocalesManager{
		logger: logger, tags: make([]language.Tag, 0, 1), sessionTimeOut: sessionTimeOut, domain: domain,
	}
}

func (m *LocalesManager) AddLang(lang language.Tag) {
	m.tags = append(m.tags, lang)
}

func (m *LocalesManager) InitMessages(localesPath string) {
	if m.matcher != nil {
		// avoid multiple initialization
		return
	}
	list := m.tags
	size := len(list)
	if size == 0 {
		m.logger.Fatal("No locales declared.")
	}
	m.MultipleLang = size > 1

	m.AllLang = make([]string, 0, size)
	for _, langTag := range list {
		m.AllLang = append(m.AllLang, langTag.String())
	}
	m.DefaultLang = m.AllLang[0]
	m.matcher = language.NewMatcher(list)
	m.messages = map[string]map[string]string{}
	for _, lang := range m.AllLang {
		messagesLang := map[string]string{}
		m.messages[lang] = messagesLang

		var pathBuilder strings.Builder
		pathBuilder.WriteString(localesPath)
		pathBuilder.WriteString("/message_")
		pathBuilder.WriteString(lang)
		pathBuilder.WriteString(".property")
		path := pathBuilder.String()
		file, err := os.Open(path)
		if err != nil {
			m.logger.Fatal("Failed to load locale file.", zap.String(pathName, path), zap.Error(err))
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
			m.logger.Error("Error reading locale file.", zap.String(pathName, path), zap.Error(err))
		}
	}

	messagesDefaultLang := m.messages[m.DefaultLang]
	for _, lang := range m.AllLang {
		if lang == m.DefaultLang {
			continue
		}
		messagesLang := m.messages[lang]
		for key, value := range messagesLang {
			if value == "" {
				messagesLang[key] = messagesDefaultLang[key]
			}
		}
	}
}

func (m *LocalesManager) GetText(key string, c *gin.Context) string {
	return m.GetMessages(c)[key]
}

func (m *LocalesManager) GetLang(c *gin.Context) string {
	lang, err := c.Cookie(LangName)
	if err != nil {
		tag, _ := language.MatchStrings(m.matcher, c.GetHeader("Accept-Language"))
		return m.setLangCookie(c, tag.String())
	}
	return m.CheckLang(lang)
}

func (m *LocalesManager) CheckLang(lang string) string {
	for _, l := range m.AllLang {
		if lang == l {
			return lang
		}
	}
	m.logger.Info("Asked not declared locale.", zap.String("askedLocale", lang))
	return m.DefaultLang
}

func (m *LocalesManager) setLangCookie(c *gin.Context, lang string) string {
	c.SetCookie(LangName, lang, m.sessionTimeOut, "/", m.domain, false, false)
	return lang
}

func (m *LocalesManager) SetLangCookie(c *gin.Context, lang string) {
	m.setLangCookie(c, m.CheckLang(lang))
}

func (m *LocalesManager) GetMessages(c *gin.Context) map[string]string {
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
