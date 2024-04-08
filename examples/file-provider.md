# File-provider

## Prerequisites

- Golang `>= 1.21`
- Makefile

## Environment setup

```bash
# Create a folder for the example assets
mkdir -p example
```

### Prepare File provider

```bash
# Create secrets for the file provider
printf "secret-value" >> "example/secret-file"
printf "super-secret-value" >> "example/super-secret-value"

#NOTE: Optionally you can set a mount path for the file provider by using the FILE_MOUNT_PATH environment variable.
```

## Define secrets to inject

```bash
# Export environment variables
export FILE_SECRET_1=file:$PWD/example/secret-file
export FILE_SECRET_2=file:$PWD/example/super-secret-value
```

## Run secret-init

```bash
# Build the secret-init binary
go build

# Run secret-init with a command e.g.
./secret-init env | grep 'FILE_SECRET_1\|FILE_SECRET_2'
```

## Cleanup

```bash
# Remove files and binary
rm -rd example/
rm -rf secret-init

# Unset the environment variables
unset FILE_SECRET_1
unset FILE_SECRET_2
```
