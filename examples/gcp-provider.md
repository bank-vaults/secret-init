# GCP provider

## Overview

The Google Cloud Provider in Secret-Init can load secrets from Google Cloud Secret Manager. This provider interfaces with Google Cloud Secret Manager's API, to fetch and load secrets.

## Prerequisites

- Golang `>= 1.21`
- Makefile
- Access to GCP services

## Environment setup

```bash
# Secret-init requires atleast this environment variable to be set properly
export GOOGLE_APPLICATION_CREDENTIALS
```

## Define secrets to inject

```bash
# Export environment variables
export MYSQL_PASSWORD=gcp:secretmanager:projects/123456789123/secrets/bank-vaults_secret-init_test_mysql_password/versions/2
export UNVERSIONED_SECRET=gcp:secretmanager:projects/123456789123/secrets/bank-vaults_secret-init_test
# NOTE: If version is not supplied then latest will be used.

# NOTE: Secret-init is designed to identify any secret-reference that starts with "gcp:secretmanager"
```

## Run secret-init

```bash
# Build the secret-init binary
make build

# Run secret-init with a command e.g.
./secret-init env | grep 'MYSQL_PASSWORD\|UNVERSIONED_SECRET'
```

## Cleanup

```bash
# Remove binary
rm -rf secret-init

# Unset the environment variables
unset MYSQL_PASSWORD
unset UNVERSIONED_SECRET
```
