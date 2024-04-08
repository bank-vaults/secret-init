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
	"os"

	"emperror.dev/errors"
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
	region, err := getRegion()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get region")
	}

	// SharedConfigDisable is the default state,
	// and will use the AWS_SDK_LOAD_CONFIG environment variable.
	// LoadFromSharedConfigEnv can be used to enable loading from AWS shared config.
	var sess *session.Session
	loadFromSharedConfig := cast.ToBool(os.Getenv(LoadFromSharedConfigEnv))
	if loadFromSharedConfig {
		sess = session.Must(session.NewSessionWithOptions(
			session.Options{
				Config: aws.Config{
					Region: region,
				},
				SharedConfigState: session.SharedConfigEnable,
			}))
	} else {
		sess = session.Must(session.NewSessionWithOptions(
			session.Options{
				Config: aws.Config{
					Region: region,
				},
				SharedConfigState: session.SharedConfigDisable,
			}))
	}

	return &Config{session: sess}, nil
}

func getRegion() (*string, error) {
	region, hasRegion := os.LookupEnv(RegionEnv)
	if !hasRegion {
		defaultRegion, hasDefaultRegion := os.LookupEnv(DefaultRegionEnv)
		if hasDefaultRegion {
			return &defaultRegion, nil
		}

		return nil, errors.New("no region found")
	}

	return &region, nil
}
