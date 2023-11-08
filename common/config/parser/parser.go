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
	"errors"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"gopkg.in/yaml.v3"
)

type ParsedConfig struct {
	Domain string `hcl:"domain,optional" yaml:"domain"`
	Port   string `hcl:"port,optional" yaml:"port"`

	SessionTimeOut     int    `hcl:"sessionTimeOut,optional" yaml:"sessionTimeOut"`
	ServiceTimeOut     string `hcl:"serviceTimeOut,optional" yaml:"serviceTimeOut"`
	MaxMultipartMemory int64  `hcl:"maxMultipartMemory,optional" yaml:"maxMultipartMemory"`
	DateFormat         string `hcl:"dateFormat,optional" yaml:"dateFormat"`
	PageSize           uint64 `hcl:"pageSize,optional" yaml:"pageSize"`
	ExtractSize        uint64 `hcl:"extractSize,optional" yaml:"extractSize"`
	FeedFormat         string `hcl:"feedFormat,optional" yaml:"feedFormat"`
	FeedSize           uint64 `hcl:"feedSize,optional" yaml:"feedSize"`

	StaticPath  string `hcl:"staticPath,optional" yaml:"staticPath"`
	FaviconPath string `hcl:"faviconPath,optional" yaml:"faviconPath"`
	Page404Url  string `hcl:"page404Url,optional" yaml:"page404Url"`

	ProfileGroupId            uint64 `hcl:"profileGroupId,optional" yaml:"profileGroupId"`
	ProfileDefaultPicturePath string `hcl:"profileDefaultPicturePath,optional" yaml:"profileDefaultPicturePath"`

	SessionServiceAddr          string `hcl:"sessionServiceAddr,optional" yaml:"sessionServiceAddr"`
	TemplateServiceAddr         string `hcl:"templateServiceAddr,optional" yaml:"templateServiceAddr"`
	PasswordStrengthServiceAddr string `hcl:"passwordStrengthServiceAddr,optional" yaml:"passwordStrengthServiceAddr"`
	SaltServiceAddr             string `hcl:"saltServiceAddr,optional" yaml:"saltServiceAddr"`
	LoginServiceAddr            string `hcl:"loginServiceAddr,optional" yaml:"loginServiceAddr"`
	RightServiceAddr            string `hcl:"rightServiceAddr,optional" yaml:"rightServiceAddr"`
	SettingsServiceAddr         string `hcl:"settingsServiceAddr,optional" yaml:"settingsServiceAddr"`
	ProfileServiceAddr          string `hcl:"profileServiceAddr,optional" yaml:"profileServiceAddr"`
	ForumServiceAddr            string `hcl:"forumServiceAddr" yaml:"forumServiceAddr"`
	MarkdownServiceAddr         string `hcl:"markdownServiceAddr" yaml:"markdownServiceAddr"`
	BlogServiceAddr             string `hcl:"blogServiceAddr" yaml:"blogServiceAddr"`
	WikiServiceAddr             string `hcl:"wikiServiceAddr" yaml:"wikiServiceAddr"`

	Locales          []LocaleConfig          `hcl:"locale,block" yaml:"locales"`
	PermissionGroups []PermissionGroupConfig `hcl:"permission,block" yaml:"permissionGroups"`
	StaticPages      []StaticPagesConfig     `hcl:"staticPages,block" yaml:"staticPages"`
	Widgets          []WidgetConfig          `hcl:"widget,block" yaml:"widgets"`
	WidgetPages      []WidgetPageConfig      `hcl:"widgetPage,block" yaml:"widgetPages"`
}

func (frame *ParsedConfig) WidgetsAsMap() map[string]WidgetConfig {
	res := map[string]WidgetConfig{}
	for _, widget := range frame.Widgets {
		res[widget.Name] = widget
	}
	return res
}

type LocaleConfig struct {
	Lang        string `hcl:"lang,label" yaml:"lang"`
	PicturePath string `hcl:"picturePath" yaml:"picturePath"`
}

type PermissionGroupConfig struct {
	Name string `hcl:"name,label" yaml:"name"`
	Id   uint64 `hcl:"groupId" yaml:"id"`
}

type StaticPagesConfig struct {
	GroupId   uint64   `hcl:"groupId" yaml:"groupId"`
	Hidden    bool     `hcl:"hidden,optional" yaml:"hidden"`
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

func ParseConfig(path string) (ParsedConfig, error) {
	var err error
	var frameConfig ParsedConfig
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

var configContext = hcl.EvalContext{Functions: map[string]function.Function{
	"env": function.New(&function.Spec{
		Params: []function.Parameter{{Name: "key", Type: cty.String}},
		Type:   function.StaticReturnType(cty.String),
		Impl:   wrappedGetenv,
	}),
	"envAsNumber": function.New(&function.Spec{
		Params: []function.Parameter{{Name: "key", Type: cty.String}},
		Type:   function.StaticReturnType(cty.Number),
		Impl:   wrappedGetenv,
	}),
	"envAsBool": function.New(&function.Spec{
		Params: []function.Parameter{{Name: "key", Type: cty.String}},
		Type:   function.StaticReturnType(cty.Bool),
		Impl:   wrappedGetenv,
	}),
}}

func wrappedGetenv(args []cty.Value, retType cty.Type) (cty.Value, error) {
	valueStr := os.Getenv(args[0].AsString())
	switch retType {
	case cty.String:
		return cty.StringVal(valueStr), nil
	case cty.Number:
		return cty.ParseNumberVal(valueStr)
	case cty.Bool:
		return cty.BoolVal(valueStr != ""), nil
	}
	return cty.NilVal, errors.New("unreachable")
}
