vault_container_name="secret-init-vault"
bao_container_name="secret-init-bao"

setup() {
  bats_load_library bats-support
  bats_load_library bats-assert

  run go build
  assert_success

  start_containers
}

start_containers() {
  docker compose up -d

  # wait for Bao and Vault to be ready
  max_attempts=${MAX_ATTEMPTS:-10}
  for ((attempts = 0; attempts < max_attempts; attempts++)); do
    if docker compose exec -T "$vault_container_name"  bao status > /dev/null 2>&1 && \
       docker compose exec -T "$bao_container_name" vault status > /dev/null 2>&1; then
      break
    fi
    sleep 1
  done
}

setup_file_provider() {
  add_secret_file

  export FILE_MOUNT_PATH="/"
  export FILE_SECRET="file:$TMPFILE_SECRET"
}

add_secret_file() {
  TMPFILE_SECRET=$(mktemp)
  printf "secret-value" > "$TMPFILE_SECRET"
}

setup_vault_provider() {
  TMPFILE_VAULT_TOKEN=$(mktemp)
  printf "227e1cce-6bf7-30bb-2d2a-acc854318caf" > "$TMPFILE_VAULT_TOKEN"

  export VAULT_ADDR="http://127.0.0.1:8200"
  export VAULT_TOKEN_FILE="$TMPFILE_VAULT_TOKEN"

  export MYSQL_PASSWORD="vault:secret/data/test/mysql#MYSQL_PASSWORD"
  export AWS_SECRET_ACCESS_KEY="vault:secret/data/test/aws#AWS_SECRET_ACCESS_KEY"
  export AWS_ACCESS_KEY_ID="vault:secret/data/test/aws#AWS_ACCESS_KEY_ID"
}

set_vault_token() {
  local token=$1
  export VAULT_TOKEN="$token"
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

  docker exec "$vault_container_name" vault kv put "$path" "${data[@]}"
}

setup_bao_provider() {
  TMPFILE_BAO_TOKEN=$(mktemp)
  printf "227e1cce-6bf7-30bb-2d2a-acc854318caf" > "$TMPFILE_BAO_TOKEN"

  export BAO_ADDR="http://127.0.0.1:8300"
  export BAO_TOKEN_FILE="$TMPFILE_BAO_TOKEN"

  export API_KEY="bao:secret/data/test/api#API_KEY"
  export RABBITMQ_USERNAME="bao:secret/data/test/rabbitmq#RABBITMQ_USERNAME"
  export RABBITMQ_PASSWORD="bao:secret/data/test/rabbitmq#RABBITMQ_PASSWORD"
}

set_bao_token() {
  local token=$1
  export BAO_TOKEN="$token"
}

add_secrets_to_bao() {
  docker exec "$bao_container_name" bao kv put secret/test/api API_KEY=sensitiveApiKey
  docker exec "$bao_container_name" bao kv put secret/test/rabbitmq RABBITMQ_USERNAME=rabbitmqUser RABBITMQ_PASSWORD=rabbitmqPassword
}

add_custom_secret_to_bao() {
  local path="$1"
  shift
  local data=()

  for secret in "$@"; do
    data+=("$secret")
  done

  docker exec "$bao_container_name" bao kv put "$path" "${data[@]}"
}

set_daemon_mode() {
  export SECRET_INIT_DAEMON="true"
}

