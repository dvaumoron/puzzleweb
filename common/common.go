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

import (
	"net/http"
	"strconv"
	"unicode"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
)

const RedirectName = "Redirect"
const BaseUrlName = "BaseUrl"
const AllowedToCreateName = "AllowedToCreate"
const AllowedToUpdateName = "AllowedToUpdate"
const AllowedToDeleteName = "AllowedToDelete"

const IdName = "Id" // current connected user id
const ViewedUserName = "ViewedUser"

var htmlVoidElement = MakeSet([]string{"area", "base", "br", "col", "embed", "hr", "img", "input", "keygen", "link", "meta", "param", "source", "track", "wbr"})

type DataAdder func(gin.H, *gin.Context)
type Redirecter func(*gin.Context) string
type TemplateRedirecter func(gin.H, *gin.Context) (string, string)

func GetCurrentUrl(c *gin.Context) string {
	path := c.Request.URL.Path
	if path[len(path)-1] != '/' {
		path += "/"
	}
	return path
}

func GetBaseUrl(levelToErase uint8, c *gin.Context) string {
	res := GetCurrentUrl(c)
	i := len(res) - 1
	var count uint8
	for count < levelToErase {
		i--
		if res[i] == '/' {
			count++
		}
	}
	return res[:i+1]
}

func checkTarget(target string) string {
	if target == "" {
		target = "/"
	}
	return target
}

func CreateRedirect(redirecter Redirecter) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Redirect(http.StatusFound, checkTarget(redirecter(c)))
	}
}

func CreateRedirectString(target string) gin.HandlerFunc {
	target = checkTarget(target)
	return func(c *gin.Context) {
		c.Redirect(http.StatusFound, target)
	}
}

func GetPagination(defaultPageSize uint64, c *gin.Context) (uint64, uint64, uint64, string) {
	pageNumber, _ := strconv.ParseUint(c.Query("pageNumber"), 10, 64)
	if pageNumber == 0 {
		pageNumber = 1
	}
	pageSize, _ := strconv.ParseUint(c.Query("pageSize"), 10, 64)
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	filter := c.Query("filter")

	start := (pageNumber - 1) * pageSize
	end := start + pageSize

	return pageNumber, start, end, filter
}

func InitPagination(data gin.H, filter string, pageNumber uint64, end uint64, total uint64) {
	data["Filter"] = filter
	if pageNumber != 1 {
		data["PreviousPageNumber"] = pageNumber - 1
	}
	if end < total {
		data["NextPageNumber"] = pageNumber + 1
	}
	data["Total"] = total
}

// html must be well formed
func FilterExtractHtml(html string, extractSize uint64) string {
	buffer := make([]rune, 0, len(html))
	chars := make(chan rune)
	go sendChar(chars, html)
	var count uint64
	tagStack := NewStack[string]()
	for char := range chars {
		if char == '<' {
			char2 := <-chars
			if char2 == '/' {
				buffer = append(buffer, '<', '/')
				buffer, _ = copyTagName(buffer, chars)
				buffer = append(buffer, '>')
				tagStack.Pop()
			} else {
				temp := make([]rune, 0, 20)
				temp = append(temp, char2)
				temp, notEnded := copyTagName(temp, chars)
				if tagName := string(temp); !htmlVoidElement.Contains(tagName) {
					tagStack.Push(tagName)
				}
				buffer = append(buffer, '<')
				tempLen := len(temp)
				buffer = slices.Grow(buffer, tempLen)
				bufferLen := len(buffer)
				buffer = buffer[:bufferLen+tempLen]
				copy(buffer[bufferLen:], temp)
				if notEnded {
					buffer = append(buffer, ' ')
					buffer = copyTagAttribute(buffer, chars)
				}
				buffer = append(buffer, '>')
			}
		} else {
			buffer = append(buffer, char)
			count++
			if count > extractSize {
				buffer = append(buffer, '.', '.', '.')
				break
			}
		}
	}

	for !tagStack.Empty() {
		buffer = append(buffer, '<', '/')
		buffer = append(buffer, []rune(tagStack.Pop())...)
		buffer = append(buffer, '>')
	}

	return string(buffer)
}

func sendChar(chars chan<- rune, s string) {
	for _, char := range s {
		chars <- char
	}
	close(chars)
}

func copyTagName(buffer []rune, chars <-chan rune) ([]rune, bool) {
	notEnded := true
	for char := range chars {
		if unicode.IsSpace(char) {
			break
		}
		if char == '>' {
			notEnded = false
			break
		}
		buffer = append(buffer, char)
	}
	return buffer, notEnded
}

func copyTagAttribute(buffer []rune, chars <-chan rune) []rune {
	for char := range chars {
		if char == '>' {
			break
		}
		buffer = append(buffer, char)
	}
	return buffer
}
