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
package cache

import (
	"html/template"
	"sync"

	"github.com/dvaumoron/puzzleweb/markdownclient"
)

type WikiContent struct {
	Version   uint64
	Markdown  string
	bodyMutex sync.RWMutex
	body      template.HTML
}

// Lazy loading for markdown application on body.
func (content *WikiContent) GetBody() (template.HTML, error) {
	var err error
	content.bodyMutex.RLock()
	body := content.body
	content.bodyMutex.RUnlock()
	if body == "" {
		if markdown := content.Markdown; markdown != "" {
			content.bodyMutex.Lock()
			if body = content.body; body == "" {
				body, err = markdownclient.Apply(markdown)
				if err == nil {
					content.body = body
				}
			}
			content.bodyMutex.Unlock()
		}
	}
	return body, err
}

var wikiCacheMutex sync.RWMutex
var wikisCache map[uint64]map[string]*WikiContent = make(map[uint64]map[string]*WikiContent)

func Load(wikiId uint64, wikiRef string) *WikiContent {
	var content *WikiContent
	wikiCacheMutex.RLock()
	wikiCache := wikisCache[wikiId]
	if wikiCache != nil {
		content = wikiCache[wikiRef]
	}
	wikiCacheMutex.RUnlock()
	return content
}

func Store(wikiId uint64, wikiRef string, content *WikiContent) {
	wikiCacheMutex.Lock()
	wikiCache := wikisCache[wikiId]
	if content == nil {
		if wikiCache != nil {
			delete(wikiCache, wikiRef)
		}
	} else {
		if wikiCache == nil {
			wikiCache = make(map[string]*WikiContent)
			wikisCache[wikiId] = wikiCache
		}
		wikiCache[wikiRef] = content
	}
	wikiCacheMutex.Unlock()
}
