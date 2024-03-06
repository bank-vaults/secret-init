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

const ProviderName = "aws"

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
		// valid secretsmanager secret examples:
		// arn:aws:secretsmanager:region:account-id:secret:secret-name
		// secretsmanager:secret-name
		if strings.Contains(path, "secretsmanager:") {
			secret, err := p.sm.GetSecretValue(&secretsmanager.GetSecretValueInput{SecretId: &path})
			if err != nil {
				return nil, errors.Wrap(err, "failed to get secret from AWS secrets manager")
			}

			if json.Valid([]byte(*secret.SecretString)) {
				var secretValues map[string]interface{}
				err := json.Unmarshal([]byte(*secret.SecretString), &secretValues)
				if err != nil {
					return nil, errors.Wrap(err, "failed to unmarshal secret value")
				}

				for key, value := range secretValues {
					secretToAppend, err := appendSecret(key, value)
					if err != nil {
						return nil, errors.Wrap(err, "failed to append secret")
					}
					secrets = append(secrets, secretToAppend)
				}

				// Only add the secrets present in the JSON
				continue
			}

			secretToAppend, err := appendSecret(path, *secret.SecretString)
			if err != nil {
				return nil, errors.Wrap(err, "failed to append secret")
			}
			secrets = append(secrets, secretToAppend)
		}

		// Valid ssm parameter examples:
		// arn:aws:ssm:region:account-id:parameter/path/to/parameter-name
		// arn:aws:ssm:us-west-2:123456789012:parameter/my-parameter
		if strings.Contains(path, "ssm:") {
			var parameteredSecretPath string
			tokens := strings.Split(path, ":")
			switch len(tokens) {
			case 6:
				parameteredSecretPath = strings.Join(tokens[5:], ":")
			case 7:
				parameteredSecretPath = strings.Join(tokens[6:], ":")
			default:
				return nil, errors.New("invalid SSM parameter path")
			}

			parameteredSecret, err := p.ssm.GetParameter(&ssm.GetParameterInput{
				Name:           aws.String(parameteredSecretPath),
				WithDecryption: aws.Bool(true),
			})
			if err != nil {
				return nil, errors.Wrap(err, "failed to get parameter from AWS SSM")
			}

			secretToAppend, err := appendSecret(parameteredSecretPath, *parameteredSecret.Parameter.Value)
			if err != nil {
				return nil, errors.Wrap(err, "failed to append secret")
			}
			secrets = append(secrets, secretToAppend)
		}
	}

	return secrets, nil
}

func appendSecret(path string, value interface{}) (provider.Secret, error) {
	switch v := value.(type) {
	case string:
		return provider.Secret{
			Path:  path,
			Value: v,
		}, nil
	case []byte:
		return provider.Secret{
			Path:  path,
			Value: string(v),
		}, nil
	default:
		return provider.Secret{}, errors.New("unsupported secret value type")
	}
}
