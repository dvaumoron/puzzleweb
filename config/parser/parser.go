/*
 *
 * Copyright 2023 puzzleweb authors.
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

package parser

import (
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"gopkg.in/yaml.v3"
)

type FrameConfig struct {
	PermissionGroups []PermissionGroupConfig `hcl:"permission,block"`
	PageGroups       []PageGroupConfig       `hcl:"pageGroup,block"`
	Widgets          []WidgetConfig          `hcl:"widget,block"`
	WidgetPages      []WidgetPageConfig      `hcl:"widgetPage,block"`
}

func (frame *FrameConfig) WidgetsAsMap() map[string]WidgetConfig {
	res := map[string]WidgetConfig{}
	for _, widget := range frame.Widgets {
		res[widget.Name] = widget
	}
	return res
}

type PermissionGroupConfig struct {
	Name string `hcl:"name,label"`
	Id   uint64 `hcl:"groupId"`
}

type PageGroupConfig struct {
	GroupId uint64   `hcl:"groupId"`
	Pages   []string `hcl:"pages"`
}

type WidgetConfig struct {
	Name        string   `hcl:"name,label"`
	Kind        string   `hcl:"kind"`
	ObjectId    uint64   `hcl:"objectId"`
	GroupId     uint64   `hcl:"groupId"`
	ServiceAddr string   `hcl:"serviceAddr,optional"`
	WidgetName  string   `hcl:"widgetName,optional"`
	Templates   []string `hcl:"templates,optional"`
}

type WidgetPageConfig struct {
	Name        string `hcl:"name,label"`
	WidgetRef   string `hcl:"widgetRef"`
	Emplacement string `hcl:"emplacement,optional"`
}

func LoadFrameConfig(path string) (FrameConfig, error) {
	var err error
	var frameConfig FrameConfig
	if strings.HasSuffix(path, ".hcl") {
		err = hclsimple.DecodeFile(path, nil, &frameConfig)
	} else {
		var frameConfigBody []byte
		frameConfigBody, err = os.ReadFile(path)
		if err != nil {
			err = yaml.Unmarshal(frameConfigBody, &frameConfig)
		}
	}
	return frameConfig, err
}
