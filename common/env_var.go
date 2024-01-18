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

package common

const (
	// main
	SecretInitLogLevel  = "SECRET_INIT_LOG_LEVEL"
	SecretInitJSONLog   = "SECRET_INIT_JSON_LOG"
	SecretInitLogServer = "SECRET_INIT_LOG_SERVER"
	SecretInitDaemon    = "SECRET_INIT_DAEMON"
	SecretInitDelay     = "SECRET_INIT_DELAY"
	Provider            = "PROVIDER"

	// file provider
	FileMountPath = "FILE_MOUNT_PATH"

	// vault provider
	VaultToken                = "VAULT_TOKEN"
	VaultTokenFile            = "VAULT_TOKEN_FILE"
	VaultAddr                 = "VAULT_ADDR"
	VaultAgentAddr            = "VAULT_AGENT_ADDR"
	VaultCACert               = "VAULT_CACERT"
	VaultCAPath               = "VAULT_CAPATH"
	VaultClientCert           = "VAULT_CLIENT_CERT"
	VaultClientKey            = "VAULT_CLIENT_KEY"
	VaultClientTimeout        = "VAULT_CLIENT_TIMEOUT"
	VaultSRVLookup            = "VAULT_SRV_LOOKUP"
	VaultSkipVerify           = "VAULT_SKIP_VERIFY"
	VaultNamespace            = "VAULT_NAMESPACE"
	VaultTLSServerName        = "VAULT_TLS_SERVER_NAME"
	VaultWrapTTL              = "VAULT_WRAP_TTL"
	VaultMFA                  = "VAULT_MFA"
	VaultMaxRetries           = "VAULT_MAX_RETRIES"
	VaultClusterAddr          = "VAULT_CLUSTER_ADDR"
	VaultRedirectAddr         = "VAULT_REDIRECT_ADDR"
	VaultCLINoColor           = "VAULT_CLI_NO_COLOR"
	VaultRateLimit            = "VAULT_RATE_LIMIT"
	VaultRole                 = "VAULT_ROLE"
	VaultPath                 = "VAULT_PATH"
	VaultAuthMethod           = "VAULT_AUTH_METHOD"
	VaultTransitKeyID         = "VAULT_TRANSIT_KEY_ID"
	VaultTransitPath          = "VAULT_TRANSIT_PATH"
	VaultTransitBatchSize     = "VAULT_TRANSIT_BATCH_SIZE"
	VaultIgnoreMissingSecrets = "VAULT_IGNORE_MISSING_SECRETS"
	VaultPassthrough          = "VAULT_PASSTHROUGH"
	VaultRevokeToken          = "VAULT_REVOKE_TOKEN"
	VaultFromPath             = "VAULT_FROM_PATH"
)
