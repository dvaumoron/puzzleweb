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

package wikiclient

import (
	"context"
	"strconv"
	"strings"

	grpcclient "github.com/dvaumoron/puzzlegrpcclient"
	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/common"
	"github.com/dvaumoron/puzzleweb/common/log"
	profileservice "github.com/dvaumoron/puzzleweb/profile/service"
	wikicache "github.com/dvaumoron/puzzleweb/wiki/client/cache"
	wikiservice "github.com/dvaumoron/puzzleweb/wiki/service"
	pb "github.com/dvaumoron/puzzlewikiservice"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type wikiClient struct {
	grpcclient.Client
	cache          *wikicache.WikiCache
	wikiId         uint64
	groupId        uint64
	dateFormat     string
	authService    adminservice.AuthService
	profileService profileservice.ProfileService
	loggerGetter   log.LoggerGetter
}

func New(serviceAddr string, dialOptions []grpc.DialOption, wikiId uint64, groupId uint64, dateFormat string, authService adminservice.AuthService, profileService profileservice.ProfileService, loggerGetter log.LoggerGetter) wikiservice.WikiService {
	return wikiClient{
		Client: grpcclient.Make(serviceAddr, dialOptions...), cache: wikicache.NewCache(), wikiId: wikiId, groupId: groupId,
		dateFormat: dateFormat, authService: authService, profileService: profileService, loggerGetter: loggerGetter,
	}
}

func (client wikiClient) LoadContent(ctx context.Context, userId uint64, lang string, title string, versionStr string) (*wikiservice.WikiContent, error) {
	err := client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionAccess)
	if err != nil {
		return nil, err
	}

	logger := client.loggerGetter.Logger(ctx)

	var version uint64
	if versionStr != "" {
		version, err = strconv.ParseUint(versionStr, 10, 64)
		if err != nil {
			logger.Info("Failed to parse wiki version, falling to last", zap.Error(err))
		}
	}

	wikiRef := buildRef(lang, title)

	conn, err := client.Dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	wikiId := client.wikiId
	pbWikiClient := pb.NewWikiClient(conn)
	if version != 0 {
		return client.innerLoadContent(ctx, pbWikiClient, wikiRef, version)
	}

	versions, err := pbWikiClient.ListVersions(ctx, &pb.VersionRequest{
		WikiId: wikiId, WikiRef: wikiRef,
	})
	if err != nil {
		return nil, err
	}

	if lastVersion := maxVersion(versions.List); lastVersion != nil {
		content := client.cache.Load(logger, wikiRef)
		if content != nil && lastVersion.Number == content.Version {
			return content, nil
		}
	}
	return client.innerLoadContent(ctx, pbWikiClient, wikiRef, 0)
}

func (client wikiClient) StoreContent(ctx context.Context, userId uint64, lang string, title string, lastStr string, markdown string) error {
	err := client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionCreate)
	if err != nil {
		return err
	}

	logger := client.loggerGetter.Logger(ctx)

	last, err := strconv.ParseUint(lastStr, 10, 64)
	if err != nil {
		logger.Warn("Failed to parse wiki last version", zap.Error(err))
		return common.ErrTechnical
	}

	wikiRef := buildRef(lang, title)

	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	response, err := pb.NewWikiClient(conn).Store(ctx, &pb.ContentRequest{
		WikiId: client.wikiId, WikiRef: wikiRef, Last: last, Text: markdown, UserId: userId,
	})
	if err != nil {
		return err
	}
	if !response.Success {
		return common.ErrBaseVersion
	}

	client.cache.Store(logger, wikiRef, &wikiservice.WikiContent{
		Version: response.Version, Markdown: markdown,
	})
	return nil
}

func (client wikiClient) GetVersions(ctx context.Context, userId uint64, lang string, title string) ([]wikiservice.Version, error) {
	err := client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionAccess)
	if err != nil {
		return nil, err
	}

	wikiRef := buildRef(lang, title)

	conn, err := client.Dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	response, err := pb.NewWikiClient(conn).ListVersions(ctx, &pb.VersionRequest{
		WikiId: client.wikiId, WikiRef: wikiRef,
	})
	if err != nil {
		return nil, err
	}
	return client.sortConvertVersions(ctx, response.List)
}

func (client wikiClient) DeleteContent(ctx context.Context, userId uint64, lang string, title string, versionStr string) error {
	err := client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionDelete)
	if err != nil {
		return err
	}

	logger := client.loggerGetter.Logger(ctx)

	version, err := strconv.ParseUint(versionStr, 10, 64)
	if err != nil {
		logger.Warn("Failed to parse wiki version to delete", zap.Error(err))
		return common.ErrTechnical
	}

	wikiRef := buildRef(lang, title)

	conn, err := client.Dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	response, err := pb.NewWikiClient(conn).Delete(ctx, &pb.WikiRequest{
		WikiId: client.wikiId, WikiRef: wikiRef, Version: version,
	})
	if err != nil {
		return err
	}
	if !response.Success {
		return common.ErrUpdate
	}

	content := client.cache.Load(logger, wikiRef)
	if content != nil && version == content.Version {
		client.cache.Delete(logger, wikiRef)
	}
	return nil
}

func (client wikiClient) DeleteRight(ctx context.Context, userId uint64) bool {
	return client.authService.AuthQuery(ctx, userId, client.groupId, adminservice.ActionDelete) == nil
}

func (client wikiClient) innerLoadContent(ctx context.Context, pbWikiClient pb.WikiClient, wikiRef string, askedVersion uint64) (*wikiservice.WikiContent, error) {
	response, err := pbWikiClient.Load(ctx, &pb.WikiRequest{
		WikiId: client.wikiId, WikiRef: wikiRef, Version: askedVersion,
	})
	if err != nil {
		return nil, err
	}
	version := response.Version
	if version == 0 { // no stored wiki page
		return nil, nil
	}

	logger := client.loggerGetter.Logger(ctx)
	content := &wikiservice.WikiContent{Version: version, Markdown: response.Text}
	if askedVersion == 0 {
		client.cache.Store(logger, wikiRef, content)
	}
	return content, nil
}

func (client wikiClient) sortConvertVersions(ctx context.Context, list []*pb.Version) ([]wikiservice.Version, error) {
	size := len(list)
	if size == 0 {
		return nil, nil
	}

	valueSet := make([]*pb.Version, maxVersion(list).Number+1)
	// no duplicate check, there is one in GetProfiles
	userIds := make([]uint64, 0, size)
	for _, value := range list {
		valueSet[value.Number] = value
		userIds = append(userIds, value.UserId)
	}
	profiles, err := client.profileService.GetProfiles(ctx, userIds)
	if err != nil {
		return nil, err
	}

	newList := make([]wikiservice.Version, 0, size)
	for _, value := range valueSet {
		if value != nil {
			newList = append(newList, wikiservice.Version{Number: value.Number, Creator: profiles[value.UserId]})
		}
	}
	return newList, nil
}

func buildRef(lang string, title string) string {
	var refBuilder strings.Builder
	refBuilder.WriteString(lang)
	refBuilder.WriteByte('/')
	refBuilder.WriteString(title)
	return refBuilder.String()
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
