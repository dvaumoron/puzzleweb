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

	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/log"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/text/language"
)

const LangName = "lang"

var matcher language.Matcher
var allLang []string
var DefaultLang string
var messages map[string]map[string]string

type Tags struct {
	list []language.Tag
}

func (a *Tags) Add(lang language.Tag) {
	a.list = append(a.list, lang)
}

var Availables Tags = Tags{list: make([]language.Tag, 0, 1)}

func InitMessages() {
	const pathName = "path"
	if matcher == nil {
		list := Availables.list
		size := len(list)
		if size == 0 {
			log.Logger.Fatal("No locales declared.")
		}
		allLang = make([]string, 0, size)
		for _, langTag := range list {
			allLang = append(allLang, langTag.String())
		}
		DefaultLang = allLang[0]
		matcher = language.NewMatcher(list)
		messages = make(map[string]map[string]string)
		for _, lang := range allLang {
			messagesLang := make(map[string]string)
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
				if equal := strings.Index(line, "="); equal >= 0 {
					if key := strings.TrimSpace(line[:equal]); len(key) > 0 {
						value := ""
						if len(line) > equal {
							value = strings.TrimSpace(line[equal+1:])
						}
						messagesLang[key] = value
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
	}
}

func GetText(key string, c *gin.Context) string {
	return getText(key, GetLang(c))
}

func GetLang(c *gin.Context) string {
	lang, err := c.Cookie(LangName)
	if err == nil {
		lang = checkLang(lang)
	} else {
		tag, _ := language.MatchStrings(matcher, c.GetHeader("Accept-Language"))
		lang = setLangCookie(c, tag.String())
	}
	return lang
}

func checkLang(lang string) string {
	for _, l := range allLang {
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
	setLangCookie(c, checkLang(lang))
}

func getText(key, lang string) string {
	text := messages[lang][key]
	if text == "" {
		if lang == DefaultLang {
			warnMissingDefault(key, DefaultLang)
			text = key
		} else {
			log.Logger.Warn("Missing key, falling to default locale.",
				zap.String("key", key),
				zap.String("currentLocale", lang),
			)
			text = messages[DefaultLang][key]
			if text == "" {
				warnMissingDefault(key, DefaultLang)
				text = key
			}
		}
	}
	return text
}

func warnMissingDefault(key, defaultLang string) {
	log.Logger.Warn("Missing key in default locale.",
		zap.String("key", key),
		zap.String("defaultLocale", defaultLang),
	)
}
