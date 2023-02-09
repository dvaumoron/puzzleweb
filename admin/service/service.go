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
package service

import (
	pb "github.com/dvaumoron/puzzlerightservice"
)

const AdminName = "admin"
const PublicName = "public"
const PublicGroupId = 0 // groupId for content always allowed to access
const AdminGroupId = 1  // groupId corresponding to role administration

type Role struct {
	Name      string
	GroupId   uint64
	GroupName string
	Actions   []pb.RightAction
}

type AuthService interface {
	AuthQuery(userId uint64, groupId uint64, action pb.RightAction) error
}

type AdminService interface {
	AuthService
	GetAllRoles(adminId uint64) ([]Role, error)
	GetActions(adminId uint64, roleName string, groupName string) ([]pb.RightAction, error)
	UpdateUser(adminId uint64, userId uint64, roles []Role) error
	UpdateRole(adminId uint64, role Role) error
	GetUserRoles(adminId uint64, userId uint64) ([]Role, error)
}
