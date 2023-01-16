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
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

const defaultSessionTimeOut = 1200
const defaultPageSize = 50

var Domain string
var Port string

var LogConfig []byte

var SessionTimeOut int
var PageSize uint64
var DateFormat string

var StaticPath string
var LocalesPath string
var TemplatesPath string

var SessionServiceAddr string
var LoginServiceAddr string
var RightServiceAddr string
var ProfileServiceAddr string
var SettingsServiceAddr string
var WikiServiceAddr string
var MarkdownServiceAddr string
var ForumServiceAddr string

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Failed to load .env file")
		os.Exit(1)
	}

	retrieveWithDefault("SITE_DOMAIN", &Domain, "localhost")
	retrieveWithDefault("SITE_PORT", &Port, "8080")

	fileLogConfigPath := os.Getenv("LOG_CONFIG_PATH")
	if fileLogConfigPath != "" {
		var err error
		LogConfig, err = os.ReadFile(fileLogConfigPath)
		if err != nil {
			fmt.Println("Failed to read logging config file :", err)
			LogConfig = nil
		}
	}

	sessionTimeOutStr := os.Getenv("SESSION_TIME_OUT")
	if sessionTimeOutStr == "" {
		SessionTimeOut = defaultSessionTimeOut
	} else {
		SessionTimeOut, _ = strconv.Atoi(sessionTimeOutStr)
		if SessionTimeOut == 0 {
			fmt.Println("Failed to parse SESSION_TIME_OUT, using default")
			SessionTimeOut = defaultSessionTimeOut
		}
	}

	pageSizeStr := os.Getenv("PAGE_SIZE")
	if pageSizeStr == "" {
		PageSize = defaultPageSize
	} else {
		PageSize, _ = strconv.ParseUint(pageSizeStr, 10, 64)
		if PageSize == 0 {
			fmt.Println("Failed to parse PAGE_SIZE, using default")
			PageSize = defaultPageSize
		}
	}

	retrieveWithDefault("DATE_FORMAT", &DateFormat, "TODO")

	retrievePath("STATIC_PATH", &StaticPath, "static")
	retrievePath("LOCALES_PATH", &LocalesPath, "locales")
	retrievePath("TEMPLATES_PATH", &TemplatesPath, "templates")

	requiredFromEnv("SESSION_SERVICE_ADDR", &SessionServiceAddr)
	requiredFromEnv("LOGIN_SERVICE_ADDR", &LoginServiceAddr)
	requiredFromEnv("RIGHT_SERVICE_ADDR", &RightServiceAddr)
	requiredFromEnv("PROFILE_SERVICE_ADDR", &ProfileServiceAddr)
	requiredFromEnv("SETTINGS_SERVICE_ADDR", &SettingsServiceAddr)
	requiredFromEnv("WIKI_SERVICE_ADDR", &WikiServiceAddr)
	requiredFromEnv("MARKDOWN_SERVICE_ADDR", &MarkdownServiceAddr)
	requiredFromEnv("FORUM_SERVICE_ADDR", &ForumServiceAddr)
}

func retrieveWithDefault(name string, pValue *string, defaultValue string) {
	if *pValue = os.Getenv(name); *pValue == "" {
		*pValue = defaultValue
	}
}

func retrievePath(name string, pValue *string, defaultValue string) {
	if *pValue = os.Getenv(name); *pValue == "" {
		*pValue = defaultValue
	} else {
		*pValue = checkPath(*pValue)
	}
}

func checkPath(path string) string {
	if last := len(path) - 1; last != -1 && path[last] == '/' {
		path = path[:last]
	}
	return path
}

func requiredFromEnv(name string, pValue *string) {
	*pValue = os.Getenv(name)
	if *pValue == "" {
		fmt.Println(name, "not found in env")
		os.Exit(1)
	}
}
