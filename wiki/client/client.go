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
	"strconv"
	"strings"

	grpcclient "github.com/dvaumoron/puzzlegrpcclient"
	adminservice "github.com/dvaumoron/puzzleweb/admin/service"
	"github.com/dvaumoron/puzzleweb/common"
	profileservice "github.com/dvaumoron/puzzleweb/profile/service"
	"github.com/dvaumoron/puzzleweb/wiki/service"
	pb "github.com/dvaumoron/puzzlewikiservice"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type wikiClient struct {
	grpcclient.Client
	cache          *wikiCache
	wikiId         uint64
	groupId        uint64
	dateFormat     string
	authService    adminservice.AuthService
	profileService profileservice.ProfileService
}

func New(serviceAddr string, dialOptions []grpc.DialOption, wikiId uint64, groupId uint64, dateFormat string, authService adminservice.AuthService, profileService profileservice.ProfileService) service.WikiService {
	return wikiClient{
		Client: grpcclient.Make(serviceAddr, dialOptions...), cache: newCache(), wikiId: wikiId,
		groupId: groupId, dateFormat: dateFormat, authService: authService, profileService: profileService,
	}
}

func (client wikiClient) LoadContent(logger otelzap.LoggerWithCtx, userId uint64, lang string, title string, versionStr string) (*service.WikiContent, error) {
	err := client.authService.AuthQuery(logger, userId, client.groupId, adminservice.ActionAccess)
	if err != nil {
		return nil, err
	}

	var version uint64
	if versionStr != "" {
		version, err = strconv.ParseUint(versionStr, 10, 64)
		if err != nil {
			logger.Info("Failed to parse wiki version, falling to last", zap.Error(err))
		}
	}
	return client.loadContent(logger, buildRef(lang, title), version)
}

func (client wikiClient) StoreContent(logger otelzap.LoggerWithCtx, userId uint64, lang string, title string, last string, markdown string) (bool, error) {
	err := client.authService.AuthQuery(logger, userId, client.groupId, adminservice.ActionCreate)
	if err != nil {
		return false, err
	}

	version, err := strconv.ParseUint(last, 10, 64)
	if err != nil {
		logger.Warn("Failed to parse wiki last version", zap.Error(err))
		return false, common.ErrTechnical
	}
	return client.storeContent(logger, userId, buildRef(lang, title), version, markdown)
}

func (client wikiClient) GetVersions(logger otelzap.LoggerWithCtx, userId uint64, lang string, title string) ([]service.Version, error) {
	err := client.authService.AuthQuery(logger, userId, client.groupId, adminservice.ActionAccess)
	if err != nil {
		return nil, err
	}
	return client.getVersions(logger, buildRef(lang, title))
}

func (client wikiClient) DeleteContent(logger otelzap.LoggerWithCtx, userId uint64, lang string, title string, versionStr string) error {
	err := client.authService.AuthQuery(logger, userId, client.groupId, adminservice.ActionDelete)
	if err != nil {
		return err
	}

	version, err := strconv.ParseUint(versionStr, 10, 64)
	if err != nil {
		logger.Warn("Failed to parse wiki version to delete", zap.Error(err))
		return common.ErrTechnical
	}
	return client.deleteContent(logger, buildRef(lang, title), version)
}

func (client wikiClient) DeleteRight(logger otelzap.LoggerWithCtx, userId uint64) bool {
	return client.authService.AuthQuery(logger, userId, client.groupId, adminservice.ActionDelete) == nil
}

func (client wikiClient) loadContent(logger otelzap.LoggerWithCtx, wikiRef string, version uint64) (*service.WikiContent, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	wikiId := client.wikiId
	pbWikiClient := pb.NewWikiClient(conn)
	if version != 0 {
		return client.innerLoadContent(logger, pbWikiClient, wikiRef, version)
	}

	versions, err := pbWikiClient.ListVersions(logger.Context(), &pb.VersionRequest{
		WikiId: wikiId, WikiRef: wikiRef,
	})
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}

	if lastVersion := maxVersion(versions.List); lastVersion != nil {
		content := client.cache.load(logger, wikiRef)
		if content != nil && lastVersion.Number == content.Version {
			return content, nil
		}
	}
	return client.innerLoadContent(logger, pbWikiClient, wikiRef, 0)
}

func (client wikiClient) innerLoadContent(logger otelzap.LoggerWithCtx, pbWikiClient pb.WikiClient, wikiRef string, askedVersion uint64) (*service.WikiContent, error) {
	response, err := pbWikiClient.Load(logger.Context(), &pb.WikiRequest{
		WikiId: client.wikiId, WikiRef: wikiRef, Version: askedVersion,
	})
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}
	version := response.Version
	if version == 0 { // no stored wiki page
		return nil, nil
	}

	content := &service.WikiContent{Version: version, Markdown: response.Text}
	if askedVersion == 0 {
		client.cache.store(logger, wikiRef, content)
	}
	return content, nil
}

func (client wikiClient) storeContent(logger otelzap.LoggerWithCtx, userId uint64, wikiRef string, last uint64, markdown string) (bool, error) {
	conn, err := client.Dial()
	if err != nil {
		return false, common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewWikiClient(conn).Store(logger.Context(), &pb.ContentRequest{
		WikiId: client.wikiId, WikiRef: wikiRef, Last: last, Text: markdown, UserId: userId,
	})
	if err != nil {
		return false, common.LogOriginalError(logger, err)
	}
	success := response.Success
	if success {
		client.cache.store(logger, wikiRef, &service.WikiContent{
			Version: response.Version, Markdown: markdown,
		})
	}
	return success, nil
}

func (client wikiClient) getVersions(logger otelzap.LoggerWithCtx, wikiRef string) ([]service.Version, error) {
	conn, err := client.Dial()
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewWikiClient(conn).ListVersions(logger.Context(), &pb.VersionRequest{
		WikiId: client.wikiId, WikiRef: wikiRef,
	})
	if err != nil {
		return nil, common.LogOriginalError(logger, err)
	}
	return client.sortConvertVersions(logger, response.List)
}

func (client wikiClient) deleteContent(logger otelzap.LoggerWithCtx, wikiRef string, version uint64) error {
	conn, err := client.Dial()
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	defer conn.Close()

	response, err := pb.NewWikiClient(conn).Delete(logger.Context(), &pb.WikiRequest{
		WikiId: client.wikiId, WikiRef: wikiRef, Version: version,
	})
	if err != nil {
		return common.LogOriginalError(logger, err)
	}
	if !response.Success {
		return common.ErrUpdate
	}

	content := client.cache.load(logger, wikiRef)
	if content != nil && version == content.Version {
		client.cache.delete(logger, wikiRef)
	}
	return nil
}

func (client wikiClient) sortConvertVersions(logger otelzap.LoggerWithCtx, list []*pb.Version) ([]service.Version, error) {
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
	profiles, err := client.profileService.GetProfiles(logger, userIds)
	if err != nil {
		return nil, err
	}

	newList := make([]service.Version, 0, size)
	for _, value := range valueSet {
		if value != nil {
			newList = append(newList, service.Version{Number: value.Number, Creator: profiles[value.UserId]})
		}
	}
	return newList, nil
}

func buildRef(lang string, title string) string {
	var refBuilder strings.Builder
	refBuilder.WriteString(lang)
	refBuilder.WriteString("/")
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
