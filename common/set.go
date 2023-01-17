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
package common

type empty = struct{}

type Set[V comparable] map[V]empty

func MakeSet[V comparable](values []V) Set[V] {
	set := Set[V]{}
	for _, value := range values {
		set[value] = empty{}
	}
	return set
}

func (s Set[V]) Add(value V) {
	s[value] = empty{}
}

func (s Set[V]) Remove(value V) {
	delete(s, value)
}

func (s Set[V]) Contains(value V) bool {
	_, exists := s[value]
	return exists
}

func (s Set[V]) Slice() []V {
	extracted := make([]V, 0, len(s))
	for value := range s {
		extracted = append(extracted, value)
	}
	return extracted
}
