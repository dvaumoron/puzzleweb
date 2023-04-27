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
	"context"

	"github.com/uptrace/opentelemetry-go-extra/otelzap"
)

type User struct {
	Id          uint64
	Login       string
	RegistredAt string
}

type UserService interface {
	GetUsers(logger otelzap.LoggerWithCtx, userIds []uint64) (map[uint64]User, error)
}

type AdvancedUserService interface {
	UserService
	ListUsers(logger otelzap.LoggerWithCtx, start uint64, end uint64, filter string) (uint64, []User, error)
	Delete(logger otelzap.LoggerWithCtx, userId uint64) error
}

type LoginService interface {
	Verify(logger otelzap.LoggerWithCtx, login string, password string) (bool, uint64, error)
	Register(logger otelzap.LoggerWithCtx, login string, password string) (bool, uint64, error)
	ChangeLogin(logger otelzap.LoggerWithCtx, userId uint64, oldLogin string, newLogin string, password string) error
	ChangePassword(logger otelzap.LoggerWithCtx, userId uint64, login string, oldPassword string, newPassword string) error
}

type FullLoginService interface {
	LoginService
	AdvancedUserService
}

type SaltService interface {
	Salt(ctx context.Context, login string, password string) (string, error)
}
