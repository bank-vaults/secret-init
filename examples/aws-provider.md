# AWS-provider

## Overview

The AWS Provider in Secret-init can load secrets from AWS Secrets Manager and AWS Systems Manager (SSM) Parameter Store as well.

## Prerequisites

- Golang `>= 1.21`
- Makefile
- Access to AWS services

## Environment setup

```bash
# Secret-ini requires atleast these environment variables to be set properly
export AWS_ACCESS_KEY_ID
export AWS_SECRET_ACCESS_KEY
export AWS_REGION
```

## Define secrets to inject

```bash
# Export environment variables
export MYSQL_PASSWORD=arn:aws:secretsmanager:eu-north-1:123456789:secret:secret/test/mysql-ASD123
export SM_JSON=arn:aws:secretsmanager:eu-north-1:123456789:secret:test/secret/JSON-ASD123
export SSM_SECRET=arn:aws:ssm:eu-north-1:123456789:parameter/bank-vaults/test

# NOTE: Secret-init is designed to identify any secret-reference that starts with "arn:aws:secretsmanager:" or "arn:aws:ssm:"
```

## Run secret-init

```bash
# Build the secret-init binary
go build

# Use in daemon mode
SECRET_INIT_DAEMON="true"

# Run secret-init with a command e.g.
./secret-init env | grep 'MYSQL_PASSWORD\|SM_JSON\|SSM_SECRET'

# JSON secrets are loaded as is:
# SM_JSON="{"firsts3cr3t":"s3cr3ton3","seconds3cr3t":"s3cr3ttwo"}"
```

## Cleanup

```bash
# Remove binary
rm -rf secret-init

# Unset the environment variables
unset MYSQL_PASSWORD
unset SM_JSON
unset SSM_SECRET
```
