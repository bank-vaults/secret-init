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

package gcp

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"

	"github.com/bank-vaults/secret-init/pkg/common"
	"github.com/bank-vaults/secret-init/pkg/provider"
)

const (
	ProviderType      = "gcp"
	referenceSelector = "gcp:secretmanager:"
	versionRegex      = `.*/versions/(latest|\d+)$`
)

type Provider struct {
	client *secretmanager.Client
}

func NewProvider(ctx context.Context, _ *common.Config) (provider.Provider, error) {
	// This will automatically use the Application Default Credentials (ADC) strategy for authentication.
	// If the GOOGLE_APPLICATION_CREDENTIALS environment variable is set,
	// the client will use the service account key JSON file that the variable points to.
	// If the environment variable is not set, the client will use the default
	// service account provided by Compute Engine, Google Kubernetes Engine,
	// App Engine, Cloud Run, and Cloud Functions, if the application is running on one of those services.
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret manager client: %v", err)
	}

	return &Provider{client: client}, nil
}

func (p *Provider) LoadSecrets(ctx context.Context, paths []string) ([]provider.Secret, error) {
	defer p.client.Close()

	var secrets []provider.Secret

	for _, path := range paths {
		split := strings.SplitN(path, "=", 2)
		originalKey, secretID := split[0], split[1]

		// valid google cloud secret manager secret examples:
		// gcp:secretmanager:projects/{PROJECT_ID}/secrets/{SECRET_NAME}
		// gcp:secretmanager:projects/{PROJECT_ID}/secrets/{SECRET_NAME}/versions/{VERSION|latest}
		secretID = strings.TrimPrefix(secretID, "gcp:secretmanager:")

		// Check if the path has version specified
		secretID, err := handleVersion(secretID)
		if err != nil {
			return nil, fmt.Errorf("failed to handle secret ID version: %v", err)
		}

		secret, err := p.client.AccessSecretVersion(
			ctx,
			&secretmanagerpb.AccessSecretVersionRequest{
				Name: secretID,
			})
		if err != nil {
			return nil, fmt.Errorf("failed to access secret version from Google Cloud secret manager: %v", err)
		}

		secrets = append(secrets, provider.Secret{
			Key:   originalKey,
			Value: string(secret.Payload.GetData()),
		})
	}

	return secrets, nil
}

// Example GCP prefixes:
// gcp:secretmanager:projects/{PROJECT_ID}/secrets/{SECRET_NAME}
// gcp:secretmanager:projects/{PROJECT_ID}/secrets/{SECRET_NAME}/versions/{VERSION|latest}
func Valid(envValue string) bool {
	return strings.HasPrefix(envValue, referenceSelector)
}

func handleVersion(secretID string) (string, error) {
	// If the version is correctly specified, return the secretID as is
	match, err := regexp.MatchString(versionRegex, secretID)
	if err != nil {
		return "", fmt.Errorf("failed to match secret ID with regex: %v", err)
	}
	if match {
		return secretID, nil
	}

	// If the version is not specified correctly, handle it
	count := strings.Count(secretID, "/")
	switch {
	// If version is not specified at all (default to latest)
	case count == 3:
		secretID = fmt.Sprintf("%s/versions/latest", secretID)
		return secretID, nil

	// Something wrong is specified after the secret name (default to latest)
	case count >= 4:
		// Delete the wrongly formatted substring
		parts := strings.Split(secretID, "/")
		if len(parts) > 4 {
			secretID = strings.Join(parts[:4], "/")
		}

		secretID = fmt.Sprintf("%s/versions/latest", secretID)
		return secretID, nil

	default:
		return "", fmt.Errorf("invalid secret ID format: %s", secretID)
	}
}
