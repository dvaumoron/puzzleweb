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

package templateclient

import (
	"context"
	"encoding/json"

	grpcclient "github.com/dvaumoron/puzzlegrpcclient"
	pb "github.com/dvaumoron/puzzletemplateservice"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/common/log"
	templateservice "github.com/dvaumoron/puzzleweb/templates/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type templateClient struct {
	grpcclient.Client
	loggerGetter log.LoggerGetter
}

func New(serviceAddr string, dialOptions []grpc.DialOption, loggerGetter log.LoggerGetter) templateservice.TemplateService {
	return templateClient{Client: grpcclient.Make(serviceAddr, dialOptions...), loggerGetter: loggerGetter}
}

func (client templateClient) Render(ctx context.Context, templateName string, data any) ([]byte, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		client.loggerGetter.Logger(ctx).Error("Failed to marshal data", zap.Error(err))
		return nil, common.ErrTechnical
	}

	conn, err := client.Dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	response, err := pb.NewTemplateClient(conn).Render(
		ctx, &pb.RenderRequest{TemplateName: templateName, Data: dataBytes},
	)
	if err != nil {
		return nil, err
	}
	return response.Content, nil
}
