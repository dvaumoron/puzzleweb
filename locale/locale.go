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

	"github.com/dvaumoron/puzzleweb/log"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/text/language"
)

const LangName = "lang"
const pathName = "Path"

var matcher language.Matcher
var AllLang []string
var DefaultLang string
var MultipleLang bool
var messages map[string]map[string]string = map[string]map[string]string{}

type Tags struct {
	list []language.Tag
}

func (a *Tags) Add(lang language.Tag) {
	a.list = append(a.list, lang)
}

var Availables Tags = Tags{list: make([]language.Tag, 0, 1)}

func InitMessages(logger *zap.Logger, localesPath string) {
	if matcher != nil {
		return
	}
	list := Availables.list
	size := len(list)
	if size == 0 {
		logger.Fatal("No locales declared.")
	}
	MultipleLang = size > 1

	AllLang = make([]string, 0, size)
	for _, langTag := range list {
		AllLang = append(AllLang, langTag.String())
	}
	DefaultLang = AllLang[0]
	matcher = language.NewMatcher(list)
	for _, lang := range AllLang {
		messagesLang := map[string]string{}
		messages[lang] = messagesLang

		var pathBuilder strings.Builder
		pathBuilder.WriteString(localesPath)
		pathBuilder.WriteString("/message_")
		pathBuilder.WriteString(lang)
		pathBuilder.WriteString(".property")
		path := pathBuilder.String()
		file, err := os.Open(path)
		if err != nil {
			logger.Fatal("Failed to load locale file.", zap.String(pathName, path), zap.Error(err))
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
			logger.Error("Error reading locale file.", zap.String(pathName, path), zap.Error(err))
		}
	}

	messagesDefaultLang := messages[DefaultLang]
	for _, lang := range AllLang {
		if lang == DefaultLang {
			continue
		}
		messagesLang := messages[lang]
		for key, value := range messagesLang {
			if value == "" {
				messagesLang[key] = messagesDefaultLang[key]
			}
		}
	}
}

func GetText(key string, c *gin.Context) string {
	return GetMessages(c)[key]
}

func GetLang(c *gin.Context) string {
	lang, err := c.Cookie(LangName)
	if err != nil {
		tag, _ := language.MatchStrings(matcher, c.GetHeader("Accept-Language"))
		return setLangCookie(c, tag.String())
	}
	return CheckLang(lang)
}

func CheckLang(lang string) string {
	for _, l := range AllLang {
		if lang == l {
			return lang
		}
	}
	log.Logger.Info("Asked not declared locale.", zap.String("askedLocale", lang))
	return DefaultLang
}

func setLangCookie(c *gin.Context, lang string) string {
	c.SetCookie(
		LangName, lang, config.Shared.SessionTimeOut,
		"/", config.Shared.Domain, false, false,
	)
	return lang
}

func SetLangCookie(c *gin.Context, lang string) {
	setLangCookie(c, CheckLang(lang))
}

func GetMessages(c *gin.Context) map[string]string {
	return messages[GetLang(c)]
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
