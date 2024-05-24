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

package azure

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"

	"github.com/bank-vaults/secret-init/pkg/provider"
)

var ProviderName = "azure"

type Provider struct {
	client *azsecrets.Client
}

func NewProvider(config *Config) (*Provider, error) {
	creds, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create default azure credentials: %v", err)
	}

	client, err := azsecrets.NewClient(config.keyvaultURL, creds, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new keyvault client: %v", err)
	}

	return &Provider{
		client: client,
	}, nil
}

func (p *Provider) LoadSecrets(ctx context.Context, paths []string) ([]provider.Secret, error) {
	var secrets []provider.Secret

	for _, path := range paths {
		split := strings.SplitN(path, "=", 2)
		originalKey, secretID := split[0], split[1]

		// valid Azure Key Vault secret examples:
		// azure:keyvault:{SECRET_NAME}
		// azure:keyvault:{SECRET_NAME}/{VERSION}
		version := ""
		secretID = strings.TrimPrefix(secretID, "azure:keyvault:")
		split = strings.Split(secretID, "/")
		secretID = split[0]
		if len(split) == 2 {
			version = split[1]
		}

		secret, err := p.client.GetSecret(ctx, secretID, version, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret %s: %v", path, err)
		}

		secrets = append(secrets, provider.Secret{
			Key:   originalKey,
			Value: *secret.Value,
		})
	}

	return secrets, nil
}
