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

package widgetclient

import (
	"context"
	"encoding/json"
	"net/http"

	grpcclient "github.com/dvaumoron/puzzlegrpcclient"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/common/log"
	widgetservice "github.com/dvaumoron/puzzleweb/remotewidget/service"
	pb "github.com/dvaumoron/puzzlewidgetservice"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type widgetClient struct {
	grpcclient.Client
	objectId     uint64
	groupId      uint64
	loggerGetter log.LoggerGetter
}

func New(serviceAddr string, dialOptions []grpc.DialOption, objectId uint64, groupId uint64, loggerGetter log.LoggerGetter) widgetservice.WidgetService {
	return widgetClient{Client: grpcclient.Make(serviceAddr, dialOptions...), objectId: objectId, groupId: groupId, loggerGetter: loggerGetter}
}

func (client widgetClient) GetDesc(ctx context.Context, name string) ([]widgetservice.Action, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	response, err := pb.NewWidgetClient(conn).GetWidget(ctx, &pb.WidgetRequest{Name: name})
	if err != nil {
		return nil, err
	}
	return convertActions(response.Actions), nil
}

func (client widgetClient) Process(ctx context.Context, widgetName string, actionName string, data gin.H, files map[string][]byte) (string, string, []byte, error) {
	data["objectId"] = client.objectId
	data["groupId"] = client.groupId
	dataBytes, err := json.Marshal(data)
	if err != nil {
		client.loggerGetter.Logger(ctx).Error("Failed to marshal data", zap.Error(err))
		return "", "", nil, common.ErrTechnical
	}

	files["puzzledata.json"] = dataBytes

	conn, err := client.Dial()
	if err != nil {
		return "", "", nil, err
	}
	defer conn.Close()

	response, err := pb.NewWidgetClient(conn).Process(ctx, &pb.ProcessRequest{
		WidgetName: widgetName, ActionName: actionName, Files: files,
	})
	if err != nil {
		return "", "", nil, err
	}
	return response.Redirect, response.TemplateName, response.Data, nil
}

func convertActions(actions []*pb.Action) []widgetservice.Action {
	res := make([]widgetservice.Action, 0, len(actions))
	for _, action := range actions {
		res = append(res, widgetservice.Action{
			Kind: converKind(action.Kind), Name: action.Name, Path: action.Path, QueryNames: action.QueryNames},
		)
	}
	return res
}

func converKind(kind pb.MethodKind) string {
	switch kind {
	case pb.MethodKind_HEAD:
		return http.MethodHead
	case pb.MethodKind_POST:
		return http.MethodPost
	case pb.MethodKind_PUT:
		return http.MethodPut
	case pb.MethodKind_PATCH:
		return http.MethodPatch
	case pb.MethodKind_DELETE:
		return http.MethodDelete
	case pb.MethodKind_CONNECT:
		return http.MethodConnect
	case pb.MethodKind_OPTIONS:
		return http.MethodOptions
	case pb.MethodKind_TRACE:
		return http.MethodTrace
	case pb.MethodKind_RAW:
		return widgetservice.RawResult
	}
	return http.MethodGet
}
