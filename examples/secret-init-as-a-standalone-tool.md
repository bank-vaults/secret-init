## Secret-init as a standalone tool
**Multi-provider setup**

## Prerequisites

- Golang `>= 1.21`
- Makefile
- Docker compose

## Environment setup

```bash
# Deploy a Vault instance
make up
```

```bash
# Create a folder for the example assets
mkdir -p example
```

- Prepare File provider
```bash
#NOTE: Optionally you can set a mount path for the file provider by using the FILE_MOUNT_PATH environment variable.
```

- Prepare Vault provider
```bash
# Create a tokenfile
printf $VAULT_TOKEN > "example/token-file"
export VAULT_TOKEN_FILE=$PWD/example/token-file

#NOTE: Secret-init can authenticate to Vault by supplying role/path credentials. 
```

## Define secrets to inject
```bash
# Create secrets for the file provider
printf "secret-value" >> "example/secret-file"
printf "super-secret-value" >> "example/super-secret-value"

# Create secrets for the vault provider
vault kv put secret/test/mysql MYSQL_PASSWORD=3xtr3ms3cr3t
vault kv put secret/test/aws AWS_ACCESS_KEY_ID=secretId AWS_SECRET_ACCESS_KEY=s3cr3t
```

```bash
# Export environment variables
export FILE_SECRET_1=file:$PWD/example/secret-file
export FILE_SECRET_2=file:$PWD/example/super-secret-value
export MYSQL_PASSWORD=vault:secret/data/test/mysql#MYSQL_PASSWORD
export AWS_SECRET_ACCESS_KEY=vault:secret/data/test/aws#AWS_SECRET_ACCESS_KEY
export AWS_ACCESS_KEY_ID=vault:secret/data/test/aws#AWS_ACCESS_KEY_ID
```

## Run secret-init

```bash
# Build the secret-init binary
go build

# Use in daemon mode
SECRET_INIT_DAEMON="true"


# Run secret-init with a command e.g.
./secret-init env | grep 'MYSQL_PASSWORD\|AWS_SECRET_ACCESS_KEY\|AWS_ACCESS_KEY_ID\|FILE_SECRET_1\|FILE_SECRET_2'
```

## Cleanup

```bash
# Remove files and binary
rm -rd example/
rm -rf secret-init

# Remove the Vault instance
make down

# Unset the environment variables
unset VAULT_TOKEN_FILE
unset SECRET_INIT_DAEMON
unset FILE_SECRET_1
unset FILE_SECRET_2
unset MYSQL_PASSWORD
unset AWS_SECRET_ACCESS_KEY
unset AWS_ACCESS_KEY_ID
```
