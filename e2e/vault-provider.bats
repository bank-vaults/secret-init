vault_container_name="vault"

setup() {
  bats_load_library bats-support
  bats_load_library bats-assert

  run go build
  assert_success
}

setup_vault_provider() {
  TMPFILE_TOKEN=$(mktemp)
  printf "227e1cce-6bf7-30bb-2d2a-acc854318caf" > "$TMPFILE_TOKEN"

  export VAULT_ADDR="http://127.0.0.1:8200"
  export VAULT_TOKEN_FILE="$TMPFILE_TOKEN"

  export MYSQL_PASSWORD="vault:secret/data/test/mysql#MYSQL_PASSWORD"
  export AWS_SECRET_ACCESS_KEY="vault:secret/data/test/aws#AWS_SECRET_ACCESS_KEY"
  export AWS_ACCESS_KEY_ID="vault:secret/data/test/aws#AWS_ACCESS_KEY_ID"

  start_vault
}

start_vault() {
  docker compose up -d

  # wait for Vault to be ready
  max_attempts=${MAX_ATTEMPTS:-10}

  for ((attempts = 0; attempts < max_attempts; attempts++)); do
    if docker compose exec -T "$vault_container_name"  vault status > /dev/null 2>&1; then
      break
    fi
    sleep 1
  done
}

set_vault_token() {
  local token=$1
  export VAULT_TOKEN="$token"
}

set_daemon_mode() {
  export SECRET_INIT_DAEMON="true"
}

add_secrets_to_vault() {
  docker exec "$vault_container_name" vault kv put secret/test/mysql MYSQL_PASSWORD=3xtr3ms3cr3t
  docker exec "$vault_container_name" vault kv put secret/test/aws AWS_ACCESS_KEY_ID=secretId AWS_SECRET_ACCESS_KEY=s3cr3t
}

add_custom_secret_to_vault() {
  local path="$1"
  shift
  local data=()

  for secret in "$@"; do
    data+=("$secret")
  done

  vault kv put "$path" "${data[@]}"
}

teardown() {
  docker compose down

  rm -f "$TMPFILE_TOKEN"
  rm -f secret-init
}

assert_output_contains() {
  local expected=$1
  local output=$2

  echo "$output" | grep -qF "$expected" || fail "Expected line not found: $expected"
}

check_process_status() {
  local process_name="$1"

  if pgrep -f "$process_name" > /dev/null; then
    echo "Process is running"
  else
    echo "Process is not running"
  fi
}


@test "secrets successfully loaded from vault" {
  setup_vault_provider
  set_vault_token 227e1cce-6bf7-30bb-2d2a-acc854318caf
  add_secrets_to_vault

  run_output=$(./secret-init env | grep 'MYSQL_PASSWORD\|AWS_SECRET_ACCESS_KEY\|AWS_ACCESS_KEY_ID')
  assert_success

  assert_output_contains "MYSQL_PASSWORD=3xtr3ms3cr3t" "$run_output"
  assert_output_contains "AWS_SECRET_ACCESS_KEY=s3cr3t" "$run_output"
  assert_output_contains "AWS_ACCESS_KEY_ID=secretId" "$run_output"
}

@test "secrets successfully loaded from vault using vault:login as token" {
  setup_vault_provider
  set_vault_token "vault:login"
  add_secrets_to_vault

  run_output=$(./secret-init env | grep 'MYSQL_PASSWORD\|AWS_SECRET_ACCESS_KEY\|AWS_ACCESS_KEY_ID')
  assert_success

  assert_output_contains "MYSQL_PASSWORD=3xtr3ms3cr3t" "$run_output"
  assert_output_contains "AWS_SECRET_ACCESS_KEY=s3cr3t" "$run_output"
  assert_output_contains "AWS_ACCESS_KEY_ID=secretId" "$run_output"
}

@test "secrets successfully loaded from vault using vault:login as token and daemon mode enabled" {
  setup_vault_provider
  set_vault_token "vault:login"
  set_daemon_mode
  add_secrets_to_vault

  run_output=$(./secret-init env | grep 'MYSQL_PASSWORD\|AWS_SECRET_ACCESS_KEY\|AWS_ACCESS_KEY_ID')
  assert_success

  assert_output_contains "MYSQL_PASSWORD=3xtr3ms3cr3t" "$run_output"
  assert_output_contains "AWS_SECRET_ACCESS_KEY=s3cr3t" "$run_output"
  assert_output_contains "AWS_ACCESS_KEY_ID=secretId" "$run_output"

  # Check if the process is still running in the background
  check_process_status "secret-init env"
  assert_success
}

@test "secrets successfully loaded from vault using VAULT_FROM_PATH" {
  # unset env vars to ensure secret-init will utilize VAULT_FROM_PATH
  unset MYSQL_PASSWORD
  unset AWS_SECRET_ACCESS_KEY
  unset AWS_ACCESS_KEY_ID

  setup_vault_provider
  set_vault_token 227e1cce-6bf7-30bb-2d2a-acc854318caf
  add_secrets_to_vault
  export VAULT_FROM_PATH="secret/data/test/mysql,secret/data/test/aws"

  run_output=$(./secret-init env | grep 'MYSQL_PASSWORD\|AWS_SECRET_ACCESS_KEY\|AWS_ACCESS_KEY_ID')
  assert_success

  assert_output_contains "MYSQL_PASSWORD=3xtr3ms3cr3t" "$run_output"
  assert_output_contains "AWS_SECRET_ACCESS_KEY=s3cr3t" "$run_output"
  assert_output_contains "AWS_ACCESS_KEY_ID=secretId" "$run_output"
}

@test "secrets sucessfully loaded from vault using different injection cases" {
  setup_vault_provider
  set_vault_token 227e1cce-6bf7-30bb-2d2a-acc854318caf
  add_secrets_to_vault

  # Secret with version
  add_custom_secret_to_vault "secret/test/mysql" "MYSQL_PASSWORD=modify3d3xtr3ms3cr3t"
  export MYSQL_PASSWORD="vault:secret/data/test/mysql#MYSQL_PASSWORD#2"

  # Inline secrets with scheme
  add_custom_secret_to_vault "secret/test/scheme" "SCHEME_SECRET1=sch3m3s3cr3tONE" "SCHEME_SECRET2=sch3m3s3cr3tTWO"
  export SCHEME_SECRET="scheme://\${vault:secret/data/test/scheme#SCHEME_SECRET1}:\${vault:secret/data/test/scheme#SCHEME_SECRET2}@$VAULT_ADDR"

  # Enable pki secrets engine and generate root certificates
  vault secrets enable -path=pki pki
  export ROOT_CERT=">>vault:pki/root/generate/internal#certificate"
  export ROOT_CERT_CACHED=">>vault:pki/root/generate/internal#certificate"

  run_output=$(./secret-init env | grep 'MYSQL_PASSWORD\|SCHEME_SECRET\|ROOT_CERT\|ROOT_CERT_CACHED')
  assert_success

  assert_output_contains "MYSQL_PASSWORD=modify3d3xtr3ms3cr3t" "$run_output"
  assert_output_contains "SCHEME_SECRET=scheme://sch3m3s3cr3tONE:sch3m3s3cr3tTWO@$VAULT_ADDR" "$run_output"

  [ $ROOT_CERT == $ROOT_CERT_CACHED ]
  assert_success "ROOT_CERT and ROOT_CERT_CACHED are not the same"
}
