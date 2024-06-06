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
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bank-vaults/secret-init/pkg/provider"
	sdk "github.com/bitwarden/sdk-go"
	"github.com/gofrs/uuid"
)

var ProviderName = "bitwarden"

type Provider struct {
	bitwardenClient sdk.BitwardenClientInterface
	organizationID  uuid.UUID
}

func NewProvider(config *Config) (*Provider, error) {
	bitwardenClient, err := sdk.NewBitwardenClient(&config.APIURL, &config.IdentityURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create bitwarden client: %w", err)
	}

	err = bitwardenClient.AccessTokenLogin(config.AccessToken, &config.StatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to login to bitwarden: %w", err)
	}

	return &Provider{
		bitwardenClient: bitwardenClient,
		organizationID:  config.OrganizationID,
	}, nil
}

func (p *Provider) LoadSecrets(_ context.Context, paths []string) ([]provider.Secret, error) {
	defer p.bitwardenClient.Close()

	var secrets []provider.Secret

	for _, path := range paths {
		split := strings.SplitN(path, "=", 2)
		originalKey, secretID := split[0], split[1]

		// if the secretID is the organizationID, retrieve all secrets
		// To retrieve all secrets in an organization:
		// bw:{ORGANIZATION_ID}
		// NOTE: (only works if BITWARDEN_ORGANIZATION_ID is also set to the same value)
		if secretID == p.organizationID.String() {
			jsonSecrets, err := p.getSecretsByIDs()
			if err != nil {
				return nil, fmt.Errorf("failed to get secrets by IDs: %w", err)
			}

			secrets = append(secrets, provider.Secret{
				Key:   originalKey,
				Value: string(jsonSecrets),
			})

			continue
		}

		// Example Bitwarden secret examples:
		// bw:{SECRET_ID}
		secret, err := p.bitwardenClient.Secrets().Get(secretID)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret %s: %w", secretID, err)
		}

		secrets = append(secrets, provider.Secret{
			Key:   secret.Key,
			Value: secret.Value,
		})
	}

	return secrets, nil
}

func (p *Provider) getSecretsByIDs() ([]byte, error) {
	secretIdentifiers, err := p.bitwardenClient.Secrets().List(p.organizationID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	// construct a slice of secret IDs
	secretIDs := make([]string, len(secretIdentifiers.Data))
	for _, identifier := range secretIdentifiers.Data {
		secretIDs = append(secretIDs, identifier.ID)
	}

	// get all secrets by IDs
	secretsFromIDs, err := p.bitwardenClient.Secrets().GetByIDS(secretIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get secrets by IDs: %w", err)
	}

	// marshal secrets back to JSON
	jsonSecrets, err := json.MarshalIndent(secretsFromIDs, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secrets: %w", err)
	}

	return jsonSecrets, nil
}
