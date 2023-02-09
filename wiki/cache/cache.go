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
	"sync"

	"github.com/dvaumoron/puzzleweb/wiki/service"
)

type WikiCache struct {
	mutex sync.RWMutex
	cache map[string]*service.WikiContent
}

func New() *WikiCache {
	return &WikiCache{cache: map[string]*service.WikiContent{}}
}

func (wiki *WikiCache) Load(wikiRef string) *service.WikiContent {
	wiki.mutex.RLock()
	content := wiki.cache[wikiRef]
	wiki.mutex.RUnlock()
	return content
}

func (wiki *WikiCache) Store(wikiRef string, content *service.WikiContent) {
	wiki.mutex.Lock()
	wiki.cache[wikiRef] = content
	wiki.mutex.Unlock()
}

func (wiki *WikiCache) Delete(wikiRef string) {
	wiki.mutex.Lock()
	delete(wiki.cache, wikiRef)
	wiki.mutex.Unlock()
}
