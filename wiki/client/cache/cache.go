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

	"github.com/dvaumoron/puzzleweb/markdownclient"
)

type WikiContent struct {
	Version  uint64
	Markdown string
	body     template.HTML
}

// Lazy loading of Body.
func (content *WikiContent) GetBody() (template.HTML, error) {
	// TODO sync apply call
	var err error
	body := content.body
	if body == "" {
		if markdown := content.Markdown; markdown != "" {
			body, err = markdownclient.Apply(markdown)
		}
	}
	return body, err
}

// TODO sync cache
var wikisCache map[uint64]map[string]*WikiContent = make(map[uint64]map[string]*WikiContent)

func Load(wikiId uint64, wikiRef string) *WikiContent {
	var content *WikiContent
	wikiCache := wikisCache[wikiId]
	if wikiCache != nil {
		content = wikiCache[wikiRef]
	}
	return content
}

func Store(wikiId uint64, wikiRef string, content *WikiContent) {
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
}
