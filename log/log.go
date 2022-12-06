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

package log

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dvaumoron/puzzleweb/config"
	"go.uber.org/zap"
)

var Logger *zap.Logger

func init() {
	defaultLogConfig := func() {
		var err error
		Logger, err = zap.NewProduction()
		if err != nil {
			fmt.Println("Failed to init logging with default config :", err)
			os.Exit(1)
		}
	}

	if len(config.LogConfig) == 0 {
		defaultLogConfig()
		return
	}

	var cfg zap.Config
	var err error = json.Unmarshal(config.LogConfig, &cfg)
	if err == nil {
		Logger, err = cfg.Build()
		if err != nil {
			fmt.Println("Failed to init logging with config file :", err)
			defaultLogConfig()
		}
	} else {
		fmt.Println("Failed to parse logging config file :", err)
		defaultLogConfig()
	}

	config.LogConfig = make([]byte, 0)
}
