# Bao provider

## Overview

The Bao provider in Secret-Init can load secrets from Open-Bao. This provider interfaces with Vault's API via Bank-Vaults's Vault-SDK, to fetch and load secrets.

## Prerequisites

- Golang `>= 1.21`
- Makefile
- Docker compose

## Environment setup

```bash
# Deploy a Bao instance
make up
```

```bash
# Create a folder for the example assets
mkdir -p example
```

### Prepare Bao provider

```bash
export BAO_ADDR=http://127.0.0.1:8300
# Create a tokenfile
export BAO_TOKEN=227e1cce-6bf7-30bb-2d2a-acc854318caf
printf $BAO_TOKEN > "example/bao-token-file"
export BAO_TOKEN_FILE=$PWD/example/bao-token-file

#NOTE: Secret-init can authenticate to Bao by supplying role/path credentials.

# Create secrets for the bao provider
docker exec secret-init-bao bao kv put secret/test/api API_KEY=sensitiveApiKey
docker exec secret-init-bao bao kv put secret/test/rabbitmq RABBITMQ_USERNAME=rabbitmqUser RABBITMQ_PASSWORD=rabbitmqPassword
```

## Define secrets to inject

```bash
# Export environment variables
export API_KEY="bao:secret/data/test/api#API_KEY"
export RABBITMQ_USERNAME="bao:secret/data/test/rabbitmq#RABBITMQ_USERNAME"
export RABBITMQ_PASSWORD="bao:secret/data/test/rabbitmq#RABBITMQ_PASSWORD"
```

## Run secret-init

```bash
# Build the secret-init binary
make build

# Use in daemon mode
export SECRET_INIT_DAEMON="true"

# Run secret-init with a command e.g.
./secret-init env | grep 'API_KEY\|RABBITMQ_USERNAME\|RABBITMQ_PASSWORD'
```

## Cleanup

```bash
# Remove files and binary
rm -rd example/
rm -rf secret-init

# Remove the Vault instance
make down

# Unset the environment variables
unset BAO_ADDR
unset BAO_TOKEN
unset BAO_TOKEN_FILE
unset SECRET_INIT_DAEMON
unset API_KEY
unset RABBITMQ_USERNAME
unset RABBITMQ_PASSWORD
```
