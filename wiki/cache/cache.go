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

	body, err := markdownclient.Apply(markdown)
	if err != nil {
		return "", err
	}

	content.body = body
	return body, nil
}

type wikiCache struct {
	mutex sync.RWMutex
	cache map[string]*WikiContent
}

func (wiki *wikiCache) load(wikiRef string) *WikiContent {
	wiki.mutex.RLock()
	content := wiki.cache[wikiRef]
	wiki.mutex.RUnlock()
	return content
}

func (wiki *wikiCache) store(wikiRef string, content *WikiContent) {
	wiki.mutex.Lock()
	wiki.cache[wikiRef] = content
	wiki.mutex.Unlock()
}

func (wiki *wikiCache) delete(wikiRef string) {
	wiki.mutex.Lock()
	delete(wiki.cache, wikiRef)
	wiki.mutex.Unlock()
}

var wikisCache map[uint64]*wikiCache = map[uint64]*wikiCache{}

func InitWiki(wikiId uint64) {
	wikisCache[wikiId] = &wikiCache{cache: map[string]*WikiContent{}}
}

func Load(wikiId uint64, wikiRef string) *WikiContent {
	return wikisCache[wikiId].load(wikiRef)
}

func Store(wikiId uint64, wikiRef string, content *WikiContent) {
	wikiCache := wikisCache[wikiId]
	if content == nil {
		wikiCache.delete(wikiRef)
	} else {
		wikiCache.store(wikiRef, content)
	}
}
