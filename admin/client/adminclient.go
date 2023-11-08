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

package adminclient

import (
	"context"

	grpcclient "github.com/dvaumoron/puzzlegrpcclient"
	pb "github.com/dvaumoron/puzzlerightservice"
	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/common/log"
	"google.golang.org/grpc"
)

// check matching with interface
var _ adminservice.AdminService = RightClient{}

type RightClient struct {
	grpcclient.Client
	logger        log.Logger // for init phase (have the context)
	groupIdToName map[uint64]string
	nameToGroupId map[string]uint64
}

func Make(serviceAddr string, dialOptions []grpc.DialOption, logger log.Logger) RightClient {
	groupIdToName := map[uint64]string{
		adminservice.PublicGroupId: adminservice.PublicName, adminservice.AdminGroupId: adminservice.AdminName,
	}
	nameToGroupId := map[string]uint64{
		adminservice.PublicName: adminservice.PublicGroupId, adminservice.AdminName: adminservice.AdminGroupId,
	}
	return RightClient{
		Client: grpcclient.Make(serviceAddr, dialOptions...), logger: logger,
		groupIdToName: groupIdToName, nameToGroupId: nameToGroupId,
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

func (client RightClient) AuthQuery(ctx context.Context, userId uint64, groupId uint64, action string) error {
	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	response, err := pb.NewRightClient(conn).AuthQuery(ctx, &pb.RightRequest{
		UserId: userId, ObjectId: groupId, Action: convertActionForRequest(action),
	})
	if err != nil {
		return err
	}
	if !response.Success {
		return common.ErrNotAuthorized
	}
	return nil
}

func (client RightClient) GetAllGroups(ctx context.Context, adminId uint64) ([]adminservice.Group, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	rightClient := pb.NewRightClient(conn)
	return client.getAllGroups(rightClient, ctx, adminId)
}

func (client RightClient) GetActions(ctx context.Context, adminId uint64, roleName string, groupName string) ([]string, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	rightClient := pb.NewRightClient(conn)
	response, err := rightClient.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: adminservice.AdminGroupId, Action: pb.RightAction_ACCESS,
	})
	if err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, common.ErrNotAuthorized
	}

	actions, err := rightClient.RoleRight(ctx, &pb.RoleRequest{
		Name: roleName, ObjectId: client.nameToGroupId[groupName],
	})
	if err != nil {
		return nil, err
	}
	return convertActionsFromRequest(actions.List), nil
}

func (client RightClient) UpdateUser(ctx context.Context, adminId uint64, userId uint64, roles []adminservice.Group) error {
	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	rightClient := pb.NewRightClient(conn)
	response, err := rightClient.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: adminservice.AdminGroupId, Action: pb.RightAction_UPDATE,
	})
	if err != nil {
		return err
	}
	if !response.Success {
		return common.ErrNotAuthorized
	}

	converted := make([]*pb.RoleRequest, 0, len(roles))
	for _, group := range roles {
		for _, role := range group.Roles {
			converted = append(converted, &pb.RoleRequest{
				Name: role.Name, ObjectId: client.nameToGroupId[group.Name],
			})
		}
	}

	response, err = rightClient.UpdateUser(ctx, &pb.UserRight{UserId: userId, List: converted})
	if err != nil {
		return err
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client RightClient) UpdateRole(ctx context.Context, adminId uint64, roleName string, groupName string, actions []string) error {
	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	rightClient := pb.NewRightClient(conn)
	response, err := rightClient.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: adminservice.AdminGroupId, Action: pb.RightAction_UPDATE,
	})
	if err != nil {
		return err
	}
	if !response.Success {
		return common.ErrNotAuthorized
	}

	response, err = rightClient.UpdateRole(ctx, &pb.Role{
		Name: roleName, ObjectId: client.nameToGroupId[groupName], List: convertActionsForRequest(actions),
	})
	if err != nil {
		return err
	}
	if !response.Success {
		return common.ErrUpdate
	}
	return nil
}

