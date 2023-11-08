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

package widgetservice

import (
	"context"

	"github.com/gin-gonic/gin"
)

const RawResult = "RAW"

type Action struct {
	Kind       string
	Name       string
	Path       string
	QueryNames []string
}

type WidgetService interface {
	GetDesc(ctx context.Context, widgetName string) ([]Action, error)
	Process(ctx context.Context, widgetName string, actionName string, data gin.H, files map[string][]byte) (string, string, []byte, error)
}
