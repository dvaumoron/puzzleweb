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

	pb "github.com/dvaumoron/puzzlemarkdownservice"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/grpcclient"
	"github.com/dvaumoron/puzzleweb/markdown/service"
	"go.uber.org/zap"
)

// check matching with interface
var _ service.MarkdownService = MarkdownClient{}

type MarkdownClient struct {
	grpcclient.Client
}

func Make(serviceAddr string, logger *zap.Logger) MarkdownClient {
	return MarkdownClient{Client: grpcclient.Make(serviceAddr, logger)}
}

func (client MarkdownClient) Apply(text string) (template.HTML, error) {
	conn, err := client.Dial()
	if err != nil {
		return "", common.LogOriginalError(client.Logger, err)
	}
	defer conn.Close()

	ctx, cancel := client.InitContext()
	defer cancel()

	markdownHtml, err := pb.NewMarkdownClient(conn).Apply(ctx, &pb.MarkdownText{Text: text})
	if err != nil {
		return "", common.LogOriginalError(client.Logger, err)
	}
	return template.HTML(markdownHtml.Html), nil
}
