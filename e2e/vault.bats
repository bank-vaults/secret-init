vault_container_name="vault"

setup() {
  bats_load_library bats-support
  bats_load_library bats-assert

  start_vault

  setup_pod

  run go build
  assert_success
}

start_vault() {
  docker-compose up -d

  # wait for Vault to be ready
  max_attempts=${MAX_ATTEMPTS:-10}

  for ((attempts = 0; attempts < max_attempts; attempts++)); do
    if docker-compose exec -T "$vault_container_name"  vault status > /dev/null 2>&1; then
      break
    fi
    sleep 1
  done
}

setup_pod() {
  TMPFILE=$(mktemp)
  printf "227e1cce-6bf7-30bb-2d2a-acc854318caf" > "$TMPFILE"

  export SECRET_INIT_PROVIDER="vault"
  export VAULT_ADDR="http://127.0.0.1:8200"
  export VAULT_TOKEN=227e1cce-6bf7-30bb-2d2a-acc854318caf
  export VAULT_TOKEN_FILE="$TMPFILE"

  export MYSQL_PASSWORD=vault:secret/data/test/mysql#MYSQL_PASSWORD
  export AWS_SECRET_ACCESS_KEY=vault:secret/data/test/aws#AWS_SECRET_ACCESS_KEY
  export AWS_ACCESS_KEY_ID=vault:secret/data/test/aws#AWS_ACCESS_KEY_ID
}

add_secrets_to_vault() {
  docker exec "$vault_container_name" vault kv put secret/test/mysql MYSQL_PASSWORD=3xtr3ms3cr3t
  docker exec "$vault_container_name" vault kv put secret/test/aws AWS_ACCESS_KEY_ID=secretId AWS_SECRET_ACCESS_KEY=s3cr3t
}

teardown() {
  stop_vault

  rm -f "$TMPFILE"
  rm -f secret-init
}

stop_vault() {
  remove_secrets_from_vault
  docker-compose down
}

remove_secrets_from_vault() {
  docker exec "$vault_container_name" vault kv delete secret/test/mysql
  docker exec "$vault_container_name" vault kv delete secret/test/aws
}

assert_output_contains() {
  local expected=$1
  local output=$2
  echo "$output" | grep -qF "$expected" || fail "Expected line not found: $expected"
}

@test "secrets successfully loaded from vault" {
  add_secrets_to_vault

  run_output=$(./secret-init env | grep 'MYSQL_PASSWORD\|AWS_SECRET_ACCESS_KEY\|AWS_ACCESS_KEY_ID')
  assert_success

  assert_output_contains "MYSQL_PASSWORD=3xtr3ms3cr3t" "$run_output"
  assert_output_contains "AWS_SECRET_ACCESS_KEY=s3cr3t" "$run_output"
  assert_output_contains "AWS_ACCESS_KEY_ID=secretId" "$run_output"
}

@test "secrets successfully loaded from vault using VAULT_FROM_PATH" {
  # unset env vars to ensure secret-init will utilize VAULT_FROM_PATH
  unset MYSQL_PASSWORD
  unset AWS_SECRET_ACCESS_KEY
  unset AWS_ACCESS_KEY_ID

  add_secrets_to_vault

  export VAULT_FROM_PATH="secret/data/test/mysql,secret/data/test/aws"

  run_output=$(./secret-init env | grep 'MYSQL_PASSWORD\|AWS_SECRET_ACCESS_KEY\|AWS_ACCESS_KEY_ID')
  assert_success

  assert_output_contains "MYSQL_PASSWORD=3xtr3ms3cr3t" "$run_output"
  assert_output_contains "AWS_SECRET_ACCESS_KEY=s3cr3t" "$run_output"
  assert_output_contains "AWS_ACCESS_KEY_ID=secretId" "$run_output"
}