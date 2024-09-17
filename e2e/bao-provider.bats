bao_container_name="secret-init-bao"

setup() {
  bats_load_library bats-support
  bats_load_library bats-assert

  run go build
  assert_success
}

setup_bao_provider() {
  TMPFILE_TOKEN=$(mktemp)
  printf "227e1cce-6bf7-30bb-2d2a-acc854318caf" > "$TMPFILE_TOKEN"

  export BAO_ADDR="http://127.0.0.1:8300"
  export BAO_TOKEN_FILE="$TMPFILE_TOKEN"

  export API_KEY="bao:secret/data/test/api#API_KEY"
  export RABBITMQ_USERNAME="bao:secret/data/test/rabbitmq#RABBITMQ_USERNAME"
  export RABBITMQ_PASSWORD="bao:secret/data/test/rabbitmq#RABBITMQ_PASSWORD"

  start_bao
}

start_bao() {
  docker compose up -d

  # wait for Bao to be ready
  max_attempts=${MAX_ATTEMPTS:-10}
  for ((attempts = 0; attempts < max_attempts; attempts++)); do
    if docker compose exec -T "$bao_container_name"  bao status > /dev/null 2>&1; then
      break
    fi
    sleep 1
  done
}

set_bao_token() {
  local token=$1
  export BAO_TOKEN="$token"
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

  docker network connect my-network "$bao_container_name"

  # Enable the database secrets engine
  docker exec "$bao_container_name" bao secrets enable database

  # Configure the database secrets engine
  docker exec "$bao_container_name" bao write database/config/my-database \
      plugin_name=postgresql-database-plugin \
      allowed_roles="my-role" \
      connection_url="postgresql://postgres:mysecretpassword@my-postgres:5432/mydb?sslmode=disable" \
      username="postgres" \
      password="mysecretpassword"

  # Create a role with a short TTL
  docker exec "$bao_container_name" bao write database/roles/my-role \
      db_name=my-database \
      creation_statements="CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'; GRANT SELECT ON ALL TABLES IN SCHEMA public TO \"{{name}}\";" \
      default_ttl="10s" \
      max_ttl="10s"

  # Set the environment variables so they can be renewed
  export DATABASE_USERNAME="bao:database/creds/my-role#username"
  export DATABASE_PASSWORD="bao:database/creds/my-role#password"
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

teardown() {
  docker compose down
  docker rm -f my-postgres
  docker network rm my-network

  rm -f "$TMPFILE_TOKEN"
  rm -f secret-init
}

assert_output_contains() {
  local expected=$1
  local output=$2

  echo "$output" | grep -qF "$expected" || fail "Expected line not found: $expected"
}

@test "secrets successfully loaded from bao" {
  setup_bao_provider
  set_bao_token 227e1cce-6bf7-30bb-2d2a-acc854318caf
  add_secrets_to_bao

  run_output=$(./secret-init env | grep 'API_KEY\|RABBITMQ_USERNAME\|RABBITMQ_PASSWORD')
  assert_success

  assert_output_contains "API_KEY=sensitiveApiKey" "$run_output"
  assert_output_contains "RABBITMQ_USERNAME=rabbitmqUser" "$run_output"
  assert_output_contains "RABBITMQ_PASSWORD=rabbitmqPassword" "$run_output"
}

@test "secrets successfully loaded from bao using bao:login as token" {
  setup_bao_provider
  set_bao_token "bao:login"
  add_secrets_to_bao

  run_output=$(./secret-init env | grep 'API_KEY\|RABBITMQ_USERNAME\|RABBITMQ_PASSWORD')
  assert_success

  assert_output_contains "API_KEY=sensitiveApiKey" "$run_output"
  assert_output_contains "RABBITMQ_USERNAME=rabbitmqUser" "$run_output"
  assert_output_contains "RABBITMQ_PASSWORD=rabbitmqPassword" "$run_output"
}

@test "secrets successfully loaded and renewed from bao with daemon mode enabled" {
  setup_bao_provider
  set_bao_token 227e1cce-6bf7-30bb-2d2a-acc854318caf
  add_secrets_to_bao

  set_daemon_mode
  setup_database_for_daemon_mode

  # Generate a new secret and get its lease ID
  secret_info_before=$(docker exec "$bao_container_name" bao read -format=json database/creds/my-role)
  lease_id_before=$(echo "$secret_info_before" | jq -r '.lease_id')

  run_output=$(./secret-init env | grep 'API_KEY\|RABBITMQ_USERNAME\|RABBITMQ_PASSWORD')
  assert_success

  # Get the lease ID after renewing the secret
  secret_info_after=$(docker exec "$bao_container_name" bao read -format=json database/creds/my-role)
  lease_id_after=$(echo "$secret_info_after" | jq -r '.lease_id')

  assert_output_contains "API_KEY=sensitiveApiKey" "$run_output"
  assert_output_contains "RABBITMQ_USERNAME=rabbitmqUser" "$run_output"
  assert_output_contains "RABBITMQ_PASSWORD=rabbitmqPassword" "$run_output"

  # Check if the lease ID has changed
  if [ "$lease_id_before" == "$lease_id_after" ]; then
    fail "Secret was not renewed"
  fi
}

@test "secrets successfully loaded from bao using BAO_FROM_PATH" {
  setup_bao_provider

  # unset env vars to ensure secret-init will utilize BAO_FROM_PATH
  unset API_KEY
  unset RABBITMQ_USERNAME
  unset RABBITMQ_PASSWORD

  set_bao_token 227e1cce-6bf7-30bb-2d2a-acc854318caf
  add_secrets_to_bao
  export BAO_FROM_PATH="secret/data/test/api,secret/data/test/rabbitmq"

  run_output=$(./secret-init env | grep 'API_KEY\|RABBITMQ_USERNAME\|RABBITMQ_PASSWORD')
  assert_success

  assert_output_contains "API_KEY=sensitiveApiKey" "$run_output"
  assert_output_contains "RABBITMQ_USERNAME=rabbitmqUser" "$run_output"
  assert_output_contains "RABBITMQ_PASSWORD=rabbitmqPassword" "$run_output"
}

@test "secrets successfully loaded from bao using different injection cases" {
  setup_bao_provider
  set_bao_token 227e1cce-6bf7-30bb-2d2a-acc854318caf
  add_secrets_to_bao

  # Secret with version
  add_custom_secret_to_bao "secret/test/api" "API_KEY=modify3dAPIs3cr3t"
  export API_KEY="bao:secret/data/test/api#API_KEY#2"

  # Inline secrets with scheme
  add_custom_secret_to_bao "secret/test/scheme" "SCHEME_SECRET1=sch3m3s3cr3tONE" "SCHEME_SECRET2=sch3m3s3cr3tTWO"
  export SCHEME_SECRET_BAO="scheme://\${bao:secret/data/test/scheme#SCHEME_SECRET1}:\${bao:secret/data/test/scheme#SCHEME_SECRET2}@$BAO_ADDR"

  # Enable pki secrets engine and generate root certificates
  docker exec "$bao_container_name" bao secrets enable -path=pki pki
  export ROOT_CERT=">>bao:pki/root/generate/internal#certificate"
  export ROOT_CERT_CACHED=">>bao:pki/root/generate/internal#certificate"

  run_output=$(./secret-init env | grep 'API_KEY\|SCHEME_SECRET\|ROOT_CERT\|ROOT_CERT_CACHED')
  assert_success

  assert_output_contains "API_KEY=modify3dAPIs3cr3t" "$run_output"
  assert_output_contains "SCHEME_SECRET_BAO=scheme://sch3m3s3cr3tONE:sch3m3s3cr3tTWO@$BAO_ADDR" "$run_output"

  [ $ROOT_CERT == $ROOT_CERT_CACHED ]
  assert_success "ROOT_CERT and ROOT_CERT_CACHED are not the same"
}
