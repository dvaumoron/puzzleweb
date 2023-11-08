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

package loginservice

import "context"

type User struct {
	Id          uint64
	Login       string
	RegistredAt string
}

type UserService interface {
	GetUsers(ctx context.Context, userIds []uint64) (map[uint64]User, error)
}

type AdvancedUserService interface {
	UserService
	ListUsers(ctx context.Context, start uint64, end uint64, filter string) (uint64, []User, error)
	Delete(ctx context.Context, userId uint64) error
}

type LoginService interface {
	Verify(ctx context.Context, login string, password string) (bool, uint64, error)
	Register(ctx context.Context, login string, password string) (bool, uint64, error)
	ChangeLogin(ctx context.Context, userId uint64, oldLogin string, newLogin string, password string) error
	ChangePassword(ctx context.Context, userId uint64, login string, oldPassword string, newPassword string) error
}

type FullLoginService interface {
	LoginService
	AdvancedUserService
}

type SaltService interface {
	Salt(ctx context.Context, login string, password string) (string, error)
}
