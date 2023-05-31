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

package service

import (
	"html/template"
	"sync"

	markdownservice "github.com/dvaumoron/puzzleweb/markdown/service"
	profileservice "github.com/dvaumoron/puzzleweb/profile/service"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
)

type WikiContent struct {
	Version   uint64
	Markdown  string
	bodyMutex sync.RWMutex
	body      template.HTML
}

// Lazy loading for markdown application on body.
func (content *WikiContent) GetBody(logger otelzap.LoggerWithCtx, markdownService markdownservice.MarkdownService) (template.HTML, error) {
	content.bodyMutex.RLock()
	body := content.body
	content.bodyMutex.RUnlock()
	if body != "" {
		return body, nil
	}
	markdown := content.Markdown
	if markdown == "" {
		return "", nil
	}

	content.bodyMutex.Lock()
	defer content.bodyMutex.Unlock()
	if body = content.body; body != "" {
		return body, nil
	}

	body, err := markdownService.Apply(logger, markdown)
	if err != nil {
		return "", err
	}

	content.body = body
	return body, nil
}

type Version struct {
	Number  uint64
	Creator profileservice.UserProfile
}

type WikiService interface {
	LoadContent(logger otelzap.LoggerWithCtx, userId uint64, lang string, title string, versionStr string) (*WikiContent, error)
	StoreContent(logger otelzap.LoggerWithCtx, userId uint64, lang string, title string, last string, markdown string) (bool, error)
	GetVersions(logger otelzap.LoggerWithCtx, userId uint64, lang string, title string) ([]Version, error)
	DeleteContent(logger otelzap.LoggerWithCtx, userId uint64, lang string, title string, versionStr string) error
	DeleteRight(logger otelzap.LoggerWithCtx, userId uint64) bool
}
