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
	"github.com/dvaumoron/puzzleweb/log"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/text/language"
)

const LangName = "lang"
const pathName = "Path"
const keyName = "key"

var matcher language.Matcher
var AllLang []string
var DefaultLang string
var messages map[string]map[string]string
var displayMessages map[string]map[string]string

type Tags struct {
	list []language.Tag
}

func (a *Tags) Add(lang language.Tag) {
	a.list = append(a.list, lang)
}

var Availables Tags = Tags{list: make([]language.Tag, 0, 1)}

func InitMessages() {
	if matcher != nil {
		return
	}
	list := Availables.list
	size := len(list)
	if size == 0 {
		log.Logger.Fatal("No locales declared.")
	}
	AllLang = make([]string, 0, size)
	for _, langTag := range list {
		AllLang = append(AllLang, langTag.String())
	}
	DefaultLang = AllLang[0]
	matcher = language.NewMatcher(list)
	messages = map[string]map[string]string{}
	for _, lang := range AllLang {
		messagesLang := map[string]string{}
		messages[lang] = messagesLang

		var pathBuilder strings.Builder
		pathBuilder.WriteString(config.LocalesPath)
		pathBuilder.WriteString("/message_")
		pathBuilder.WriteString(lang)
		pathBuilder.WriteString(".property")
		path := pathBuilder.String()
		file, err := os.Open(path)
		if err != nil {
			log.Logger.Fatal("Failed to load locale file.",
				zap.String(pathName, path),
				zap.Error(err),
			)
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
			log.Logger.Error("Error reading locale file.",
				zap.String(pathName, path),
				zap.Error(err),
			)
		}
	}

	messagesDefaultLang := messages[DefaultLang]
	for _, lang := range AllLang {
		displayMessagesLang := map[string]string{}
		displayMessages[lang] = displayMessagesLang
		if lang == DefaultLang {
			for key, value := range messagesDefaultLang {
				displayMessagesLang[transformKey(key)] = value
			}
		} else {
			messagesLang := messages[lang]
			for key, value := range messagesLang {
				if value == "" {
					value = messagesDefaultLang[key]
				}
				displayMessagesLang[transformKey(key)] = value
			}
		}
	}
}

func GetText(key string, c *gin.Context) string {
	return getText(key, GetLang(c))
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
	log.Logger.Info("Asked not declared locale.",
		zap.String("askedLocale", lang),
	)
	return DefaultLang
}

func setLangCookie(c *gin.Context, lang string) string {
	c.SetCookie(
		LangName, lang, config.SessionTimeOut,
		"/", config.Domain, false, false,
	)
	return lang
}

func SetLangCookie(c *gin.Context, lang string) {
	setLangCookie(c, CheckLang(lang))
}

func GetMessages(c *gin.Context) map[string]string {
	return displayMessages[GetLang(c)]
}

func getText(key string, lang string) string {
	if text := messages[lang][key]; text != "" {
		return text
	}
	if lang == DefaultLang {
		return warnMissingDefault(key, DefaultLang)
	}

	log.Logger.Warn("Missing key, falling to default locale.",
		zap.String("key", key), zap.String("currentLocale", lang),
	)

	if text := messages[DefaultLang][key]; text != "" {
		return text
	}
	return warnMissingDefault(key, DefaultLang)
}

func warnMissingDefault(key string, defaultLang string) string {
	log.Logger.Warn("Missing key in default locale.",
		zap.String(keyName, key), zap.String("defaultLocale", defaultLang),
	)
	return key
}

func transformKey(key string) string {
	var keyBuilder strings.Builder
	for _, part := range strings.Split(key, ".") {
		keyBuilder.WriteString(transformWord(part))
	}
	return keyBuilder.String()
}

func transformWord(word string) string {
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
