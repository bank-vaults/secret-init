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

package vault

import (
	"fmt"
	"log/slog"
	"os"
	"syscall"
	"time"

	"github.com/bank-vaults/vault-sdk/vault"
	vaultapi "github.com/hashicorp/vault/api"
)

type daemonSecretRenewer struct {
	client *vault.Client
	sigs   chan os.Signal
}

func (r daemonSecretRenewer) Renew(path string, secret *vaultapi.Secret) error {
	watcherInput := vaultapi.LifetimeWatcherInput{Secret: secret}
	watcher, err := r.client.RawClient().NewLifetimeWatcher(&watcherInput)
	if err != nil {
		return fmt.Errorf("failed to create lifetime watcher: %w", err)
	}

	go watcher.Start()

	go func() {
		defer watcher.Stop()
		for {
			select {
			case renewOutput := <-watcher.RenewCh():
				slog.Info("secret renewed", slog.String("path", path), slog.Duration("lease-duration", time.Duration(renewOutput.Secret.LeaseDuration)*time.Second))
			case doneError := <-watcher.DoneCh():
				if !secret.Renewable {
					leaseDuration := time.Duration(secret.LeaseDuration) * time.Second
					time.Sleep(leaseDuration)

					slog.Info("secret lease has expired", slog.String("path", path), slog.Duration("lease-duration", leaseDuration))
				}

				slog.Info("secret renewal has stopped, sending SIGTERM to process", slog.String("path", path), slog.Any("done-error", doneError))

				r.sigs <- syscall.SIGTERM

				timeout := <-time.After(10 * time.Second)
				slog.Info("killing process due to SIGTERM timeout", slog.Time("timeout", timeout))
				r.sigs <- syscall.SIGKILL

				return
			}
		}
	}()

	return nil
}
