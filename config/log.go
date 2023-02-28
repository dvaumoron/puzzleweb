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
package config

import (
	"encoding/json"
	"fmt"
	"os"

	"go.uber.org/zap"
)

func newLogger(logConfig []byte) *zap.Logger {
	if len(logConfig) == 0 {
		return defaultLogConfig()
	}

	var cfg zap.Config
	err := json.Unmarshal(logConfig, &cfg)
	if err != nil {
		fmt.Println("Failed to parse logging config file :", err)
		return defaultLogConfig()
	}

	logger, err := cfg.Build()
	if err != nil {
		fmt.Println("Failed to init logging with config file :", err)
		return defaultLogConfig()
	}
	return logger
}

func defaultLogConfig() *zap.Logger {
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Println("Failed to init logging with default config :", err)
		os.Exit(1)
	}
	return logger
}
