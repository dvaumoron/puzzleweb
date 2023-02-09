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

import loginservice "github.com/dvaumoron/puzzleweb/login/service"

type UserProfile struct {
	loginservice.User
	Desc string
	Info map[string]string
}

type ProfileService interface {
	GetProfiles([]uint64) (map[uint64]UserProfile, error)
}

type PictureService interface {
	GetPicture(userId uint64) ([]byte, error)
}

type AdvancedProfileService interface {
	ProfileService
	PictureService
	UpdateProfile(userId uint64, desc string, info map[string]string) error
	UpdatePicture(userId uint64, data []byte) error
	Delete(userId uint64) error
	ViewRight(userId uint64) error
}