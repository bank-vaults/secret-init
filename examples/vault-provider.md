# Vault-provider

## Prerequisites

- Golang `>= 1.21`
- Makefile
- Docker compose

## Environment setup

```bash
# Deploy a Vault and a Bao instance
make up
```

```bash
# Create a folder for the example assets
mkdir -p example
```

### Prepare Vault provider

```bash
export VAULT_ADDR=http://127.0.0.1:8200
# Create a tokenfile
export VAULT_TOKEN=227e1cce-6bf7-30bb-2d2a-acc854318caf
printf $VAULT_TOKEN > "example/vault-token-file"
export VAULT_TOKEN_FILE=$PWD/example/vault-token-file

#NOTE: Secret-init can authenticate to Vault by supplying role/path credentials.

# Create secrets for the vault provider
docker exec secret-init-vault vault kv put secret/test/mysql MYSQL_PASSWORD=3xtr3ms3cr3t
docker exec secret-init-vault vault kv put secret/test/aws AWS_ACCESS_KEY_ID=secretId AWS_SECRET_ACCESS_KEY=s3cr3t
```

## Define secrets to inject

```bash
# Export environment variables
export MYSQL_PASSWORD=vault:secret/data/test/mysql#MYSQL_PASSWORD
export AWS_SECRET_ACCESS_KEY=vault:secret/data/test/aws#AWS_SECRET_ACCESS_KEY
export AWS_ACCESS_KEY_ID=vault:secret/data/test/aws#AWS_ACCESS_KEY_ID
```

## Run secret-init

```bash
# Build the secret-init binary
go build

# Use in daemon mode
export SECRET_INIT_DAEMON="true"

# Run secret-init with a command e.g.
./secret-init env | grep 'MYSQL_PASSWORD\|AWS_SECRET_ACCESS_KEY\|AWS_ACCESS_KEY_ID'
```

## Cleanup

```bash
# Remove files and binary
rm -rd example/
rm -rf secret-init

# Remove the Vault instance
make down

# Unset the environment variables
unset VAULT_ADDR
unset VAULT_TOKEN
unset VAULT_TOKEN_FILE
unset SECRET_INIT_DAEMON
unset MYSQL_PASSWORD
unset AWS_SECRET_ACCESS_KEY
unset AWS_ACCESS_KEY_ID
```
