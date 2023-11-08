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

package profileservice

import (
	"context"

	loginservice "github.com/dvaumoron/puzzleweb/login/service"
)

type UserProfile struct {
	loginservice.User
	Desc string
	Info map[string]string
}

type ProfileService interface {
	GetProfiles(ctx context.Context, userIds []uint64) (map[uint64]UserProfile, error)
}

type AdvancedProfileService interface {
	ProfileService
	GetPicture(ctx context.Context, userId uint64) []byte
	UpdateProfile(ctx context.Context, userId uint64, desc string, info map[string]string) error
	UpdatePicture(ctx context.Context, userId uint64, data []byte) error
	Delete(ctx context.Context, userId uint64) error
	ViewRight(ctx context.Context, userId uint64) error
}
