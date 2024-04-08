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
	"strings"

	"emperror.dev/errors"
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

func (p *Provider) LoadSecrets(_ context.Context, paths []string) ([]provider.Secret, error) {
	var secrets []provider.Secret

	for _, path := range paths {
		split := strings.SplitN(path, "=", 2)
		originalKey, secretID := split[0], split[1]

		// valid secretsmanager secret examples:
		// arn:aws:secretsmanager:region:account-id:secret:secret-name
		// secretsmanager:secret-name
		if strings.Contains(secretID, "secretsmanager:") {
			secret, err := p.sm.GetSecretValue(
				&secretsmanager.GetSecretValueInput{
					SecretId: aws.String(secretID),
				})
			if err != nil {
				return nil, errors.Wrap(err, "failed to get secret from AWS secrets manager")
			}

			if json.Valid([]byte(*secret.SecretString)) {
				var secretValue map[string]interface{}
				err := json.Unmarshal([]byte(*secret.SecretString), &secretValue)
				if err != nil {
					return nil, errors.Wrap(err, "failed to unmarshal secret value")
				}

				// If there is only one key in the secret, append it directly
				if len(secretValue) == 1 {
					for _, value := range secretValue {
						secretToAppend, err := appendSecret(originalKey, value)
						if err != nil {
							return nil, errors.Wrap(err, "failed to append secret")
						}

						secrets = append(secrets, secretToAppend)
					}

					continue
				}

				// If the secret is a JSON object, append it as a single secret
				JSONValue, err := json.Marshal(secretValue)
				if err != nil {
					return nil, errors.Wrap(err, "failed to marshal secret value")
				}

				secretToAppend, err := appendSecret(originalKey, JSONValue)
				if err != nil {
					return nil, errors.Wrap(err, "failed to append secret")
				}

				secrets = append(secrets, secretToAppend)

			} else {
				// If the secret is not a JSON object, append it directly
				secretToAppend, err := appendSecret(originalKey, aws.StringValue(secret.SecretString))
				if err != nil {
					return nil, errors.Wrap(err, "failed to append secret")
				}

				secrets = append(secrets, secretToAppend)
			}
		}

		// Valid ssm parameter examples:
		// arn:aws:ssm:region:account-id:parameter/path/to/parameter-name
		// arn:aws:ssm:us-west-2:123456789012:parameter/my-parameter
		if strings.Contains(secretID, "ssm:") {
			parameteredSecret, err := p.ssm.GetParameter(
				&ssm.GetParameterInput{
					Name:           aws.String(secretID),
					WithDecryption: aws.Bool(true),
				})
			if err != nil {
				return nil, errors.Wrap(err, "failed to get parameter from AWS SSM")
			}

			secretToAppend, err := appendSecret(originalKey, *parameteredSecret.Parameter.Value)
			if err != nil {
				return nil, errors.Wrap(err, "failed to append secret")
			}

			secrets = append(secrets, secretToAppend)
		}
	}

	return secrets, nil
}

func appendSecret(key string, value interface{}) (provider.Secret, error) {
	switch v := value.(type) {
	case string:
		return provider.Secret{
			Key:   key,
			Value: v,
		}, nil

	case []byte:
		return provider.Secret{
			Key:   key,
			Value: string(v),
		}, nil

	default:
		return provider.Secret{}, errors.New("unsupported secret value type")
	}
}
