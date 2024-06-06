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

package bitwarden

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/gofrs/uuid"
)

const (
	AccessTokenEnv     = "BITWARDEN_ACCESS_TOKEN"
	OrganizationIDEnv  = "BITWARDEN_ORGANIZATION_ID"
	ProjectNameEnv     = "BITWARDEN_PROJECT_NAME"
	APIURLEnv          = "BITWARDEN_API_URL"
	IdentityURLEnv     = "BITWARDEN_IDENTITY_URL"
	StatePathEnv       = "BITWARDEN_STATE_PATH"
	DefaultProjectName = "default"
	DefaultAPIURL      = "127.0.0.1:8400"
)

type Config struct {
	AccessToken    string
	OrganizationID uuid.UUID
	ProjectName    string
	APIURL         string
	IdentityURL    string
	StatePath      string
}

func LoadConfig() (*Config, error) {
	AccessToken, ok := os.LookupEnv(AccessTokenEnv)
	if !ok {
		return nil, fmt.Errorf("AccessToken environment variable not provided")
	}

	var organizationID uuid.UUID
	var err error
	OrganizationIDStr, ok := os.LookupEnv(OrganizationIDEnv)
	if ok {
		organizationID, err = uuid.FromString(OrganizationIDStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse OrganizationID: %w", err)
		}
	}

	projectName, ok := os.LookupEnv(ProjectNameEnv)
	if !ok {
		slog.Warn("ProjectName environment variable not provided, using default", slog.String("project-name", DefaultProjectName))
		projectName = DefaultProjectName
	}

	apiURL, ok := os.LookupEnv(APIURLEnv)
	if !ok {
		slog.Warn("APIURL environment variable not provided, using default", slog.String("api-url", DefaultAPIURL))
		apiURL = DefaultAPIURL
	}

	return &Config{
		AccessToken:    AccessToken,
		OrganizationID: organizationID,
		ProjectName:    projectName,
		APIURL:         apiURL,
		IdentityURL:    os.Getenv(IdentityURLEnv),
		StatePath:      os.Getenv(StatePathEnv),
	}, nil
}
