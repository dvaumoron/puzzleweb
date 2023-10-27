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
	PermissionGroups []PermissionGroupConfig `hcl:"permission,block" yaml:"permissionGroups"`
	PageGroups       []PageGroupConfig       `hcl:"pageGroup,block" yaml:"pageGroups"`
	Widgets          []WidgetConfig          `hcl:"widget,block" yaml:"widgets"`
	WidgetPages      []WidgetPageConfig      `hcl:"widgetPage,block" yaml:"widgetPages"`
}

func (frame *FrameConfig) WidgetsAsMap() map[string]WidgetConfig {
	res := map[string]WidgetConfig{}
	for _, widget := range frame.Widgets {
		res[widget.Name] = widget
	}
	return res
}

type PermissionGroupConfig struct {
	Name string `hcl:"name,label" yaml:"name"`
	Id   uint64 `hcl:"groupId" yaml:"id"`
}

type PageGroupConfig struct {
	GroupId uint64   `hcl:"groupId" yaml:"groupId"`
	Pages   []string `hcl:"pages" yaml:"pages"`
}

type WidgetConfig struct {
	Name        string   `hcl:"name,label" yaml:"name"`
	Kind        string   `hcl:"kind" yaml:"kind"`
	ObjectId    uint64   `hcl:"objectId" yaml:"objectId"`
	GroupId     uint64   `hcl:"groupId" yaml:"groupId"`
	ServiceAddr string   `hcl:"serviceAddr,optional" yaml:"serviceAddr"`
	WidgetName  string   `hcl:"widgetName,optional" yaml:"widgetName"`
	Templates   []string `hcl:"templates,optional" yaml:"templates"`
}

type WidgetPageConfig struct {
	Name        string `hcl:"name,label" yaml:"name"`
	WidgetRef   string `hcl:"widgetRef" yaml:"widgetRef"`
	Emplacement string `hcl:"emplacement,optional" yaml:"emplacement"`
}

func LoadFrameConfig(path string) (FrameConfig, error) {
	var err error
	var frameConfig FrameConfig
	if strings.HasSuffix(path, ".hcl") {
		err = hclsimple.DecodeFile(path, nil, &frameConfig)
	} else {
		var frameConfigBody []byte
		frameConfigBody, err = os.ReadFile(path)
		if err == nil {
			err = yaml.Unmarshal(frameConfigBody, &frameConfig)
		}
	}
	return frameConfig, err
}
