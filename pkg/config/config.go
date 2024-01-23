// Copyright © 2023 Bank-Vaults Maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"os"
	"time"

	"github.com/spf13/cast"
)

const (
	LogLevelEnv  = "SECRET_INIT_LOG_LEVEL"
	JSONLogEnv   = "SECRET_INIT_JSON_LOG"
	LogServerEnv = "SECRET_INIT_LOG_SERVER"
	DaemonEnv    = "SECRET_INIT_DAEMON"
	DelayEnv     = "SECRET_INIT_DELAY"
	ProviderEnv  = "SECRET_INIT_PROVIDER"
)

type Config struct {
	LogLevel  string        `json:"log_level"`
	JSONLog   bool          `json:"json_log"`
	LogServer string        `json:"log_server"`
	Daemon    bool          `json:"daemon"`
	Delay     time.Duration `json:"delay"`
	Provider  string        `json:"provider"`
}

func LoadConfig() (*Config, error) {
	return &Config{
		LogLevel:  os.Getenv(LogLevelEnv),
		JSONLog:   cast.ToBool(os.Getenv(JSONLogEnv)),
		LogServer: os.Getenv(LogServerEnv),
		Daemon:    cast.ToBool(os.Getenv(DaemonEnv)),
		Delay:     cast.ToDuration(os.Getenv(DelayEnv)),
		Provider:  os.Getenv(ProviderEnv),
	}, nil
}
