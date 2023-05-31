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
	pb "github.com/dvaumoron/puzzlerightservice"
	"github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"google.golang.org/grpc"
)

// check matching with interface
var _ service.AdminService = RightClient{}

type RightClient struct {
	grpcclient.Client
	logger        *otelzap.Logger
	groupIdToName map[uint64]string
	nameToGroupId map[string]uint64
}

func Make(serviceAddr string, dialOptions []grpc.DialOption) RightClient {
	groupIdToName := map[uint64]string{
		service.PublicGroupId: service.PublicName, service.AdminGroupId: service.AdminName,
	}
	nameToGroupId := map[string]uint64{
		service.PublicName: service.PublicGroupId, service.AdminName: service.AdminGroupId,
	}
	return RightClient{
		Client: grpcclient.Make(serviceAddr, dialOptions...), groupIdToName: groupIdToName, nameToGroupId: nameToGroupId,
	}
}

func (client RightClient) RegisterGroup(groupId uint64, groupName string) {
	for usedId := range client.groupIdToName {
		if groupId == usedId {
			client.logger.Fatal("Duplicate groupId")
		}
	}
	client.groupIdToName[groupId] = groupName
	client.nameToGroupId[groupName] = groupId
}

func (client RightClient) GetGroupId(groupName string) uint64 {
	return client.nameToGroupId[groupName]
}

func (client RightClient) GetGroupName(groupId uint64) string {
	return client.groupIdToName[groupId]
}

func (client RightClient) GetAllGroups(logger otelzap.LoggerWithCtx) []service.Group {
	groups := make([]service.Group, 0, len(client.groupIdToName))
	for id, name := range client.groupIdToName {
		groups = append(groups, service.Group{Id: id, Name: name})
	}
	return groups
}

func (client RightClient) AuthQuery(logger otelzap.LoggerWithCtx, userId uint64, groupId uint64, action string) error {
	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewRightClient(conn).AuthQuery(logger.Context(), &pb.RightRequest{
		UserId: userId, ObjectId: groupId, Action: convertActionForRequest(action),
	})
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	if !response.Success {
		return common.ErrNotAuthorized
	}
	return nil
}

func (client RightClient) GetAllRoles(logger otelzap.LoggerWithCtx, adminId uint64) ([]service.Role, error) {
	groupIds := make([]uint64, 0, len(client.groupIdToName))
	for groupId := range client.groupIdToName {
		groupIds = append(groupIds, groupId)
	}
	return client.getGroupRoles(logger, adminId, groupIds)
}

func (client RightClient) GetActions(logger otelzap.LoggerWithCtx, adminId uint64, roleName string, groupName string) ([]string, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	ctx := logger.Context()
	rightClient := pb.NewRightClient(conn)
	response, err := rightClient.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: service.AdminGroupId, Action: pb.RightAction_ACCESS,
	})
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}
	if !response.Success {
		return nil, common.ErrNotAuthorized
	}

	actions, err := rightClient.RoleRight(ctx, &pb.RoleRequest{
		Name: roleName, ObjectId: client.nameToGroupId[groupName],
	})
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}
	return convertActionsFromRequest(actions.List), nil
}

