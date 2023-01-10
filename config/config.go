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

var Domain string
var Port string

var LogConfig []byte

var SessionTimeOut int

var StaticPath string
var LocalesPath string
var TemplatesPath string

const defaultServiceAddr = "localhost:50051"

var SessionServiceAddr string
var LoginServiceAddr string
var RightServiceAddr string
var ProfileServiceAddr string
var SettingsServiceAddr string
var WikiServiceAddr string
var MarkdownServiceAddr string

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Failed to load .env file")
		os.Exit(1)
	}

	Domain = os.Getenv("SITE_DOMAIN")
	if Domain == "" {
		Domain = "localhost"
	}

	Port = os.Getenv("SITE_PORT")
	if Port == "" {
		Port = "8080"
	}

	fileLogConfigPath := os.Getenv("LOG_CONFIG_PATH")
	if fileLogConfigPath == "" {
		LogConfig = make([]byte, 0)
	} else {
		var err error
		LogConfig, err = os.ReadFile(fileLogConfigPath)
		if err != nil {
			fmt.Println("Failed to read logging config file :", err)
			LogConfig = make([]byte, 0)
		}
	}

	sessionTimeOutStr := os.Getenv("SESSION_TIME_OUT")
	if sessionTimeOutStr == "" {
		SessionTimeOut = defaultSessionTimeOut
	} else {
		var err error
		SessionTimeOut, err = strconv.Atoi(sessionTimeOutStr)
		if err != nil {
			fmt.Println("Failed to parse SESSION_TIME_OUT")
			SessionTimeOut = defaultSessionTimeOut
		}
	}

	StaticPath = os.Getenv("STATIC_PATH")
	if StaticPath == "" {
		StaticPath = "static"
	} else {
		StaticPath = checkPath(StaticPath)
	}

	LocalesPath = os.Getenv("LOCALES_PATH")
	if LocalesPath == "" {
		LocalesPath = "locales"
	} else {
		LocalesPath = checkPath(LocalesPath)
	}

	TemplatesPath = os.Getenv("TEMPLATES_PATH")
	if TemplatesPath == "" {
		TemplatesPath = "templates"
	} else {
		TemplatesPath = checkPath(TemplatesPath)
	}

	SessionServiceAddr = os.Getenv("SESSION_SERVICE_ADDR")
	if SessionServiceAddr == "" {
		SessionServiceAddr = defaultServiceAddr
	}

	LoginServiceAddr = os.Getenv("LOGIN_SERVICE_ADDR")
	if LoginServiceAddr == "" {
		LoginServiceAddr = defaultServiceAddr
	}

	RightServiceAddr = os.Getenv("RIGHT_SERVICE_ADDR")
	if RightServiceAddr == "" {
		RightServiceAddr = defaultServiceAddr
	}

	ProfileServiceAddr = os.Getenv("PROFILE_SERVICE_ADDR")
	if ProfileServiceAddr == "" {
		ProfileServiceAddr = defaultServiceAddr
	}

	SettingsServiceAddr = os.Getenv("SETTINGS_SERVICE_ADDR")
	if SettingsServiceAddr == "" {
		SettingsServiceAddr = defaultServiceAddr
	}

	WikiServiceAddr = os.Getenv("WIKI_SERVICE_ADDR")
	if WikiServiceAddr == "" {
		WikiServiceAddr = defaultServiceAddr
	}

	MarkdownServiceAddr = os.Getenv("MARKDOWN_SERVICE_ADDR")
	if MarkdownServiceAddr == "" {
		MarkdownServiceAddr = defaultServiceAddr
	}
}

func checkPath(path string) string {
	if last := len(path) - 1; last != -1 && path[last] == '/' {
		path = path[:last]
	}
	return path
}
