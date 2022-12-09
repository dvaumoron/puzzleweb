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
package rightclient

import (
	"context"
	"time"

	pb "github.com/dvaumoron/puzzlerightservice"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	ActionAccess = pb.RightAction_ACCESS
	ActionCreate = pb.RightAction_CREATE
	ActionUpdate = pb.RightAction_UPDATE
	ActionDelete = pb.RightAction_DELETE
)

const roleAdminObjectId = 1 // ObjectId corresponding to role administration

type Role struct {
	Name     string
	ObjectId uint64
	Actions  []pb.RightAction
}

func AuthQuery(userId uint64, objectId uint64, action pb.RightAction) (bool, error) {
	conn, err := grpc.Dial(config.RightServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	b := false
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.RightResponse
		response, err = pb.NewRightClient(conn).AuthQuery(ctx, &pb.RightRequest{
			UserId: userId, ObjectId: objectId, Action: action,
		})
		if err == nil {
			b = response.Authorized
		} else {
			err = errors.ErrorTechnical
		}
	} else {
		err = errors.ErrorTechnical
	}
	return b, err
}

func GetRoles(adminId uint64, objectIds []uint64) ([]*Role, error) {
	conn, err := grpc.Dial(config.RightServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	var roleList []*Role
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.RightResponse
		client := pb.NewRightClient(conn)
		response, err = client.AuthQuery(ctx, &pb.RightRequest{
			UserId: adminId, ObjectId: roleAdminObjectId, Action: ActionAccess,
		})
		if err == nil {
			if response.Authorized {
				var roles *pb.Roles
				roles, err = client.ListRoles(ctx, &pb.ObjectIds{Ids: objectIds})
				if err == nil {
					list := roles.List
					roleList = make([]*Role, 0, len(list))
					for _, role := range list {
						roleList = append(roleList, &Role{
							Name: role.Name, ObjectId: role.ObjectId,
							Actions: role.List.List,
						})
					}
				} else {
					err = errors.ErrorTechnical
				}
			} else {
				err = errors.ErrorNotAuthorized
			}
		} else {
			err = errors.ErrorTechnical
		}
	} else {
		err = errors.ErrorTechnical
	}
	return roleList, err
}

func GetActions(adminId uint64, roleName string, objectId uint64) ([]pb.RightAction, error) {
	conn, err := grpc.Dial(config.RightServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	var list []pb.RightAction
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.RightResponse
		client := pb.NewRightClient(conn)
		response, err = client.AuthQuery(ctx, &pb.RightRequest{
			UserId: adminId, ObjectId: roleAdminObjectId, Action: ActionAccess,
		})
		if err == nil {
			if response.Authorized {
				var actions *pb.Actions
				actions, err = client.RoleRight(ctx, &pb.RoleRequest{Name: roleName, ObjectId: objectId})
				if err == nil {
					list = actions.List
				} else {
					err = errors.ErrorTechnical
				}
			} else {
				err = errors.ErrorNotAuthorized
			}
		} else {
			err = errors.ErrorTechnical
		}
	} else {
		err = errors.ErrorTechnical
	}
	return list, err
}

func UpdateUser(adminId uint64, userId uint64, roles []*Role) error {
	conn, err := grpc.Dial(config.RightServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.RightResponse
		client := pb.NewRightClient(conn)
		response, err = client.AuthQuery(ctx, &pb.RightRequest{
			UserId: adminId, ObjectId: roleAdminObjectId, Action: ActionUpdate,
		})
		if err == nil {
			if response.Authorized {
				converted := make([]*pb.RoleRequest, 0, len(roles))
				for _, role := range roles {
					converted = append(converted, &pb.RoleRequest{
						Name: role.Name, ObjectId: role.ObjectId,
					})
				}

				response, err = client.UpdateUser(ctx, &pb.UserRight{
					UserId: userId, List: converted,
				})
				if err == nil {
					if !response.Authorized {
						err = errors.ErrorUpdate
					}
				} else {
					err = errors.ErrorTechnical
				}
			} else {
				err = errors.ErrorNotAuthorized
			}
		} else {
			err = errors.ErrorTechnical
		}
	} else {
		err = errors.ErrorTechnical
	}
	return err
}

func UpdateRole(adminId uint64, role Role) error {
	conn, err := grpc.Dial(config.RightServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.RightResponse
		client := pb.NewRightClient(conn)
		response, err = client.AuthQuery(ctx, &pb.RightRequest{
			UserId: adminId, ObjectId: roleAdminObjectId, Action: ActionUpdate,
		})
		if err == nil {
			if response.Authorized {
				response, err = client.UpdateRole(ctx, &pb.Role{
					Name: role.Name, ObjectId: role.ObjectId,
					List: &pb.Actions{List: role.Actions},
				})
				if err == nil {
					if !response.Authorized {
						err = errors.ErrorUpdate
					}
				} else {
					err = errors.ErrorTechnical
				}
			} else {
				err = errors.ErrorNotAuthorized
			}
		} else {
			err = errors.ErrorTechnical
		}
	} else {
		err = errors.ErrorTechnical
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

		var response *pb.RightResponse
		client := pb.NewRightClient(conn)
		response, err = client.AuthQuery(ctx, &pb.RightRequest{
			UserId: adminId, ObjectId: roleAdminObjectId, Action: ActionAccess,
		})
		if err == nil {
			if response.Authorized {
				var roles *pb.Roles
				roles, err = client.ListUserRoles(ctx, &pb.UserId{Id: userId})
				if err == nil {
					list := roles.List
					roleList = make([]*Role, 0, len(list))
					for _, role := range list {
						roleList = append(roleList, &Role{
							Name: role.Name, ObjectId: role.ObjectId,
							Actions: role.List.List,
						})
					}
				} else {
					err = errors.ErrorTechnical
				}
			} else {
				err = errors.ErrorNotAuthorized
			}
		} else {
			err = errors.ErrorTechnical
		}
	} else {
		err = errors.ErrorTechnical
	}
	return roleList, err
}
