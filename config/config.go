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

var Shared = loadDefault()

type Config struct {
	Domain string
	Port   string

	LogConfig []byte

	SessionTimeOut int
	PageSize       uint64
	DateFormat     string

	StaticPath    string
	LocalesPath   string
	TemplatesPath string

	SessionServiceAddr  string
	SaltServiceAddr     string
	LoginServiceAddr    string
	RightServiceAddr    string
	ProfileServiceAddr  string
	SettingsServiceAddr string
	WikiServiceAddr     string
	MarkdownServiceAddr string
	ForumServiceAddr    string
	BlogServiceAddr     string
}

func loadDefault() Config {
	if godotenv.Overload() == nil {
		fmt.Println("Loaded .env file")
	}

	var err error
	var logConfig []byte
	var sessionTimeOut int
	var pageSize uint64

	domain := retrieveWithDefault("SITE_DOMAIN", "localhost")
	port := retrieveWithDefault("SITE_PORT", "8080")

	fileLogConfigPath := os.Getenv("LOG_CONFIG_PATH")
	if fileLogConfigPath != "" {
		logConfig, err = os.ReadFile(fileLogConfigPath)
		if err != nil {
			fmt.Println("Failed to read logging config file :", err)
			logConfig = nil
		}
	}

	sessionTimeOutStr := os.Getenv("SESSION_TIME_OUT")
	if sessionTimeOutStr == "" {
		sessionTimeOut = defaultSessionTimeOut
	} else {
		sessionTimeOut, _ = strconv.Atoi(sessionTimeOutStr)
		if sessionTimeOut == 0 {
			fmt.Println("Failed to parse SESSION_TIME_OUT, using default")
			sessionTimeOut = defaultSessionTimeOut
		}
	}

	pageSizeStr := os.Getenv("PAGE_SIZE")
	if pageSizeStr == "" {
		pageSize = defaultPageSize
	} else {
		pageSize, _ = strconv.ParseUint(pageSizeStr, 10, 64)
		if pageSize == 0 {
			fmt.Println("Failed to parse PAGE_SIZE, using default")
			pageSize = defaultPageSize
		}
	}

	dateFormat := retrieveWithDefault("DATE_FORMAT", "2/1/2006 15:04:05")

	return Config{
		Domain: domain, Port: port, LogConfig: logConfig, SessionTimeOut: sessionTimeOut,
		PageSize: pageSize, DateFormat: dateFormat,

		StaticPath:    retrievePath("STATIC_PATH", "static"),
		LocalesPath:   retrievePath("LOCALES_PATH", "locales"),
		TemplatesPath: retrievePath("TEMPLATES_PATH", "templates"),

		SessionServiceAddr: requiredFromEnv("SESSION_SERVICE_ADDR"),
	}
}

func (c *Config) LoadLogin() {
	if c.SaltServiceAddr == "" {
		c.SaltServiceAddr = requiredFromEnv("SALT_SERVICE_ADDR")
		c.LoginServiceAddr = requiredFromEnv("LOGIN_SERVICE_ADDR")
	}
}

func (c *Config) LoadRight() {
	if c.RightServiceAddr == "" {
		c.LoadLogin()
		c.RightServiceAddr = requiredFromEnv("RIGHT_SERVICE_ADDR")
	}
}

func (c *Config) LoadProfile() {
	if c.ProfileServiceAddr == "" {
		c.LoadLogin()
		c.ProfileServiceAddr = requiredFromEnv("PROFILE_SERVICE_ADDR")
	}
}

func (c *Config) LoadSettings() {
	if c.SettingsServiceAddr == "" {
		c.LoadLogin()
		c.SettingsServiceAddr = requiredFromEnv("SETTINGS_SERVICE_ADDR")
	}
}

func (c *Config) loadMarkdown() {
	if c.MarkdownServiceAddr == "" {
		c.MarkdownServiceAddr = requiredFromEnv("MARKDOWN_SERVICE_ADDR")
	}
}

func (c *Config) LoadWiki() {
	if c.WikiServiceAddr == "" {
		c.LoadRight()
		c.LoadProfile()
		c.loadMarkdown()
		c.WikiServiceAddr = requiredFromEnv("WIKI_SERVICE_ADDR")
	}
}

func (c *Config) LoadForum() {
	if c.ForumServiceAddr == "" {
		c.LoadRight()
		c.LoadProfile()
		c.ForumServiceAddr = requiredFromEnv("FORUM_SERVICE_ADDR")
	}
}

func (c *Config) LoadBlog() {
	if c.BlogServiceAddr == "" {
		c.LoadForum()
		c.loadMarkdown()
		c.BlogServiceAddr = requiredFromEnv("BLOG_SERVICE_ADDR")
	}
}

func retrieveWithDefault(name string, defaultValue string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return defaultValue
}

func retrievePath(name string, defaultValue string) string {
	if value := os.Getenv(name); value != "" {
		return checkPath(value)
	}
	return defaultValue
}

func checkPath(path string) string {
	if last := len(path) - 1; last != -1 && path[last] == '/' {
		path = path[:last]
	}
	return path
}

func requiredFromEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		fmt.Println(name, "not found in env")
		os.Exit(1)
	}
	return value
}
