// Copyright © 2024 Bank-Vaults Maintainers
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

package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

const (
	DefaultRegionEnv = "AWS_DEFAULT_REGION"
	RegionEnv        = "AWS_REGION"
)

type Config struct {
	config aws.Config
}

func LoadConfig(ctx context.Context) (*Config, error) {
	config, err := config.LoadDefaultConfig(ctx, config.WithRegion(getRegionEnv()))
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %w", err)
	}

	return &Config{config: config}, nil
}

func getRegionEnv() string {
	region, hasRegion := os.LookupEnv(RegionEnv)
	if hasRegion {
		return region
	}

	defaultRegion, hasDefaultRegion := os.LookupEnv(DefaultRegionEnv)
	if hasDefaultRegion {
		return defaultRegion
	}

	// Return an empty string if no region is found, allowing the AWS SDK to attempt to
	// determine the region from the shared config or environment variables.
	return ""
}
