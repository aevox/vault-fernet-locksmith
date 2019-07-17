// Copyright © 2019 Marc Fouché
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

package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aevox/vault-fernet-locksmith/pkg/consul"
	"github.com/aevox/vault-fernet-locksmith/pkg/locksmith"
	"github.com/aevox/vault-fernet-locksmith/pkg/vault"

	health "github.com/docker/go-healthcheck"
	"github.com/gorilla/mux"
	consulapi "github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// watchCmd represents the watch command
var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch keys in Vault(s) and rotate them when needed",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		vaultClients, err := createVaultClients()
		if err != nil {
			log.Fatalf("Error creating vault clients: %v", err)
		}
		watch(vaultClients)
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)

	watchCmd.Flags().Int("ttl", 120, "Interval between each vault secret fetch")
	watchCmd.Flags().Bool("health", false, "enable endpoint /health on port 8080")
	watchCmd.Flags().Int("health-period", 120, "period between each health check in seconds")
	watchCmd.Flags().Bool("consul-lock", false, "acquires a lock with consul to ensure that only one instance of locksmith is running")
	watchCmd.Flags().String("consul-lock-key", "locks/locksmith/.lock", "Key used by consul lock")
	watchCmd.Flags().String("consul-address", "http://127.0.0.1:8200", "Consul address")
	watchCmd.Flags().String("consul-proxy", "", "Proxy URL used to contact Consul")
	watchCmd.Flags().String("consul-token", "", "Consul token used to authenticate with consul")
	watchCmd.Flags().String("consul-token-file", "", "file containing the vault token used to authenticate with Consul")

	viper.BindPFlag("ttl", watchCmd.Flags().Lookup("ttl"))
	viper.BindPFlag("health", watchCmd.Flags().Lookup("health"))
	viper.BindPFlag("healthPeriod", watchCmd.Flags().Lookup("health-period"))
	viper.BindPFlag("consul.lock", watchCmd.Flags().Lookup("consul-lock"))
	viper.BindPFlag("consul.lockKey", watchCmd.Flags().Lookup("consul-lock-key"))
	viper.BindPFlag("consul.address", watchCmd.Flags().Lookup("consul-address"))
	viper.BindPFlag("consul.proxy", watchCmd.Flags().Lookup("consul-proxy"))
	viper.BindPFlag("consul.token", watchCmd.Flags().Lookup("consul-token"))
	viper.BindPFlag("consul.tokenFile", watchCmd.Flags().Lookup("consul-token-file"))
}

