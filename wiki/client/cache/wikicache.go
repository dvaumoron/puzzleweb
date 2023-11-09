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

package wikicache

import (
	"sync"

	"github.com/dvaumoron/puzzleweb/common/log"
	wikiservice "github.com/dvaumoron/puzzleweb/wiki/service"
	"go.uber.org/zap"
)

const wikiRefName = "wikiRef"

type WikiCache struct {
	mutex sync.RWMutex
	cache map[string]*wikiservice.WikiContent
}

func NewCache() *WikiCache {
	return &WikiCache{cache: map[string]*wikiservice.WikiContent{}}
}

func (wiki *WikiCache) Load(logger log.Logger, wikiRef string) *wikiservice.WikiContent {
	wiki.mutex.RLock()
	content, ok := wiki.cache[wikiRef]
	wiki.mutex.RUnlock()
	if !ok {
		logger.Debug("wikiCache miss", zap.String(wikiRefName, wikiRef))
	}
	return content
}

func (wiki *WikiCache) Store(logger log.Logger, wikiRef string, content *wikiservice.WikiContent) {
	wiki.mutex.Lock()
	wiki.cache[wikiRef] = content
	wiki.mutex.Unlock()
	logger.Debug("wikiCache store", zap.String(wikiRefName, wikiRef))
}

func (wiki *WikiCache) Delete(logger log.Logger, wikiRef string) {
	wiki.mutex.Lock()
	delete(wiki.cache, wikiRef)
	wiki.mutex.Unlock()
	logger.Debug("wikiCache delete", zap.String(wikiRefName, wikiRef))
}
