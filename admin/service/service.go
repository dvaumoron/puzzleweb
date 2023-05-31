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

import "github.com/uptrace/opentelemetry-go-extra/otelzap"

const AdminName = "admin"
const PublicName = "public"
const PublicGroupId = 0 // groupId for content always allowed to access
const AdminGroupId = 1  // groupId corresponding to role administration

const (
	ActionAccess = "access"
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"
)

type Group struct {
	Id   uint64
	Name string
}

type Role struct {
	Name      string
	GroupId   uint64
	GroupName string
	Actions   []string
}

type AuthService interface {
	AuthQuery(logger otelzap.LoggerWithCtx, userId uint64, groupId uint64, action string) error
}

type AdminService interface {
	AuthService
	GetAllGroups(logger otelzap.LoggerWithCtx) []Group
	GetAllRoles(logger otelzap.LoggerWithCtx, adminId uint64) ([]Role, error)
	GetActions(logger otelzap.LoggerWithCtx, adminId uint64, roleName string, groupName string) ([]string, error)
	UpdateUser(logger otelzap.LoggerWithCtx, adminId uint64, userId uint64, roles []Role) error
	UpdateRole(logger otelzap.LoggerWithCtx, adminId uint64, role Role) error
	GetUserRoles(logger otelzap.LoggerWithCtx, adminId uint64, userId uint64) ([]Role, error)
}
