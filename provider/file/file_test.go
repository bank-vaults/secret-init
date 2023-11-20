// Copyright © 2023 Bank-Vaults Maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package file

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	exitCode := m.Run()

	// teardown environment variables after tests are done
	teardownEnvs()

	os.Exit(exitCode)
}

func TestNewFileProvider(t *testing.T) {
	// create a new secret file and write secrets into it
	tmpfile := createTempFileWithContent(t)
	defer os.Remove(tmpfile.Name())

	// create new environment variables
	setupEnvs(t, tmpfile)

	fileProvider, err := NewFileProvider(os.Getenv("SECRETS_FILE_PATH"))
	if err != nil {
		t.Fatal(err)
	}

	// check if file provider is correctly created
	_, ok := fileProvider.(*Provider)
	if !ok {
		t.Fatal("provider is not of type file")
	}
}

func TestFileLoadSecrets(t *testing.T) {
	// create a new secret-file and write secrets into it
	tmpfile := createTempFileWithContent(t)
	defer os.Remove(tmpfile.Name())

	// create new environment variables
	// for file-path and secrets to get
	setupEnvs(t, tmpfile)

	fileProvider, err := NewFileProvider(os.Getenv("SECRETS_FILE_PATH"))
	if err != nil {
		t.Fatal(err)
	}

	environ := make(map[string]string, len(os.Environ()))
	for _, env := range os.Environ() {
		split := strings.SplitN(env, "=", 2)
		name := split[0]
		value := split[1]
		environ[name] = value
	}

	ctx := context.Background()
	envs, err := fileProvider.LoadSecrets(ctx, environ)
	if err != nil {
		t.Fatal(err)
	}

	test := []string{
		"MYSQL_PASSWORD=3xtr3ms3cr3t",
		"AWS_SECRET_ACCESS_KEY=s3cr3t",
		"AWS_ACCESS_KEY_ID=secretId",
	}

	// check if secrets have been correctly loaded
	areEqual(t, envs, test)
}

func areEqual(t *testing.T, actual, expected []string) {
	actualMap := make(map[string]string, len(expected))
	expectedMap := make(map[string]string, len(expected))

	for _, env := range actual {
		split := strings.SplitN(env, "=", 2)
		key := split[0]
		value := split[1]
		actualMap[key] = value
	}

	for _, env := range expected {
		split := strings.SplitN(env, "=", 2)
		key := split[0]
		value := split[1]
		expectedMap[key] = value
	}

	for key, actualValue := range actualMap {
		expectedValue, ok := expectedMap[key]
		if !ok || actualValue != expectedValue {
			t.Fatalf("mismatch for key %s: actual: %s, expected: %s", key, actualValue, expectedValue)
		}
	}
}

func createTempFileWithContent(t *testing.T) *os.File {
	content := []byte("sqlPassword: 3xtr3ms3cr3t\nawsSecretAccessKey: s3cr3t\nawsAccessKeyId: secretId\n")
	tmpfile, err := os.CreateTemp("", "secrets-*.yaml")
	if err != nil {
		t.Fatal(err)
	}

	_, err = tmpfile.Write(content)
	if err != nil {
		t.Fatal(err)
	}

	err = tmpfile.Close()
	if err != nil {
		t.Fatal(err)
	}

	return tmpfile
}

func setupEnvs(t *testing.T, tmpfile *os.File) {
	err := os.Setenv("PROVIDER", "file")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Setenv("SECRETS_FILE_PATH", tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	err = os.Setenv("MYSQL_PASSWORD", "file:sqlPassword")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Setenv("AWS_SECRET_ACCESS_KEY", "file:awsSecretAccessKey")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Setenv("AWS_ACCESS_KEY_ID", "file:awsAccessKeyId")
	if err != nil {
		t.Fatal(err)
	}
}

func teardownEnvs() {
	os.Unsetenv("PROVIDER")
	os.Unsetenv("SECRETS_FILE_PATH")
	os.Unsetenv("MYSQL_PASSWORD")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	os.Unsetenv("AWS_ACCESS_KEY_ID")
}
