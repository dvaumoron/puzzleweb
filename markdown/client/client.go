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
	grpcclient "github.com/dvaumoron/puzzlegrpcclient"
	pb "github.com/dvaumoron/puzzlemarkdownservice"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/markdown/service"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"google.golang.org/grpc"
)

type markdownClient struct {
	grpcclient.Client
}

func New(serviceAddr string, dialOptions []grpc.DialOption) service.MarkdownService {
	return markdownClient{Client: grpcclient.Make(serviceAddr, dialOptions...)}
}

func (client markdownClient) Apply(logger otelzap.LoggerWithCtx, text string) (string, error) {
	conn, err := client.Dial()
	if err != nil {
		return "", common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	markdownHtml, err := pb.NewMarkdownClient(conn).Apply(logger.Context(), &pb.MarkdownText{Text: text})
	if err != nil {
		return "", common.LogOriginalError(logger, err)
	}
	return markdownHtml.Html, nil
}
