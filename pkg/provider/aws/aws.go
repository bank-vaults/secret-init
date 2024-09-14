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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/ssm"

	"github.com/bank-vaults/secret-init/pkg/common"
	"github.com/bank-vaults/secret-init/pkg/provider"
)

const (
	ProviderType         = "aws"
	referenceSelectorSM  = "arn:aws:secretsmanager:"
	referenceSelectorSSM = "arn:aws:ssm:"
)

type Provider struct {
	sm  *secretsmanager.SecretsManager
	ssm *ssm.SSM
}

func NewProvider(_ context.Context, _ *common.Config) (provider.Provider, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create vault config: %w", err)
	}

	return &Provider{
		sm:  secretsmanager.New(config.session),
		ssm: ssm.New(config.session),
	}, nil
}

func (p *Provider) LoadSecrets(ctx context.Context, paths []string) ([]provider.Secret, error) {
	var secrets []provider.Secret

	for _, path := range paths {
		split := strings.SplitN(path, "=", 2)
		originalKey, secretID := split[0], split[1]

		// valid secretsmanager secret examples:
		// arn:aws:secretsmanager:region:account-id:secret:secret-name
		// secretsmanager:secret-name
		if strings.Contains(secretID, "secretsmanager:") {
			secret, err := p.sm.GetSecretValueWithContext(
				ctx,
				&secretsmanager.GetSecretValueInput{
					SecretId: aws.String(secretID),
				})
			if err != nil {
				return nil, fmt.Errorf("failed to get secret from AWS secrets manager: %w", err)
			}

			secretBytes, err := extractSecretValueFromSM(secret)
			if err != nil {
				return nil, fmt.Errorf("failed to extract secret value from AWS secrets manager: %w", err)
			}

			secretValue, err := parseSecretValueFromSM(secretBytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse secret value from AWS secrets manager: %w", err)
			}

			secrets = append(secrets, provider.Secret{
				Key:   originalKey,
				Value: string(secretValue),
			})
		}

		// Valid ssm parameter examples:
		// arn:aws:ssm:region:account-id:parameter/path/to/parameter-name
		// arn:aws:ssm:us-west-2:123456789012:parameter/my-parameter
		if strings.Contains(secretID, "ssm:") {
			parameteredSecret, err := p.ssm.GetParameterWithContext(
				ctx,
				&ssm.GetParameterInput{
					Name:           aws.String(secretID),
					WithDecryption: aws.Bool(true),
				})
			if err != nil {
				return nil, fmt.Errorf("failed to get secret from AWS SSM: %w", err)
			}

			secrets = append(secrets, provider.Secret{
				Key:   originalKey,
				Value: aws.StringValue(parameteredSecret.Parameter.Value),
			})
		}
	}

	return secrets, nil
}

// Example AWS prefixes:
// arn:aws:secretsmanager:us-west-2:123456789012:secret:my-secret
// arn:aws:ssm:us-west-2:123456789012:parameter/my-parameter
func Valid(envValue string) bool {
	return strings.HasPrefix(envValue, referenceSelectorSM) || strings.HasPrefix(envValue, referenceSelectorSSM)
}

// AWS Secrets Manager can store secrets in two formats:
// - SecretString: for text-based secrets, returned as a byte slice.
// - SecretBinary: for binary secrets, returned as a byte slice without additional encoding.
// If neither is available, the function returns an error.
//
// Ref: https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_GetSecretValue.html
func extractSecretValueFromSM(secret *secretsmanager.GetSecretValueOutput) ([]byte, error) {
	// Secret available as string
	if secret.SecretString != nil {
		return []byte(aws.StringValue(secret.SecretString)), nil
	}

	// Secret available as binary
	if secret.SecretBinary != nil {
		return secret.SecretBinary, nil
	}

	// Handle the case where neither SecretString nor SecretBinary is available
	return []byte{}, fmt.Errorf("secret does not contain a value in expected formats")
}

// parseSecretValueFromSM takes a secret and attempts to parse it.
// It unifies the handling of all secrets coming from AWS SM,
// ensuring the output is consistent in the form of a []byte slice.
func parseSecretValueFromSM(secretBytes []byte) ([]byte, error) {
	// If the secret is not a JSON object, append it as a single secret
	if !json.Valid(secretBytes) {
		return secretBytes, nil
	}

	var secretValue map[string]interface{}
	err := json.Unmarshal(secretBytes, &secretValue)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret from AWS Secrets Manager: %w", err)
	}

	// If the JSON object contains a single key-value pair, the value is the actual secret
	if len(secretValue) == 1 {
		for _, value := range secretValue {
			valueBytes, err := json.Marshal(value)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal secret from map: %w", err)
			}

			return valueBytes, nil
		}
	}

	// For JSON objects with multiple key-value pairs, the original JSON is returned as is
	return secretBytes, nil
}
