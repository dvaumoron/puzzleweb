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
package markdownclient

import (
	"context"
	"html/template"
	"time"

	pb "github.com/dvaumoron/puzzlemarkdownservice"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Apply(text string) (template.HTML, error) {
	conn, err := grpc.Dial(config.MarkdownServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return "", common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	markdownHtml, err := pb.NewMarkdownClient(conn).Apply(ctx, &pb.MarkdownText{Text: text})
	if err != nil {
		common.LogOriginalError(err)
		return "", common.ErrTechnical
	}
	return template.HTML(markdownHtml.Html), nil
}
