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
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/log"
	profileclient "github.com/dvaumoron/puzzleweb/profile/client"
	"github.com/dvaumoron/puzzleweb/wiki/cache"
	pb "github.com/dvaumoron/puzzlewikiservice"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Version struct {
	Number  uint64
	Creator profileclient.Profile
}

func LoadContent(wikiId uint64, groupId uint64, userId uint64, lang string, title string, versionStr string) (*cache.WikiContent, error) {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionAccess)
	if err != nil {
		return nil, err
	}

	var version uint64
	if versionStr != "" {
		version, err = strconv.ParseUint(versionStr, 10, 64)
		if err != nil {
			log.Logger.Info("Failed to parse wiki version, falling to last.",
				zap.Error(err),
			)
			version = 0
		}
	}
	return loadContent(wikiId, buildRef(lang, title), version)
}

func StoreContent(wikiId uint64, groupId uint64, userId uint64, lang string, title string, last string, markdown string) error {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionCreate)
	if err != nil {
		return err
	}

	version, err := strconv.ParseUint(last, 10, 64)
	if err != nil {
		log.Logger.Warn("Failed to parse wiki last version.", zap.Error(err))
		return common.ErrorTechnical
	}
	return storeContent(wikiId, userId, buildRef(lang, title), version, markdown)
}

func GetVersions(wikiId uint64, groupId uint64, userId uint64, lang string, title string) ([]Version, error) {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionAccess)
	if err != nil {
		return nil, err
	}
	return getVersions(wikiId, buildRef(lang, title))
}

func DeleteContent(wikiId uint64, groupId uint64, userId uint64, lang string, title string, versionStr string) error {
	err := rightclient.AuthQuery(userId, groupId, rightclient.ActionDelete)
	if err != nil {
		return err
	}

	version, err := strconv.ParseUint(versionStr, 10, 64)
	if err != nil {
		log.Logger.Warn("Failed to parse wiki version to delete.", zap.Error(err))
		return common.ErrorTechnical
	}
	return deleteContent(wikiId, buildRef(lang, title), version)
}

func buildRef(lang string, title string) string {
	var refBuilder strings.Builder
	refBuilder.WriteString(lang)
	refBuilder.WriteString("/")
	refBuilder.WriteString(title)
	return refBuilder.String()
}

func loadContent(wikiId uint64, wikiRef string, version uint64) (*cache.WikiContent, error) {
	conn, err := grpc.Dial(config.WikiServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrorTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client := pb.NewWikiClient(conn)
	if version != 0 {
		return innerLoadContent(ctx, client, wikiId, wikiRef, version)
	}

	versions, err := client.ListVersions(ctx, &pb.VersionRequest{
		WikiId: wikiId, WikiRef: wikiRef,
	})
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrorTechnical
	}

	if lastVersion := maxVersion(versions.List); lastVersion != nil {
		content := cache.Load(wikiId, wikiRef)
		if content != nil && lastVersion.Number == content.Version {
			return content, nil
		}
	}
	return innerLoadContent(ctx, client, wikiId, wikiRef, 0)
}

func innerLoadContent(ctx context.Context, client pb.WikiClient, wikiId uint64, wikiRef string, askedVersion uint64) (*cache.WikiContent, error) {
	response, err := client.Load(ctx, &pb.WikiRequest{
		WikiId: wikiId, WikiRef: wikiRef, Version: askedVersion,
	})
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrorTechnical
	}
	version := response.Version
	if version == 0 { // no stored wiki page
		return nil, nil
	}

	content := &cache.WikiContent{Version: version, Markdown: response.Text}
	if askedVersion == 0 {
		cache.Store(wikiId, wikiRef, content)
	}
	return content, nil
}

func storeContent(wikiId uint64, userId uint64, wikiRef string, last uint64, markdown string) error {
	conn, err := grpc.Dial(config.WikiServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrorTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewWikiClient(conn).Store(ctx, &pb.ContentRequest{
		WikiId: wikiId, WikiRef: wikiRef, Last: last, Text: markdown, UserId: userId,
	})
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrorTechnical
	}
	if !response.Success {
		return common.ErrorUpdate
	}

	cache.Store(wikiId, wikiRef, &cache.WikiContent{
		Version: response.Version, Markdown: markdown,
	})
	return nil
}

func getVersions(wikiId uint64, wikiRef string) ([]Version, error) {
	conn, err := grpc.Dial(config.WikiServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrorTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewWikiClient(conn).ListVersions(ctx, &pb.VersionRequest{
		WikiId: wikiId, WikiRef: wikiRef,
	})
	if err != nil {
		common.LogOriginalError(err)
		return nil, common.ErrorTechnical
	}
	list := response.List
	if len(list) == 0 {
		return nil, nil
	}
	return sortConvertVersions(list)
}

func deleteContent(wikiId uint64, wikiRef string, version uint64) error {
	conn, err := grpc.Dial(config.WikiServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrorTechnical
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := pb.NewWikiClient(conn).Delete(ctx, &pb.WikiRequest{
		WikiId: wikiId, WikiRef: wikiRef, Version: version,
	})
	if err != nil {
		common.LogOriginalError(err)
		return common.ErrorTechnical
	}
	if !response.Success {
		return common.ErrorUpdate
	}

	content := cache.Load(wikiId, wikiRef)
	if content != nil && version == content.Version {
		cache.Store(wikiId, wikiRef, nil)
	}
	return nil
}

func maxVersion(list []*pb.Version) *pb.Version {
	var res *pb.Version
	if len(list) != 0 {
		res = list[0]
		for _, current := range list {
			if current.Number > res.Number {
				res = current
			}
		}
	}
	return res
}

func sortConvertVersions(list []*pb.Version) ([]Version, error) {
	size := len(list)
	if size == 0 {
		return nil, nil
	}

	valueSet := make([]*pb.Version, maxVersion(list).Number)
	// no duplicate check, there is one in GetProfiles
	userIds := make([]uint64, 0, size)
	for _, value := range list {
		valueSet[value.Number] = value
		userIds = append(userIds, value.UserId)
	}
	profiles, err := profileclient.GetProfiles(userIds)
	if err != nil {
		return nil, err
	}

	newList := make([]Version, 0, size)
	for _, value := range valueSet {
		if value != nil {
			newList = append(newList, Version{Number: value.Number, Creator: profiles[value.UserId]})
		}
	}
	return newList, nil
}
