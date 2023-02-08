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
	"context"
	"time"

	pb "github.com/dvaumoron/puzzlerightservice"
	"github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/common"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// check matching with interface
var _ service.AdminService = AdminClient{}

type AdminClient struct {
	serviceAddr   string
	logger        *zap.Logger
	groupIdToName map[uint64]string
	nameToGroupId map[string]uint64
}

func Make(serviceAddr string, logger *zap.Logger) AdminClient {
	groupIdToName := map[uint64]string{
		service.PublicGroupId: service.PublicName, service.AdminGroupId: service.AdminName,
	}
	nameToGroupId := map[string]uint64{
		service.PublicName: service.PublicGroupId, service.AdminName: service.AdminGroupId,
	}
	return AdminClient{
		serviceAddr: serviceAddr, logger: logger, groupIdToName: groupIdToName, nameToGroupId: nameToGroupId,
	}
}

func (client AdminClient) RegisterGroup(groupId uint64, groupName string) {
	for usedId := range client.groupIdToName {
		if groupId == usedId {
			client.logger.Fatal("Duplicate groupId.")
		}
	}
	client.groupIdToName[groupId] = groupName
	client.nameToGroupId[groupName] = groupId
}

func (client AdminClient) GetGroupId(groupName string) uint64 {
	return client.nameToGroupId[groupName]
}

func (client AdminClient) GetGroupName(groupId uint64) string {
	return client.groupIdToName[groupId]
}

func (client AdminClient) AuthQuery(userId uint64, groupId uint64, action pb.RightAction) error {
	conn, err := grpc.Dial(client.serviceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return common.LogOriginalError(client.logger, err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewRightClient(conn).AuthQuery(ctx, &pb.RightRequest{
		UserId: userId, ObjectId: groupId, Action: action,
	})
	if err != nil {
		return common.LogOriginalError(client.logger, err)
	}
	if !response.Success {
		return common.ErrNotAuthorized
	}
	return nil
}

func (client AdminClient) GetAllRoles(adminId uint64) ([]service.Role, error) {
	groupIds := make([]uint64, 0, len(client.groupIdToName))
	for groupId := range client.groupIdToName {
		groupIds = append(groupIds, groupId)
	}
	return client.getGroupRoles(adminId, groupIds)
}

func (client AdminClient) GetActions(adminId uint64, roleName string, groupName string) ([]pb.RightAction, error) {
	conn, err := grpc.Dial(client.serviceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, common.LogOriginalError(client.logger, err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	rightClient := pb.NewRightClient(conn)
	response, err := rightClient.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: service.AdminGroupId, Action: pb.RightAction_ACCESS,
	})
	if err != nil {
		return nil, common.LogOriginalError(client.logger, err)
	}
	if !response.Success {
		return nil, common.ErrNotAuthorized
	}

	actions, err := rightClient.RoleRight(ctx, &pb.RoleRequest{
		Name: roleName, ObjectId: client.nameToGroupId[groupName],
	})
	if err != nil {
		return nil, common.LogOriginalError(client.logger, err)
	}
	return actions.List, nil
}

func (client AdminClient) UpdateUser(adminId uint64, userId uint64, roles []service.Role) error {
	conn, err := grpc.Dial(client.serviceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return common.LogOriginalError(client.logger, err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	rightClient := pb.NewRightClient(conn)
	response, err := rightClient.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: service.AdminGroupId, Action: pb.RightAction_UPDATE,
	})
	if err != nil {
		return common.LogOriginalError(client.logger, err)
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
		return common.LogOriginalError(client.logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client AdminClient) UpdateRole(adminId uint64, role service.Role) error {
	conn, err := grpc.Dial(client.serviceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return common.LogOriginalError(client.logger, err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	rightClient := pb.NewRightClient(conn)
	response, err := rightClient.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: service.AdminGroupId, Action: pb.RightAction_UPDATE,
	})
	if err != nil {
		return common.LogOriginalError(client.logger, err)
	}
	if !response.Success {
		return common.ErrNotAuthorized
	}

	response, err = rightClient.UpdateRole(ctx, &pb.Role{
		Name: role.Name, ObjectId: client.nameToGroupId[role.GroupName], List: role.Actions,
	})
	if err != nil {
		return common.LogOriginalError(client.logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client AdminClient) GetUserRoles(adminId uint64, userId uint64) ([]service.Role, error) {
	conn, err := grpc.Dial(client.serviceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, common.LogOriginalError(client.logger, err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	rightClient := pb.NewRightClient(conn)
	if adminId == userId {
		return client.getUserRoles(rightClient, ctx, userId)
	}

	response, err := rightClient.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: service.AdminGroupId, Action: pb.RightAction_ACCESS,
	})
	if err != nil {
		return nil, common.LogOriginalError(client.logger, err)
	}
	if !response.Success {
		return nil, common.ErrNotAuthorized
	}
	return client.getUserRoles(rightClient, ctx, userId)
}

func (client AdminClient) getGroupRoles(adminId uint64, groupIds []uint64) ([]service.Role, error) {
	conn, err := grpc.Dial(client.serviceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, common.LogOriginalError(client.logger, err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	rightClient := pb.NewRightClient(conn)
	response, err := rightClient.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: service.AdminGroupId, Action: pb.RightAction_ACCESS,
	})
	if err != nil {
		return nil, common.LogOriginalError(client.logger, err)
	}
	if !response.Success {
		return nil, common.ErrNotAuthorized
	}

	roles, err := rightClient.ListRoles(ctx, &pb.ObjectIds{Ids: groupIds})
	if err != nil {
		return nil, common.LogOriginalError(client.logger, err)
	}
	return convertRolesFromRequest(roles.List, client.groupIdToName), nil
}

func (client AdminClient) getUserRoles(rightClient pb.RightClient, ctx context.Context, userId uint64) ([]service.Role, error) {
	roles, err := rightClient.ListUserRoles(ctx, &pb.UserId{Id: userId})
	if err != nil {
		return nil, common.LogOriginalError(client.logger, err)
	}
	return convertRolesFromRequest(roles.List, client.groupIdToName), nil
}

func convertRolesFromRequest(roles []*pb.Role, groupIdToName map[uint64]string) []service.Role {
	resRoles := make([]service.Role, 0, len(roles))
	for _, role := range roles {
		groupId := role.ObjectId
		resRoles = append(resRoles, service.Role{
			Name: role.Name, GroupId: groupId, GroupName: groupIdToName[groupId], Actions: role.List,
		})
	}
	return resRoles
}
