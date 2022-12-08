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
package client

import (
	"html/template"
	"strconv"
)

type WikiContent struct {
	Version  uint64
	Markdown string
	Body     template.HTML
}

func LoadContent(wikiId uint64, userId uint64, lang string, title string) (*WikiContent, error) {
	// TODO
	return &WikiContent{
		Version:  1,
		Markdown: "",
		Body:     "",
	}, nil
}

func StoreContent(wikiId uint64, userId uint64, lang string, title string, preVersion string, markdown string) error {
	ver, err := strconv.ParseUint(preVersion, 10, 64)
	if err == nil {
		// TODO
	}
	return err
}
