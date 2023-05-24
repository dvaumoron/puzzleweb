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
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/remotewidget/service"
	pb "github.com/dvaumoron/puzzlewidgetservice"
	"github.com/gin-gonic/gin"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"google.golang.org/grpc"
)

type widgetClient struct {
	grpcclient.Client
	objectId uint64
	groupId  uint64
}

func New(serviceAddr string, dialOptions []grpc.DialOption, objectId uint64, groupId uint64) service.WidgetService {
	return widgetClient{Client: grpcclient.Make(serviceAddr, dialOptions...), objectId: objectId, groupId: groupId}
}

func (client widgetClient) GetDesc(logger otelzap.LoggerWithCtx, name string) ([]service.Action, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewWidgetClient(conn).GetWidget(logger.Context(), &pb.WidgetRequest{Name: name})
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}
	return convertActions(response.Actions), nil
}

func (client widgetClient) Process(logger otelzap.LoggerWithCtx, widgetName string, actionName string, data gin.H) (string, string, []byte, error) {
	// TODO
	return "", "", nil, nil
}

func convertActions(actions []*pb.Action) []service.Action {
	// TODO
	return nil
}