func (client RightClient) UpdateUser(logger otelzap.LoggerWithCtx, adminId uint64, userId uint64, roles []service.Role) error {
	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	ctx := logger.Context()
	rightClient := pb.NewRightClient(conn)
	response, err := rightClient.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: service.AdminGroupId, Action: pb.RightAction_UPDATE,
	})
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	if !response.Success {
		return common.ErrNotAuthorized
	}

	converted := make([]*pb.RoleRequest, 0, len(roles))
	for _, role := range roles {
		converted = append(converted, &pb.RoleRequest{
			Name: role.Name, ObjectId: client.nameToGroupId[role.GroupName],
		})
	}

	response, err = rightClient.UpdateUser(ctx, &pb.UserRight{UserId: userId, List: converted})
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client RightClient) UpdateRole(logger otelzap.LoggerWithCtx, adminId uint64, role service.Role) error {
	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	ctx := logger.Context()
	rightClient := pb.NewRightClient(conn)
	response, err := rightClient.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: service.AdminGroupId, Action: pb.RightAction_UPDATE,
	})
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	if !response.Success {
		return common.ErrNotAuthorized
	}

	response, err = rightClient.UpdateRole(ctx, &pb.Role{
		Name: role.Name, ObjectId: client.nameToGroupId[role.GroupName],
		List: convertActionsForRequest(role.Actions),
	})
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client RightClient) GetUserRoles(logger otelzap.LoggerWithCtx, adminId uint64, userId uint64) ([]service.Role, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	rightClient := pb.NewRightClient(conn)
	if adminId == userId {
		return client.getUserRoles(rightClient, logger, userId)
	}

	response, err := rightClient.AuthQuery(logger.Context(), &pb.RightRequest{
		UserId: adminId, ObjectId: service.AdminGroupId, Action: pb.RightAction_ACCESS,
	})
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}
	if !response.Success {
		return nil, common.ErrNotAuthorized
	}
	return client.getUserRoles(rightClient, logger, userId)
}

func (client RightClient) getGroupRoles(logger otelzap.LoggerWithCtx, adminId uint64, groupIds []uint64) ([]service.Role, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	ctx := logger.Context()
	rightClient := pb.NewRightClient(conn)
	response, err := rightClient.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: service.AdminGroupId, Action: pb.RightAction_ACCESS,
	})
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}
	if !response.Success {
		return nil, common.ErrNotAuthorized
	}

	roles, err := rightClient.ListRoles(ctx, &pb.ObjectIds{Ids: groupIds})
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}
	return convertRolesFromRequest(roles.List, client.groupIdToName), nil
}

func (client RightClient) getUserRoles(rightClient pb.RightClient, logger otelzap.LoggerWithCtx, userId uint64) ([]service.Role, error) {
	roles, err := rightClient.ListUserRoles(logger.Context(), &pb.UserId{Id: userId})
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}
	return convertRolesFromRequest(roles.List, client.groupIdToName), nil
}

func convertRolesFromRequest(roles []*pb.Role, groupIdToName map[uint64]string) []service.Role {
	resRoles := make([]service.Role, 0, len(roles))
	for _, role := range roles {
		groupId := role.ObjectId
		resRoles = append(resRoles, service.Role{
			Name: role.Name, GroupId: groupId, GroupName: groupIdToName[groupId],
			Actions: convertActionsFromRequest(role.List),
		})
	}
	return resRoles
}

func convertActionsFromRequest(actions []pb.RightAction) []string {
	resActions := make([]string, 0, len(actions))
	for _, action := range actions {
		resActions = append(resActions, convertActionFromRequest(action))
	}
	return resActions
}

func convertActionFromRequest(action pb.RightAction) string {
	switch action {
	case pb.RightAction_ACCESS:
		return service.ActionAccess
	case pb.RightAction_CREATE:
		return service.ActionCreate
	case pb.RightAction_UPDATE:
		return service.ActionUpdate
	case pb.RightAction_DELETE:
		return service.ActionDelete
	}
	return service.ActionAccess
}

func convertActionForRequest(action string) pb.RightAction {
	switch action {
	case service.ActionAccess:
		return pb.RightAction_ACCESS
	case service.ActionCreate:
		return pb.RightAction_CREATE
	case service.ActionUpdate:
		return pb.RightAction_UPDATE
	case service.ActionDelete:
		return pb.RightAction_DELETE
	}
	return 0
}

func convertActionsForRequest(actions []string) []pb.RightAction {
	resActions := make([]pb.RightAction, 0, 4)
	// use Set to remove duplicate
	for action := range common.MakeSet(actions) {
		resActions = append(resActions, convertActionForRequest(action))
	}
	return resActions
}
