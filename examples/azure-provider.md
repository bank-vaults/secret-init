# Azure provider

## Overview

The Azure Provider in Secret-init can load secrets from Azure Key Vault. This provider interfaces with Azure's API, to fetch and load secrets.

## Prerequisites

- Golang `>= 1.21`
- Makefile
- Access to Azure services

## Environment setup

```bash
# Secret-init requires atleast this environment variable to be set properly
export AZURE_KEY_VAULT_URL
```

The options provided in the [Azure SDK Authentication Guide](https://learn.microsoft.com/en-us/azure/developer/go/azure-sdk-authentication?tabs=bash#2-authenticate-with-azure) are all viable when using `secret-init`.
This includes:

- Authenticating with Azure using environment variables.
- A service principal with a client secret.
- A service principal with a certificate.
- A managed identities for Azure resources.

## Define secrets to inject

```bash
# Export environment variables
export AZURE_SECRET=azure:keyvault:secret-init-test
export AZURE_SECRET_WITH_VERSION=azure:keyvault:secret-init-test/1234567f0c4848958aeee4e3e8eabb9e
# NOTE: If version is not supplied then latest will be used.

# NOTE: Secret-init is designed to identify any secret-reference that starts with "azure:keyvault"
```

## Run secret-init

```bash
# Build the secret-init binary
make build

# Run secret-init with a command e.g.
./secret-init env | grep 'AZURE_SECRET\|AZURE_SECRET_WITH_VERSION'
```

## Cleanup

```bash
# Remove binary
rm -rf secret-init

# Unset the environment variables
unset AZURE_SECRET
unset AZURE_SECRET_WITH_VERSION
```