func (client RightClient) GetUserRoles(ctx context.Context, adminId uint64, userId uint64) ([]adminservice.Group, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	rightClient := pb.NewRightClient(conn)
	if adminId == userId {
		return client.getUserRoles(rightClient, ctx, userId)
	}

	response, err := rightClient.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: adminservice.AdminGroupId, Action: pb.RightAction_ACCESS,
	})
	if err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, common.ErrNotAuthorized
	}
	return client.getUserRoles(rightClient, ctx, userId)
}

func (client RightClient) ViewUserRoles(ctx context.Context, adminId uint64, userId uint64) (bool, []adminservice.Group, error) {
	conn, err := client.Dial()
	if err != nil {
		return false, nil, err
	}
	defer conn.Close()

	rightClient := pb.NewRightClient(conn)
	response, err := rightClient.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: adminservice.AdminGroupId, Action: pb.RightAction_UPDATE,
	})
	updateRight := err == nil && response.Success

	if adminId == userId {
		userRoles, err := client.getUserRoles(rightClient, ctx, userId)
		return updateRight, userRoles, err
	}

	response, err = rightClient.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: adminservice.AdminGroupId, Action: pb.RightAction_ACCESS,
	})
	if err != nil {
		return false, nil, err
	}
	if !response.Success {
		return false, nil, common.ErrNotAuthorized
	}

	userRoles, err := client.getUserRoles(rightClient, ctx, userId)
	return updateRight, userRoles, err
}

func (client RightClient) EditUserRoles(ctx context.Context, adminId uint64, userId uint64) ([]adminservice.Group, []adminservice.Group, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, nil, err
	}
	defer conn.Close()

	rightClient := pb.NewRightClient(conn)
	allRoles, err := client.getAllGroups(rightClient, ctx, adminId)
	if err != nil {
		return nil, nil, err
	}

	userRoles, err := client.getUserRoles(rightClient, ctx, userId)
	return userRoles, allRoles, err
}

func (client RightClient) getAllGroups(rightClient pb.RightClient, ctx context.Context, adminId uint64) ([]adminservice.Group, error) {
	response, err := rightClient.AuthQuery(ctx, &pb.RightRequest{
		UserId: adminId, ObjectId: adminservice.AdminGroupId, Action: pb.RightAction_ACCESS,
	})
	if err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, common.ErrNotAuthorized
	}

	groupIds := make([]uint64, 0, len(client.groupIdToName))
	for groupId := range client.groupIdToName {
		groupIds = append(groupIds, groupId)
	}

	roles, err := rightClient.ListRoles(ctx, &pb.ObjectIds{Ids: groupIds})
	if err != nil {
		return nil, err
	}
	return convertRolesFromRequest(roles.List, client.groupIdToName), nil
}

func (client RightClient) getUserRoles(rightClient pb.RightClient, ctx context.Context, userId uint64) ([]adminservice.Group, error) {
	roles, err := rightClient.ListUserRoles(ctx, &pb.UserId{Id: userId})
	if err != nil {
		return nil, err
	}
	return convertRolesFromRequest(roles.List, client.groupIdToName), nil
}

func convertRolesFromRequest(roles []*pb.Role, groupIdToName map[uint64]string) []adminservice.Group {
	groupIdToRoles := map[uint64][]adminservice.Role{}
	for _, role := range roles {
		groupId := role.ObjectId
		groupIdToRoles[groupId] = append(groupIdToRoles[groupId], adminservice.Role{
			Name: role.Name, Actions: convertActionsFromRequest(role.List),
		})
	}

	res := make([]adminservice.Group, 0, len(roles))
	for groupId, roles := range groupIdToRoles {
		res = append(res, adminservice.Group{
			Id: groupId, Name: groupIdToName[groupId], Roles: roles,
		})
	}
	return res
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
		return adminservice.ActionAccess
	case pb.RightAction_CREATE:
		return adminservice.ActionCreate
	case pb.RightAction_UPDATE:
		return adminservice.ActionUpdate
	case pb.RightAction_DELETE:
		return adminservice.ActionDelete
	}
	return adminservice.ActionAccess
}

func convertActionForRequest(action string) pb.RightAction {
	switch action {
	case adminservice.ActionAccess:
		return pb.RightAction_ACCESS
	case adminservice.ActionCreate:
		return pb.RightAction_CREATE
	case adminservice.ActionUpdate:
		return pb.RightAction_UPDATE
	case adminservice.ActionDelete:
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