setup_database_for_daemon_mode() {
  docker network create my-network

  # Start a PostgreSQL container so a renewable secret can be created
  docker run --network=my-network --name my-postgres -e POSTGRES_PASSWORD=mysecretpassword -e POSTGRES_DB=mydb -p 5432:5432 -d postgres

  # wait for Postgre to be ready
  max_attempts=${MAX_ATTEMPTS:-10}
  for ((attempts = 0; attempts < max_attempts; attempts++)); do
    if docker exec my-postgres pg_isready -U postgres -d mydb > /dev/null 2>&1; then
      break
    fi
    sleep 1
  done

  docker network connect my-network "$vault_container_name"
  docker network connect my-network "$bao_container_name"

  # Enable the database secrets engine
  docker exec "$vault_container_name" vault secrets enable database
  docker exec "$bao_container_name" bao secrets enable database

  # Configure the database secrets engine
  docker exec "$vault_container_name" vault write database/config/my-database \
      plugin_name=postgresql-database-plugin \
      allowed_roles="my-role-vault" \
      connection_url="postgresql://postgres:mysecretpassword@my-postgres:5432/mydb?sslmode=disable" \
      username="postgres" \
      password="mysecretpassword"

  docker exec "$bao_container_name" bao write database/config/my-database \
      plugin_name=postgresql-database-plugin \
      allowed_roles="my-role-bao" \
      connection_url="postgresql://postgres:mysecretpassword@my-postgres:5432/mydb?sslmode=disable" \
      username="postgres" \
      password="mysecretpassword"

  # Create a role with a short TTL
  docker exec "$vault_container_name" vault write database/roles/my-role-vault \
      db_name=my-database \
      creation_statements="CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'; GRANT SELECT ON ALL TABLES IN SCHEMA public TO \"{{name}}\";" \
      default_ttl="10s" \
      max_ttl="10s"

  docker exec "$bao_container_name" bao write database/roles/my-role-bao \
      db_name=my-database \
      creation_statements="CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'; GRANT SELECT ON ALL TABLES IN SCHEMA public TO \"{{name}}\";" \
      default_ttl="10s" \
      max_ttl="10s"

  # Set the environment variables so they can be renewed
  export DATABASE_USERNAME="vault:database/creds/my-role-vault#username"
  export DATABASE_PASSWORD="vault:database/creds/my-role-vault#password"
  export DATABASE_USERNAME="bao:database/creds/my-role-bao#username"
  export DATABASE_PASSWORD="bao:database/creds/my-role-bao#password"
}

teardown() {
  docker compose down
  docker rm -f my-postgres
  docker network rm my-network

  rm -f "$TMPFILE_SECRET"
  rm -f "$TMPFILE_VAULT_TOKEN"
  rm -f "$TMPFILE_BAO_TOKEN"
  rm -f secret-init
}

assert_output_contains() {
  local expected=$1
  local output=$2

  echo "$output" | grep -qF "$expected" || fail "Expected line not found: $expected"
}

@test "secrets successfully loaded from providers" {
  setup_file_provider

  setup_vault_provider
  set_vault_token 227e1cce-6bf7-30bb-2d2a-acc854318caf
  add_secrets_to_vault

  setup_bao_provider
  set_bao_token 227e1cce-6bf7-30bb-2d2a-acc854318caf
  add_secrets_to_bao

  run_output=$(./secret-init env | grep 'FILE_SECRET\|MYSQL_PASSWORD\|AWS_SECRET_ACCESS_KEY\|AWS_ACCESS_KEY_ID\|API_KEY\|RABBITMQ_USERNAME\|RABBITMQ_PASSWORD')
  assert_success

  assert_output_contains "FILE_SECRET=secret-value" "$run_output"
  assert_output_contains "MYSQL_PASSWORD=3xtr3ms3cr3t" "$run_output"
  assert_output_contains "AWS_SECRET_ACCESS_KEY=s3cr3t" "$run_output"
  assert_output_contains "AWS_ACCESS_KEY_ID=secretId" "$run_output"
  assert_output_contains "API_KEY=sensitiveApiKey" "$run_output"
  assert_output_contains "RABBITMQ_USERNAME=rabbitmqUser" "$run_output"
  assert_output_contains "RABBITMQ_PASSWORD=rabbitmqPassword" "$run_output"
}

@test "secrets successfully loaded using vault:login and bao:login as tokens" {
  setup_file_provider

  setup_vault_provider
  set_vault_token "vault:login"
  add_secrets_to_vault

  setup_bao_provider
  set_bao_token "bao:login"
  add_secrets_to_bao

  run_output=$(./secret-init env | grep 'FILE_SECRET\|MYSQL_PASSWORD\|AWS_SECRET_ACCESS_KEY\|AWS_ACCESS_KEY_ID\|API_KEY\|RABBITMQ_USERNAME\|RABBITMQ_PASSWORD')
  assert_success

  assert_output_contains "FILE_SECRET=secret-value" "$run_output"
  assert_output_contains "MYSQL_PASSWORD=3xtr3ms3cr3t" "$run_output"
  assert_output_contains "AWS_SECRET_ACCESS_KEY=s3cr3t" "$run_output"
  assert_output_contains "AWS_ACCESS_KEY_ID=secretId" "$run_output"
  assert_output_contains "API_KEY=sensitiveApiKey" "$run_output"
  assert_output_contains "RABBITMQ_USERNAME=rabbitmqUser" "$run_output"
  assert_output_contains "RABBITMQ_PASSWORD=rabbitmqPassword" "$run_output"
}

