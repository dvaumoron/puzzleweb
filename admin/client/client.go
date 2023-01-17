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

func RegisterGroup(groupId uint64, name string) {
	for usedId := range groupIdToName {
		if groupId == usedId {
			fmt.Println("duplicate objectId")
			os.Exit(1)
		}
	}
	groupIdToName[groupId] = name
	nameToGroupId[name] = groupId
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
	var err error
	if groupId != PublicGroupId || action != ActionAccess {
		var conn *grpc.ClientConn
		conn, err = grpc.Dial(config.RightServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			defer conn.Close()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			var response *pb.Response
			response, err = pb.NewRightClient(conn).AuthQuery(ctx, &pb.RightRequest{
				UserId: userId, ObjectId: groupId, Action: action,
			})
			if err == nil {
				if !response.Success {
					err = common.ErrorNotAuthorized
				}
			} else {
				common.LogOriginalError(err)
				err = common.ErrorTechnical
			}
		} else {
			common.LogOriginalError(err)
			err = common.ErrorTechnical
		}
	}
	return err
}

func GetAllRoles(adminId uint64) ([]*Role, error) {
	groupIds := make([]uint64, 0, len(groupIdToName))
	for groupId := range groupIdToName {
		groupIds = append(groupIds, groupId)
	}
	return getGroupRoles(adminId, groupIds)
}

func GetActions(adminId uint64, roleName string, groupName string) ([]pb.RightAction, error) {
	conn, err := grpc.Dial(config.RightServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	var list []pb.RightAction
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.Response
		client := pb.NewRightClient(conn)
		response, err = client.AuthQuery(ctx, &pb.RightRequest{
			UserId: adminId, ObjectId: AdminGroupId, Action: ActionAccess,
		})
		if err == nil {
			if response.Success {
				var actions *pb.Actions
				actions, err = client.RoleRight(ctx, &pb.RoleRequest{
					Name: roleName, ObjectId: nameToGroupId[groupName],
				})
				if err == nil {
					list = actions.List
				} else {
					common.LogOriginalError(err)
					err = common.ErrorTechnical
				}
			} else {
				err = common.ErrorNotAuthorized
			}
		} else {
			common.LogOriginalError(err)
			err = common.ErrorTechnical
		}
	} else {
		common.LogOriginalError(err)
		err = common.ErrorTechnical
	}
	return list, err
}

func UpdateUser(adminId uint64, userId uint64, roles []*Role) error {
	conn, err := grpc.Dial(config.RightServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.Response
		client := pb.NewRightClient(conn)
		response, err = client.AuthQuery(ctx, &pb.RightRequest{
			UserId: adminId, ObjectId: AdminGroupId, Action: ActionUpdate,
		})
		if err == nil {
			if response.Success {
				converted := make([]*pb.RoleRequest, 0, len(roles))
				for _, role := range roles {
					converted = append(converted, &pb.RoleRequest{
						Name: role.Name, ObjectId: nameToGroupId[role.Group],
					})
				}

				response, err = client.UpdateUser(ctx, &pb.UserRight{
					UserId: userId, List: converted,
				})
				if err == nil {
					if !response.Success {
						err = common.ErrorUpdate
					}
				} else {
					common.LogOriginalError(err)
					err = common.ErrorTechnical
				}
			} else {
				err = common.ErrorNotAuthorized
			}
		} else {
			common.LogOriginalError(err)
			err = common.ErrorTechnical
		}
	} else {
		common.LogOriginalError(err)
		err = common.ErrorTechnical
	}
	return err
}

func UpdateRole(adminId uint64, role *Role) error {
	conn, err := grpc.Dial(config.RightServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.Response
		client := pb.NewRightClient(conn)
		response, err = client.AuthQuery(ctx, &pb.RightRequest{
			UserId: adminId, ObjectId: AdminGroupId, Action: ActionUpdate,
		})
		if err == nil {
			if response.Success {
				response, err = client.UpdateRole(ctx, &pb.Role{
					Name: role.Name, ObjectId: nameToGroupId[role.Group],
					List: role.Actions,
				})
				if err == nil {
					if !response.Success {
						err = common.ErrorUpdate
					}
				} else {
					common.LogOriginalError(err)
					err = common.ErrorTechnical
				}
			} else {
				err = common.ErrorNotAuthorized
			}
		} else {
			common.LogOriginalError(err)
			err = common.ErrorTechnical
		}
	} else {
		common.LogOriginalError(err)
		err = common.ErrorTechnical
	}
	return err
}

func GetUserRoles(adminId uint64, userId uint64) ([]*Role, error) {
	conn, err := grpc.Dial(config.RightServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	var roleList []*Role
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		client := pb.NewRightClient(conn)
		if adminId == userId {
			roleList, err = getUserRoles(client, ctx, userId)
		} else {
			var response *pb.Response
			response, err = client.AuthQuery(ctx, &pb.RightRequest{
				UserId: adminId, ObjectId: AdminGroupId, Action: ActionAccess,
			})
			if err == nil {
				if response.Success {
					roleList, err = getUserRoles(client, ctx, userId)
				} else {
					err = common.ErrorNotAuthorized
				}
			} else {
				common.LogOriginalError(err)
				err = common.ErrorTechnical
			}
		}
	} else {
		common.LogOriginalError(err)
		err = common.ErrorTechnical
	}
	return roleList, err
}

func getGroupRoles(adminId uint64, groupIds []uint64) ([]*Role, error) {
	conn, err := grpc.Dial(config.RightServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	var roleList []*Role
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.Response
		client := pb.NewRightClient(conn)
		response, err = client.AuthQuery(ctx, &pb.RightRequest{
			UserId: adminId, ObjectId: AdminGroupId, Action: ActionAccess,
		})
		if err == nil {
			if response.Success {
				var roles *pb.Roles
				roles, err = client.ListRoles(ctx, &pb.ObjectIds{Ids: groupIds})
				if err == nil {
					roleList = convertRolesFromRequest(roles.List)
				} else {
					common.LogOriginalError(err)
					err = common.ErrorTechnical
				}
			} else {
				err = common.ErrorNotAuthorized
			}
		} else {
			common.LogOriginalError(err)
			err = common.ErrorTechnical
		}
	} else {
		common.LogOriginalError(err)
		err = common.ErrorTechnical
	}
	return roleList, err
}

func getUserRoles(client pb.RightClient, ctx context.Context, userId uint64) ([]*Role, error) {
	var roleList []*Role
	roles, err := client.ListUserRoles(ctx, &pb.UserId{Id: userId})
	if err == nil {
		roleList = convertRolesFromRequest(roles.List)
	} else {
		common.LogOriginalError(err)
		err = common.ErrorTechnical
	}
	return roleList, err
}

func convertRolesFromRequest(roles []*pb.Role) []*Role {
	resRoles := make([]*Role, 0, len(roles))
	for _, role := range roles {
		resRoles = append(resRoles, &Role{
			Name: role.Name, Group: groupIdToName[role.ObjectId],
			Actions: role.List,
		})
	}
	return resRoles
}
