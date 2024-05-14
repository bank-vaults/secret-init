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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/ssm"

	"github.com/bank-vaults/secret-init/pkg/provider"
)

var ProviderName = "aws"

type Provider struct {
	sm  *secretsmanager.SecretsManager
	ssm *ssm.SSM
}

func NewProvider(config *Config) *Provider {
	return &Provider{
		sm:  secretsmanager.New(config.session),
		ssm: ssm.New(config.session),
	}
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

			switch value := secretValue.(type) {
			case string:
				secrets = append(secrets, provider.Secret{
					Key:   originalKey,
					Value: value,
				})

			case []byte:
				secrets = append(secrets, provider.Secret{
					Key:   originalKey,
					Value: string(value),
				})
			}
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

// https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_GetSecretValue.html
func extractSecretValueFromSM(secret *secretsmanager.GetSecretValueOutput) ([]byte, error) {
	// Secret available as string
	if secret.SecretString != nil {
		return []byte(aws.StringValue(secret.SecretString)), nil
	}

	// Secret available as binary
	decodedSecret, err := base64.StdEncoding.DecodeString(string(secret.SecretBinary))
	if err != nil {
		return nil, fmt.Errorf("failed to decode secret: %w", err)
	}

	return decodedSecret, nil
}

// Parse the secret value from AWS Secrets Manager
func parseSecretValueFromSM(secretBytes []byte) (interface{}, error) {
	// If the secret is not a JSON object, append it as a single secret
	if !json.Valid(secretBytes) {
		return string(secretBytes), nil
	}

	var secretValue map[string]interface{}
	err := json.Unmarshal(secretBytes, &secretValue)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret from AWS Secrets Manager: %w", err)
	}

	// If the map contains a single KV, the actual secret is the value
	if len(secretValue) == 1 {
		for _, value := range secretValue {
			value, err := json.Marshal(value)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal secret from map: %w", err)
			}
			return value, nil
		}
	}

	// If the secret is a JSON object, append it as a single secret
	JSONValue, err := json.Marshal(secretValue)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secret from map: %w", err)
	}

	return JSONValue, nil
}
