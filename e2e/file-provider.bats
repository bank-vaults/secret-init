setup() {
  bats_load_library bats-support
  bats_load_library bats-assert

  run go build
  assert_success
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

teardown() {
  rm -f "$TMPFILE_SECRET"
  rm -f secret-init
}

assert_output_contains() {
  local expected=$1
  local output=$2

  echo "$output" | grep -qF "$expected" || fail "Expected line not found: $expected"
}

@test "secret successfully loaded" {
  setup_file_provider

  run_output=$(./secret-init env | grep FILE_SECRET)
  assert_success

  assert_output_contains "FILE_SECRET=secret-value" "$run_output"
}