@test "secrets successfully loaded and renewed with daemon mode enabled" {
  setup_file_provider

  setup_vault_provider
  set_vault_token 227e1cce-6bf7-30bb-2d2a-acc854318caf
  add_secrets_to_vault

  setup_bao_provider
  set_bao_token 227e1cce-6bf7-30bb-2d2a-acc854318caf
  add_secrets_to_bao

  set_daemon_mode
  setup_database_for_daemon_mode

  # Generate a new secret and get its lease duration
  secret_info_vault_before=$(docker exec "$vault_container_name" vault read -format=json database/creds/my-role-vault)
  lease_id_vault_before=$(echo "$secret_info_vault_before" | jq -r '.lease_id')

  secret_info_bao_before=$(docker exec "$bao_container_name" bao read -format=json database/creds/my-role-bao)
  lease_id_bao_before=$(echo "$secret_info_bao_before" | jq -r '.lease_id')

  run_output=$(./secret-init env | grep 'FILE_SECRET\|MYSQL_PASSWORD\|AWS_SECRET_ACCESS_KEY\|AWS_ACCESS_KEY_ID\|API_KEY\|RABBITMQ_USERNAME\|RABBITMQ_PASSWORD')
  assert_success

  # Get the lease ID adter renewing the secret
  secret_info_vault_after=$(docker exec "$vault_container_name" vault read -format=json database/creds/my-role-vault)
  lease_id_vault_after=$(echo "$secret_info_vault_after" | jq -r '.lease_id')

  secret_info_bao_after=$(docker exec "$bao_container_name" bao read -format=json database/creds/my-role-bao)
  lease_id_bao_after=$(echo "$secret_info_bao_after" | jq -r '.lease_id')

  assert_output_contains "FILE_SECRET=secret-value" "$run_output"
  assert_output_contains "MYSQL_PASSWORD=3xtr3ms3cr3t" "$run_output"
  assert_output_contains "AWS_SECRET_ACCESS_KEY=s3cr3t" "$run_output"
  assert_output_contains "AWS_ACCESS_KEY_ID=secretId" "$run_output"
  assert_output_contains "API_KEY=sensitiveApiKey" "$run_output"
  assert_output_contains "RABBITMQ_USERNAME=rabbitmqUser" "$run_output"
  assert_output_contains "RABBITMQ_PASSWORD=rabbitmqPassword" "$run_output"

  # Check if the lease ID has changed
  if [ "$lease_id_vault_before" == "$lease_id_vault_after" ]; then
    fail "Secret was not renewed"
  fi

  # Check if the lease ID has changed
  if [ "$lease_id_bao_before" == "$lease_id_bao_after" ]; then
    fail "Secret was not renewed"
  fi
}

@test "secrets successfully loaded using VAULT_FROM_PATH and BAO_FROM_PATH" {
  # unset env vars to ensure secret-init will utilize VAULT_FROM_PATH and BAO_FROM_PATH
  unset MYSQL_PASSWORD
  unset AWS_SECRET_ACCESS_KEY
  unset AWS_ACCESS_KEY_ID
  unset API_KEY
  unset RABBITMQ_USERNAME
  unset RABBITMQ_PASSWORD

  setup_file_provider

  setup_vault_provider
  set_vault_token 227e1cce-6bf7-30bb-2d2a-acc854318caf
  add_secrets_to_vault
  export VAULT_FROM_PATH="secret/data/test/mysql,secret/data/test/aws"

  setup_bao_provider
  set_bao_token 227e1cce-6bf7-30bb-2d2a-acc854318caf
  add_secrets_to_bao
  export BAO_FROM_PATH="secret/data/test/api,secret/data/test/rabbitmq"

  run_output=$(./secret-init env | grep 'FILE_SECRET\|MYSQL_PASSWORD\|AWS_SECRET_ACCESS_KEY\|AWS_ACCESS_KEY_ID\|API_KEY\|RABBITMQ_USERNAME\|RABBITMQ_PASSWORD')
  assert_success

  assert_output_contains "FILE_SECRET=secret-value" "$run_output"
  assert_output_contains "MYSQL_PASSWORD=3xtr3ms3cr3t" "$run_output"
  assert_output_contains "AWS_SECRET_ACCESS_KEY=s3cr3t" "$run_output"
  assert_output_contains "AWS_ACCESS_KEY_ID=secretId" "$run_output"
  assert_output_contains "API_KEY=sensitiveApiKey" "$run_output"
  assert_output_contains "RABBITMQ_USERNAME=rabbitmqUser" "$run_output"
  assert_output_contains "RABBITMQ_PASSWORD=rabbitmqPassword" "$run_output"
}

