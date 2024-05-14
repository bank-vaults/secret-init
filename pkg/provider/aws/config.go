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
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/spf13/cast"
)

const (
	LoadFromSharedConfigEnv = "AWS_LOAD_FROM_SHARED_CONFIG"
	DefaultRegionEnv        = "AWS_DEFAULT_REGION"
	RegionEnv               = "AWS_REGION"
)

type Config struct {
	session *session.Session
}

func LoadConfig() (*Config, error) {
	// Loading session data from shared config is disabled by default and needs to be
	// explicitly enabled via AWS_LOAD_FROM_SHARED_CONFIG
	options := session.Options{
		SharedConfigState: session.SharedConfigDisable,
	}

	// Override session options from env configs
	if cast.ToBool(os.Getenv(LoadFromSharedConfigEnv)) {
		options.SharedConfigState = session.SharedConfigEnable
	}

	if region := getRegionEnv(); region != nil {
		options.Config = aws.Config{Region: region}
	}

	// Create session
	sess, err := session.NewSessionWithOptions(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	return &Config{session: sess}, nil
}

func getRegionEnv() *string {
	region, hasRegion := os.LookupEnv(RegionEnv)
	if hasRegion {
		return aws.String(region)
	}

	defaultRegion, hasDefaultRegion := os.LookupEnv(DefaultRegionEnv)
	if hasDefaultRegion {
		return aws.String(defaultRegion)
	}

	// Return nil if no region is found, allowing the AWS SDK to attempt to
	// determine the region from the shared config or environment variables.
	return nil
}
