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

package file

import (
	"log/slog"
	"os"

	"github.com/bank-vaults/secret-init/common"
)

const defaultMountPath = "/"

type Config struct {
	MountPath string `json:"mountPath"`
}

func NewConfig(logger *slog.Logger) *Config {
	mountPath, ok := os.LookupEnv(common.FileMountPath)
	if !ok {
		logger.Warn("Mount path not provided. Using default.", slog.String("Default Mount Path", defaultMountPath))
		mountPath = defaultMountPath
	}

	return &Config{MountPath: mountPath}
}