@test "secrets successfully loaded using different injection cases" {
  setup_file_provider

  setup_vault_provider
  set_vault_token 227e1cce-6bf7-30bb-2d2a-acc854318caf
  add_secrets_to_vault

  setup_bao_provider
  set_bao_token 227e1cce-6bf7-30bb-2d2a-acc854318caf
  add_secrets_to_bao

  # Secret with version
  add_custom_secret_to_vault "secret/test/mysql" "MYSQL_PASSWORD=modify3d3xtr3ms3cr3t"
  export MYSQL_PASSWORD="vault:secret/data/test/mysql#MYSQL_PASSWORD#2"

  add_custom_secret_to_bao "secret/test/api" "API_KEY=modify3dAPIs3cr3t"
  export API_KEY="bao:secret/data/test/api#API_KEY#2"

  # Inline secrets with scheme
  add_custom_secret_to_vault "secret/test/scheme" "SCHEME_SECRET1=sch3m3s3cr3tONE" "SCHEME_SECRET2=sch3m3s3cr3tTWO"
  export SCHEME_SECRET_VAULT="scheme://\${vault:secret/data/test/scheme#SCHEME_SECRET1}:\${vault:secret/data/test/scheme#SCHEME_SECRET2}@$VAULT_ADDR"

  add_custom_secret_to_bao "secret/test/scheme" "SCHEME_SECRET1=sch3m3s3cr3tONE" "SCHEME_SECRET2=sch3m3s3cr3tTWO"
  export SCHEME_SECRET_BAO="scheme://\${bao:secret/data/test/scheme#SCHEME_SECRET1}:\${bao:secret/data/test/scheme#SCHEME_SECRET2}@$BAO_ADDR"

  # Enable pki secrets engine and generate root certificates
  docker exec "$vault_container_name" vault secrets enable -path=pki pki
  export ROOT_CERT_VAULT=">>vault:pki/root/generate/internal#certificate"
  export ROOT_CERT_CACHED_VAULT=">>vault:pki/root/generate/internal#certificate"

  docker exec "$bao_container_name" bao secrets enable -path=pki pki
  export ROOT_CERT_BAO=">>bao:pki/root/generate/internal#certificate"
  export ROOT_CERT_CACHED_BAO=">>bao:pki/root/generate/internal#certificate"

  run_output=$(./secret-init env | grep 'FILE_SECRET\|MYSQL_PASSWORD\|SCHEME_SECRET_VAULT\|ROOT_CERT_VAULT\|ROOT_CERT_CACHED_VAULT\|API_KEY\|SCHEME_SECRET_BAO\|ROOT_CERT_BAO\|ROOT_CERT_CACHED_BAO')
  assert_success

  assert_output_contains "FILE_SECRET=secret-value" "$run_output"
  assert_output_contains "MYSQL_PASSWORD=modify3d3xtr3ms3cr3t" "$run_output"
  assert_output_contains "SCHEME_SECRET_VAULT=scheme://sch3m3s3cr3tONE:sch3m3s3cr3tTWO@$VAULT_ADDR" "$run_output"
  assert_output_contains "API_KEY=modify3dAPIs3cr3t" "$run_output"
  assert_output_contains "SCHEME_SECRET_BAO=scheme://sch3m3s3cr3tONE:sch3m3s3cr3tTWO@$BAO_ADDR" "$run_output"

  [ $ROOT_CERT_VAULT == $ROOT_CERT_CACHED_VAULT ]
  assert_success "ROOT_CERT_VAULT and ROOT_CERT_CACHED_VAULT are not the same"

  [ $ROOT_CERT_BAO == $ROOT_CERT_CACHED_BAO ]
  assert_success "ROOT_CERT_BAO and ROOT_CERT_CACHED_BAO are not the same"
}
