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
	"strconv"
	"strings"
	"time"

	rightclient "github.com/dvaumoron/puzzleweb/admin/client"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/errors"
	"github.com/dvaumoron/puzzleweb/log"
	loginclient "github.com/dvaumoron/puzzleweb/login/client"
	"github.com/dvaumoron/puzzleweb/wiki/cache"
	pb "github.com/dvaumoron/puzzlewikiservice"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Version struct {
	Number    uint64
	UserLogin string
}

func LoadContent(wikiId uint64, groupId uint64, userId uint64, lang string, title string, versionStr string) (*cache.WikiContent, error) {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionAccess)
	var content *cache.WikiContent
	if err == nil {
		version := uint64(0)
		if versionStr != "" {
			version, err = strconv.ParseUint(versionStr, 10, 64)
			if err != nil {
				log.Logger.Info("Failed to parse wiki version, falling to last.",
					zap.Error(err),
				)
				version = 0
			}
		}

		content, err = loadContent(wikiId, buildRef(lang, title), version)
	}
	return content, err
}

func StoreContent(wikiId uint64, groupId uint64, userId uint64, lang string, title string, last string, markdown string) error {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionCreate)
	if err == nil {
		version := uint64(0)
		version, err = strconv.ParseUint(last, 10, 64)
		if err == nil {
			err = storeContent(wikiId, buildRef(lang, title), version, markdown)
		} else {
			log.Logger.Warn("Failed to parse wiki last version.",
				zap.Error(err),
			)
			err = errors.ErrorTechnical
		}
	}
	return err
}

func GetVersions(wikiId uint64, groupId uint64, userId uint64, lang string, title string) ([]Version, error) {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionAccess)
	var versions []Version
	if err == nil {
		versions, err = getVersions(wikiId, buildRef(lang, title))
	}
	return versions, err
}

func DeleteContent(wikiId uint64, groupId uint64, userId uint64, lang string, title string, versionStr string) error {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionDelete)
	if err == nil {
		version := uint64(0)
		version, err = strconv.ParseUint(versionStr, 10, 64)
		if err == nil {
			err = deleteContent(wikiId, buildRef(lang, title), version)
		} else {
			log.Logger.Warn("Failed to parse wiki version to delete.",
				zap.Error(err),
			)
			err = errors.ErrorTechnical
		}
	}
	return err
}

func buildRef(lang, title string) string {
	var refBuilder strings.Builder
	refBuilder.WriteString(lang)
	refBuilder.WriteString("/")
	refBuilder.WriteString(title)
	return refBuilder.String()
}

func loadContent(wikiId uint64, wikiRef string, version uint64) (*cache.WikiContent, error) {
	conn, err := grpc.Dial(config.WikiServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	var content *cache.WikiContent
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		client := pb.NewWikiClient(conn)
		if version == 0 {
			var versions *pb.Versions
			versions, err = client.ListVersions(ctx, &pb.VersionRequest{
				WikiId: wikiId, WikiRef: wikiRef,
			})
			if err == nil {
				lastVersion := maxVersion(versions.List)
				if lastVersion == nil {
					content, err = innerLoadContent(ctx, client, wikiId, wikiRef, 0)
				} else {
					content = cache.Load(wikiId, wikiRef)
					if content == nil || lastVersion.Number != content.Version {
						content, err = innerLoadContent(ctx, client, wikiId, wikiRef, 0)
					}
				}
			} else {
				errors.LogOriginalError(err)
				err = errors.ErrorTechnical
			}
		} else {
			content, err = innerLoadContent(ctx, client, wikiId, wikiRef, version)
		}
	} else {
		errors.LogOriginalError(err)
		err = errors.ErrorTechnical
	}
	return content, err
}

func innerLoadContent(ctx context.Context, client pb.WikiClient, wikiId uint64, wikiRef string, version uint64) (*cache.WikiContent, error) {
	response, err := client.Load(ctx, &pb.WikiRequest{
		WikiId: wikiId, WikiRef: wikiRef, Version: version,
	})
	var content *cache.WikiContent
	if err == nil {
		content = &cache.WikiContent{
			Version: response.Version, Markdown: response.Text,
		}
		if version == 0 {
			cache.Store(wikiId, wikiRef, content)
		}
	} else {
		errors.LogOriginalError(err)
		err = errors.ErrorTechnical
	}
	return content, err
}

func storeContent(wikiId uint64, wikiRef string, last uint64, markdown string) error {
	conn, err := grpc.Dial(config.WikiServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.Confirm
		response, err = pb.NewWikiClient(conn).Store(ctx, &pb.ContentRequest{
			WikiId: wikiId, WikiRef: wikiRef, Last: last, Text: markdown,
		})
		if err == nil {
			if response.Success {
				cache.Store(wikiId, wikiRef, &cache.WikiContent{
					Version: response.Version, Markdown: markdown,
				})
			} else {
				err = errors.ErrorUpdate
			}
		} else {
			errors.LogOriginalError(err)
			err = errors.ErrorTechnical
		}
	} else {
		errors.LogOriginalError(err)
		err = errors.ErrorTechnical
	}
	return err
}

func getVersions(wikiId uint64, wikiRef string) ([]Version, error) {
	conn, err := grpc.Dial(config.WikiServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	var versions []Version
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.Versions
		response, err = pb.NewWikiClient(conn).ListVersions(ctx, &pb.VersionRequest{
			WikiId: wikiId, WikiRef: wikiRef,
		})
		if err == nil {
			list := response.List
			if len(list) != 0 {
				versions = sortConvertVersion(list)
			}
		} else {
			errors.LogOriginalError(err)
			err = errors.ErrorTechnical
		}
	} else {
		errors.LogOriginalError(err)
		err = errors.ErrorTechnical
	}
	return versions, err
}

func deleteContent(wikiId uint64, wikiRef string, version uint64) error {
	conn, err := grpc.Dial(config.WikiServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.Confirm
		response, err = pb.NewWikiClient(conn).Delete(ctx, &pb.WikiRequest{
			WikiId: wikiId, WikiRef: wikiRef, Version: version,
		})
		if err == nil {
			if response.Success {
				content := cache.Load(wikiId, wikiRef)
				if content != nil && version == content.Version {
					cache.Store(wikiId, wikiRef, nil)
				}
			} else {
				err = errors.ErrorUpdate
			}
		} else {
			errors.LogOriginalError(err)
			err = errors.ErrorTechnical
		}
	} else {
		errors.LogOriginalError(err)
		err = errors.ErrorTechnical
	}
	return err
}

func maxVersion(list []*pb.Version) *pb.Version {
	var res *pb.Version
	if len(list) != 0 {
		res = list[0]
	}
	for _, current := range list {
		if current.Number > res.Number {
			res = current
		}
	}
	return res
}

func sortConvertVersion(list []*pb.Version) []Version {
	size := len(list)
	valueSet := make([]*pb.Version, maxVersion(list).Number)
	userIds := make([]uint64, 0, size)
	for _, value := range list {
		valueSet[value.Number] = value
		userIds = append(userIds, value.UserId)
	}
	logins, err := loginclient.GetLogins(userIds)
	if err != nil {
		errors.LogOriginalError(err)
	}
	newList := make([]Version, 0, size)
	for _, value := range valueSet {
		if value != nil {
			newList = append(newList, Version{Number: value.Number, UserLogin: logins[value.UserId]})
		}
	}
	return newList
}
