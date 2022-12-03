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

package puzzleweb

import "strings"

type PageTree struct {
	Name     string
	SubPages []PageTree
	Widget   Widget
}

func MakePage(name string) PageTree {
	return PageTree{Name: name, SubPages: make([]PageTree, 0)}
}

func (pt *PageTree) SetWidget(widget Widget) {
	pt.Widget = widget
}

func (pt *PageTree) AddSubPage(page PageTree) {
	pt.SubPages = append(pt.SubPages, page)
}

func (pt *PageTree) getSubPage(name string) *PageTree {
	for _, sub := range pt.SubPages {
		if sub.Name == name {
			return &sub
		}
	}
	return nil
}

func extractPageAndPath(root *PageTree, path string) (*PageTree, []string) {
	current := root
	splitted := strings.Split(path, "/")
	names := make([]string, 0, len(splitted))
	for _, name := range splitted {
		subPage := current.getSubPage(name)
		if subPage == nil {
			break
		}
		current = subPage
		names = append(names, name)
	}
	return current, names
}
