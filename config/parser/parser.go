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

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"gopkg.in/yaml.v3"
)

type FrameConfig struct {
	PermissionGroups []PermissionGroupConfig `hcl:"permission,block" yaml:"permissionGroups"`
	StaticPages      []StaticPagesConfig     `hcl:"staticPages,block" yaml:"staticPages"`
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

type StaticPagesConfig struct {
	GroupId   uint64   `hcl:"groupId" yaml:"groupId"`
	Locations []string `hcl:"locations" yaml:"locations"`
}

type WidgetConfig struct {
	Name        string   `hcl:"name,label" yaml:"name"`
	Kind        string   `hcl:"kind" yaml:"kind"`
	ObjectId    uint64   `hcl:"objectId" yaml:"objectId"`
	GroupId     uint64   `hcl:"groupId" yaml:"groupId"`
	ServiceAddr string   `hcl:"serviceAddr,optional" yaml:"serviceAddr"`
	Templates   []string `hcl:"templates,optional" yaml:"templates"`
}

type WidgetPageConfig struct {
	Path      string `hcl:"path,label" yaml:"path"`
	WidgetRef string `hcl:"widgetRef" yaml:"widgetRef"`
}

func LoadFrameConfig(path string) (FrameConfig, error) {
	var err error
	var frameConfig FrameConfig
	if strings.HasSuffix(path, ".hcl") {
		err = hclsimple.DecodeFile(path, &configContext, &frameConfig)
	} else {
		var frameConfigBody []byte
		frameConfigBody, err = os.ReadFile(path)
		if err == nil {
			err = yaml.Unmarshal(frameConfigBody, &frameConfig)
		}
	}
	return frameConfig, err
}

var configContext = hcl.EvalContext{Functions: map[string]function.Function{"env": function.New(&function.Spec{
	Params: []function.Parameter{{Name: "key", Type: cty.String}},
	Type:   function.StaticReturnType(cty.String),
	Impl:   wrappedGetenv,
})}}

func wrappedGetenv(args []cty.Value, retType cty.Type) (cty.Value, error) {
	return cty.StringVal(os.Getenv(args[0].AsString())), nil
}