func watch(vaultClients []*vault.Vault) {
	for _, v := range vaultClients {
		if v.RenewToken {
			go func(vc *vault.Vault) {
				for c := time.Tick(time.Duration(cfg.TTL) * time.Second); ; <-c {
					log.Debugf("Renewing vault token for %s", vc.Client.Address())
					if err := vc.SelfRenew(); err != nil {
						log.Warningf("Something went wrong renewing vault token for %s: %v", vc.Client.Address(), err)
					}
				}
			}(v)
		}
	}

	if cfg.Health {
		for _, vaultClient := range vaultClients {
			health.Register(fmt.Sprintf("vaultChecker-%s", vaultClient.Client.Address()), health.PeriodicChecker(vaultChecker(vaultClient, cfg.SecretPath), time.Second*time.Duration(cfg.HealthPeriod)))
		}

		go func() {
			// create http server to expose health status
			r := mux.NewRouter()

			r.HandleFunc("/health", health.StatusHandler)

			srv := &http.Server{
				Handler:     r,
				Addr:        "0.0.0.0:8080",
				ReadTimeout: 15 * time.Second,
			}
			log.Fatal(srv.ListenAndServe())
		}()
	}

	if cfg.Consul.Lock {
		log.Debug("Creating consul client")
		var consulToken string
		if cfg.Consul.Token != "" {
			consulToken = cfg.Consul.Token
		} else if cfg.Consul.TokenFile != "" {
			data, err := ioutil.ReadFile(cfg.Consul.TokenFile)
			if err != nil {
				log.Fatalf("Cannot read vault token file: %v", err)
			}
			consulToken = string(data)
		}

		consulClient, err := consul.NewClient(cfg.Consul.Address, cfg.Consul.Proxy, consulToken)
		if err != nil {
			log.Fatalf("Failed to create consul client: %v", err)
		}
		if cfg.Health {
			health.Register("consulChecker", health.PeriodicChecker(consulChecker(consulClient.Client, cfg.Consul.LockKey), time.Second*time.Duration(cfg.HealthPeriod)))
		}

		log.Info("Attempting to acquire lock...")
		lock, err := consulClient.Client.LockKey(cfg.Consul.LockKey)
		if err != nil {
			log.Fatalf("Lock setup failed :%v", err)
		}
		stopCh := make(chan struct{})
		lockCh, err := lock.Lock(stopCh)
		if err != nil {
			log.Fatalf("Failed acquiring lock: %v", err)
		}
		log.Info("Lock acquired")

		// Handle SIGINT and SIGTERM.
		sigs := make(chan os.Signal)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			for {
				select {
				case <-lockCh:
					log.Fatal("Lost lock, Exting")
				case sig := <-sigs:
					log.Infof("Recieved signal: %v", sig)
					if cfg.Consul.Lock {
						// Attempt to release lock and destroy it
						if err := consul.CleanLock(lock); err != nil {
							log.Fatalf("Error cleaning consul lock: %v", err)
						}
					}
					os.Exit(0)
				}
			}
		}()
	}

	log.Info("Starting")
	// run smith() every TTL
	for c := time.Tick(time.Duration(cfg.TTL) * time.Second); ; <-c {
		if err := smith(vaultClients, cfg.SecretPath, cfg.TTL); err != nil {
			log.Error(err)
			continue
		}
	}
}

// smith reads the fernet keys in vault and rotates them when their age is less than a TTL away
// to be equal to the period of rotation.
// If ls.RenewVaultToken is true, it tries to renew the vault clients token before reading secrets.
func smith(vlist []*vault.Vault, path string, ttl int) error {

	log.Debug("Getting fernet keys")
	fkeys, err := locksmith.GetFernetKeys(vlist, path)
	if err != nil {
		return fmt.Errorf("Cannot smith new keys: %v", err)
	}

	if time.Now().Unix() < (fkeys.CreationTime + fkeys.Period - int64(ttl)) {
		log.Debug("All keys are fresh, no rotation needed")
		return nil
	}

	log.Info("Time to rotate keys")
	// rotate(0) means that we do not change the period
	if err := fkeys.Rotate(0); err != nil {
		return fmt.Errorf("Error rotating keys: %v", err)
	}
	log.Debugf("New keys: %v", *fkeys)

	for _, v := range vlist {
		vaultName := v.Client.Address()
		log.Infof("Writing keys to %s", vaultName)
		if err := locksmith.WriteFernetKeys(v, path, fkeys, ttl); err != nil {
			return fmt.Errorf("Cannot write fernet keys to %s : %v", vaultName, err)
		}
		log.Debugf("Keys written to %s", vaultName)
	}
	log.Infof("Rotation complete")

	return nil
}

func vaultChecker(v *vault.Vault, path string) health.Checker {
	return health.CheckFunc(func() error {
		b, err := v.Read(path)
		if err != nil {
			return fmt.Errorf("Cannot access vault: %v", err)
		}
		if b == nil {
			return fmt.Errorf("%s is empty in %s", path, v.Client.Address())
		}
		return nil
	})
}

func consulChecker(c *consulapi.Client, key string) health.Checker {
	return health.CheckFunc(func() error {
		lock, _, err := c.KV().Get(key, &consulapi.QueryOptions{})
		if err != nil {
			return fmt.Errorf("Cannot access consul lock: %v", err)
		}
		if lock == nil {
			return errors.New("Lock does not exist")
		}
		return nil
	})
}
