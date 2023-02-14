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

func MapToValueSlice[K comparable, V any](objects map[K]V) []V {
	res := make([]V, 0, len(objects))
	for _, object := range objects {
		res = append(res, object)
	}
	return res
}

type Stack[T any] struct {
	inner []T
}

func (s *Stack[T]) Push(e T) {
	s.inner = append(s.inner, e)
}

func (s *Stack[T]) Peek() T {
	return s.inner[len(s.inner)-1]
}

func (s *Stack[T]) Pop() T {
	last := len(s.inner) - 1
	res := s.inner[last]
	s.inner = s.inner[:last]
	return res
}

func NewStack[T any]() *Stack[T] {
	return &Stack[T]{}
}
