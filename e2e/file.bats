setup() {
  bats_load_library bats-support
  bats_load_library bats-assert

  setup_pod

  run go build
  assert_success
}

setup_pod() {
  TMPFILE=$(mktemp)
  printf "secret-value" > "$TMPFILE"

  export SECRET_INIT_PROVIDER="file"
  export FILE_MOUNT_PATH="/"
  export Secret="file:$TMPFILE"
}

teardown() {
  rm -f "$TMPFILE"
  rm -f secret-init
}

@test "secret successfully loaded" {
  run_output=$(./secret-init env | grep Secret)
  assert_success
  expected_output="Secret=secret-value"

  assert_equal "$run_output" "$expected_output"
}