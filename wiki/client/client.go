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
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/dvaumoron/puzzleweb/config"
	"github.com/dvaumoron/puzzleweb/errors"
	"github.com/dvaumoron/puzzleweb/log"
	"github.com/dvaumoron/puzzleweb/markdownclient"
	"github.com/dvaumoron/puzzleweb/rightclient"
	pb "github.com/dvaumoron/puzzlewikiservice"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const VersionName = "version"

type WikiContent struct {
	Version  uint64
	Markdown string
	Body     template.HTML
}

// Lazy loading of Body.
func (content *WikiContent) GetBody() (template.HTML, error) {
	// TODO sync apply call
	var err error
	body := content.Body
	if body == "" {
		if markdown := content.Markdown; markdown != "" {
			body, err = markdownclient.Apply(markdown)
		}
	}
	return body, err
}

// TODO sync cache
var wikisCache map[uint64]map[string]*WikiContent = make(map[uint64]map[string]*WikiContent)

func LoadContent(wikiId uint64, userId uint64, lang string, title string, version string) (*WikiContent, error) {
	authorized, err := rightclient.AuthQuery(userId, wikiId, rightclient.ActionAccess)
	var content *WikiContent
	if err == nil {
		if authorized {
			ver := uint64(0)
			if version != "" {
				ver, err = strconv.ParseUint(version, 10, 64)
				if err != nil {
					log.Logger.Info("Failed to parse wiki version, falling to last.",
						zap.String(VersionName, version),
					)
					ver = 0
				}
			}

			content, err = loadContent(wikiId, buildRef(lang, title), ver)
		} else {
			err = errors.ErrorNotAuthorized
		}
	}
	return content, err
}

func StoreContent(wikiId uint64, userId uint64, lang string, title string, last string, markdown string) error {
	authorized, err := rightclient.AuthQuery(userId, wikiId, rightclient.ActionCreate)
	if err == nil {
		if authorized {
			ver := uint64(0)
			ver, err = strconv.ParseUint(last, 10, 64)
			if err == nil {
				err = storeContent(wikiId, buildRef(lang, title), ver, markdown)
			}
		} else {
			err = errors.ErrorNotAuthorized
		}
	}
	return err
}

func GetVersions(wikiId uint64, userId uint64, lang string, title string) ([]uint64, error) {
	authorized, err := rightclient.AuthQuery(userId, wikiId, rightclient.ActionAccess)
	var versions []uint64
	if err == nil {
		if authorized {
			versions, err = getVersions(wikiId, buildRef(lang, title))
		} else {
			err = errors.ErrorNotAuthorized
		}
	}
	return versions, err
}

func DeleteContent(wikiId uint64, userId uint64, lang string, title string, version string) error {
	authorized, err := rightclient.AuthQuery(userId, wikiId, rightclient.ActionDelete)
	if err == nil {
		if authorized {
			ver := uint64(0)
			ver, err = strconv.ParseUint(version, 10, 64)
			if err == nil {
				err = deleteContent(wikiId, buildRef(lang, title), ver)
			}
		} else {
			err = errors.ErrorNotAuthorized
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

func loadContent(wikiId uint64, wikiRef string, version uint64) (*WikiContent, error) {
	conn, err := grpc.Dial(config.WikiServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	var content *WikiContent
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
				lastVersion := getLastVersion(versions.List)

				content = loadCacheContent(wikiId, wikiRef)
				if content == nil || lastVersion != content.Version {
					content, err = innerLoadContent(ctx, client, wikiId, wikiRef, 0)
				}
			}
		} else {
			content, err = innerLoadContent(ctx, client, wikiId, wikiRef, version)
		}
	}
	return content, err
}

func innerLoadContent(ctx context.Context, client pb.WikiClient, wikiId uint64, wikiRef string, version uint64) (*WikiContent, error) {
	response, err := client.Load(ctx, &pb.WikiRequest{
		WikiId: wikiId, WikiRef: wikiRef, Version: version,
	})
	var content *WikiContent
	if err == nil {
		var html template.HTML
		text := response.Text
		html, err = markdownclient.Apply(text)
		if err == nil {
			content = &WikiContent{
				Version: response.Version, Markdown: text, Body: html,
			}
			if version == 0 {
				storeCacheContent(wikiId, wikiRef, content)
			}
		}
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
				storeCacheContent(wikiId, wikiRef, &WikiContent{
					Version: last, Markdown: markdown, Body: "",
				})
			} else {
				err = errors.ErrorUpdate
			}
		}
	}
	return err
}

func getVersions(wikiId uint64, wikiRef string) ([]uint64, error) {
	conn, err := grpc.Dial(config.WikiServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	var versions []uint64
	if err == nil {
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var response *pb.Versions
		response, err = pb.NewWikiClient(conn).ListVersions(ctx, &pb.VersionRequest{
			WikiId: wikiId, WikiRef: wikiRef,
		})
		if err == nil {
			versions = response.List
		}
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
				content := loadCacheContent(wikiId, wikiRef)
				if content != nil && version == content.Version {
					storeCacheContent(wikiId, wikiRef, nil)
				}
			} else {
				err = errors.ErrorUpdate
			}
		}
	}
	return err
}

func getLastVersion(versions []uint64) uint64 {
	version := uint64(0)
	for _, current := range versions {
		if current > version {
			current = version
		}
	}
	return version
}

func loadCacheContent(wikiId uint64, wikiRef string) *WikiContent {
	var content *WikiContent
	wikiCache := wikisCache[wikiId]
	if wikiCache != nil {
		content = wikiCache[wikiRef]
	}
	return content
}

func storeCacheContent(wikiId uint64, wikiRef string, content *WikiContent) {
	wikiCache := wikisCache[wikiId]
	if content == nil {
		if wikiCache != nil {
			delete(wikiCache, wikiRef)
		}
	} else {
		if wikiCache == nil {
			wikiCache = make(map[string]*WikiContent)
			wikisCache[wikiId] = wikiCache
		}
		wikiCache[wikiRef] = content
	}
}
