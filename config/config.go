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
)

const defaultSessionTimeOut = 1200

var Domain string
var Port string

var LogConfig []byte

var SessionTimeOut int

var StaticPath string
var TemplatePath string

var SessionServiceAddr string
var MarkdownServiceAddr string

func init() {
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
			fmt.Println("failed to read logging config file :", err)
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
			SessionTimeOut = defaultSessionTimeOut
		}
	}

	StaticPath = os.Getenv("STATIC_PATH")
	if StaticPath == "" {
		StaticPath = "static"
	}

	TemplatePath = os.Getenv("TEMPLATE_PATH")
	if TemplatePath == "" {
		TemplatePath = "templates"
	}

	SessionServiceAddr = os.Getenv("SESSION_SERVICE_ADDR")
	if SessionServiceAddr == "" {
		SessionServiceAddr = "localhost:50051"
	}

	MarkdownServiceAddr = os.Getenv("MARKDOWN_SERVICE_ADDR")
	if MarkdownServiceAddr == "" {
		MarkdownServiceAddr = "localhost:50051"
	}
}
