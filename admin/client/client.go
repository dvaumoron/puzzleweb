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
	"fmt"
	"os"
	"time"

	pb "github.com/dvaumoron/puzzlerightservice"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	ActionAccess = pb.RightAction_ACCESS
	ActionCreate = pb.RightAction_CREATE
	ActionUpdate = pb.RightAction_UPDATE
	ActionDelete = pb.RightAction_DELETE
)

const AdminName = "admin"
const PublicName = "public"
const PublicGroupId = 0 // groupId for content always allowed to access
const AdminGroupId = 1  // groupId corresponding to role administration

var groupIdToName = map[uint64]string{PublicGroupId: PublicName, AdminGroupId: AdminName}
var nameToGroupId = map[string]uint64{PublicName: PublicGroupId, AdminName: AdminGroupId}

func RegisterGroup(groupId uint64, groupName string) {
	for usedId := range groupIdToName {
		if groupId == usedId {
			fmt.Println("duplicate groupId")
			os.Exit(1)
		}
	}
	groupIdToName[groupId] = groupName
	nameToGroupId[groupName] = groupId
}

func GetGroupId(groupName string) uint64 {
	return nameToGroupId[groupName]
}

func GetGroupName(groupId uint64) string {
	return groupIdToName[groupId]
}

type Role struct {
	Name    string
	Group   string
	Actions []pb.RightAction
}

func AuthQuery(userId uint64, groupId uint64, action pb.RightAction) error {
	if groupId == PublicGroupId && action == ActionAccess {
		return nil
	}
	if userId == 0 {
		return common.ErrNotAuthorized
	}

	conn, err := grpc.Dial(config.Shared.RightServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewRightClient(conn).AuthQuery(ctx, &pb.RightRequest{
		UserId: userId, ObjectId: groupId, Action: action,
	})
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	if !response.Success {
		return common.ErrNotAuthorized
	}
	return nil
}

func GetAllRoles(adminId uint64) ([]Role, error) {
	groupIds := make([]uint64, 0, len(groupIdToName))
	for groupId := range groupIdToName {
		groupIds = append(groupIds, groupId)
	}
	return getGroupRoles(adminId, groupIds)
}

func GetActions(adminId uint64, roleName string, groupName string) ([]pb.RightAction, error) {
	conn, err := grpc.Dial(config.Shared.RightServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client := pb.NewRightClient(conn)
	response, err := client.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: AdminGroupId, Action: ActionAccess,
	})
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrTechnical
	}
	if !response.Success {
		return nil, common.ErrNotAuthorized
	}

	actions, err := client.RoleRight(ctx, &pb.RoleRequest{
		Name: roleName, ObjectId: nameToGroupId[groupName],
	})
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrTechnical
	}
	return actions.List, nil
}

func UpdateUser(adminId uint64, userId uint64, roles []Role) error {
	conn, err := grpc.Dial(config.Shared.RightServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client := pb.NewRightClient(conn)
	response, err := client.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: AdminGroupId, Action: ActionUpdate,
	})
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	if !response.Success {
		return common.ErrNotAuthorized
	}

	converted := make([]*pb.RoleRequest, 0, len(roles))
	for _, role := range roles {
		converted = append(converted, &pb.RoleRequest{
			Name: role.Name, ObjectId: nameToGroupId[role.Group],
		})
	}

	response, err = client.UpdateUser(ctx, &pb.UserRight{
		UserId: userId, List: converted,
	})
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func UpdateRole(adminId uint64, role Role) error {
	conn, err := grpc.Dial(config.Shared.RightServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client := pb.NewRightClient(conn)
	response, err := client.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: AdminGroupId, Action: ActionUpdate,
	})
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	if !response.Success {
		return common.ErrNotAuthorized
	}

	response, err = client.UpdateRole(ctx, &pb.Role{
		Name: role.Name, ObjectId: nameToGroupId[role.Group],
		List: role.Actions,
	})
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrTechnical
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func GetUserRoles(adminId uint64, userId uint64) ([]Role, error) {
	conn, err := grpc.Dial(config.Shared.RightServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client := pb.NewRightClient(conn)
	if adminId == userId {
		return getUserRoles(client, ctx, userId)
	}

	response, err := client.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: AdminGroupId, Action: ActionAccess,
	})
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrTechnical
	}
	if !response.Success {
		return nil, common.ErrNotAuthorized
	}
	return getUserRoles(client, ctx, userId)
}

func getGroupRoles(adminId uint64, groupIds []uint64) ([]Role, error) {
	conn, err := grpc.Dial(config.Shared.RightServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client := pb.NewRightClient(conn)
	response, err := client.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: AdminGroupId, Action: ActionAccess,
	})
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrTechnical
	}
	if !response.Success {
		return nil, common.ErrNotAuthorized
	}

	roles, err := client.ListRoles(ctx, &pb.ObjectIds{Ids: groupIds})
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrTechnical
	}
	return convertRolesFromRequest(roles.List), nil
}

func getUserRoles(client pb.RightClient, ctx context.Context, userId uint64) ([]Role, error) {
	roles, err := client.ListUserRoles(ctx, &pb.UserId{Id: userId})
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrTechnical
	}
	return convertRolesFromRequest(roles.List), nil
}

func convertRolesFromRequest(roles []*pb.Role) []Role {
	resRoles := make([]Role, 0, len(roles))
	for _, role := range roles {
		resRoles = append(resRoles, Role{
			Name: role.Name, Group: groupIdToName[role.ObjectId], Actions: role.List,
		})
	}
	return resRoles
}
