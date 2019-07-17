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
	"fmt"

	"github.com/aevox/vault-fernet-locksmith/pkg/locksmith"
	"github.com/aevox/vault-fernet-locksmith/pkg/vault"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var forceBootstrap bool

// bootstrapCmd represents the bootstrap command
var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Generate first set of fernet keys in Vault(s)",
	Long: `Create n fernet keys (n > 2) and store them as a secret in Vault(s).
The secret is a list of keys with associated with a creation time, a TTL and a period.`,
	// TODO:
	// If a secret alreay exists in the primary Vault, it copies it into the the secondary Vaults
	Run: func(cmd *cobra.Command, args []string) {
		vaultClients, err := createVaultClients()
		if err != nil {
			log.Fatalf("Error creating vault clients: %v", err)
		}

		bootstrap(vaultClients)
	},
}

func init() {
	rootCmd.AddCommand(bootstrapCmd)

	bootstrapCmd.Flags().IntP("num-keys", "k", 3, "number of fernet keys to create")
	bootstrapCmd.Flags().Int64P("period", "p", 3600, "period between each key rotation in seconds")
	bootstrapCmd.Flags().BoolVar(&forceBootstrap, "force", false, "force bootstraping over existing keys")

	viper.BindPFlag("bootstrap.numKeys", bootstrapCmd.Flags().Lookup("num-keys"))
	viper.BindPFlag("bootstrap.period", bootstrapCmd.Flags().Lookup("period"))
}

func bootstrap(vaultClients []*vault.Vault) {
	if cfg.Bootstrap.NumKeys < 3 {
		log.Fatal("Keys number must be superior to 3")
	}

	if cfg.Bootstrap.Period <= 1 {
		log.Fatal("Keys period must be superior to 0")
	}

	// Create the new fernet keys
	fernetKeys, err := locksmith.NewFernetKeys(cfg.Bootstrap.Period, cfg.Bootstrap.NumKeys)
	if err != nil {
		log.Fatalf("Error creating new fernet keys: %v", err)
	}

	// Write fernet keys to Vault
	for _, v := range vaultClients {
		log.Debugf("Reading secret in %s", v.Client.Address())
		s, err := v.Read(cfg.SecretPath)
		if err != nil {
			log.Fatalf("Cannot read secret from %s: %v", v.Client.Address(), err)
		}

		// Exit if keys already exist. Continue if the option --force is set.
		if !forceBootstrap {
			if s != nil {
				log.Fatalf("Keys already exist in Vault %s. Use the option --force if you want to bootstrap over it", v.Client.Address())
			}
		}
	}

	for _, v := range vaultClients {
		log.Infof("Writing keys to %s", v.Client.Address())
		if err := locksmith.WriteFernetKeys(v, cfg.SecretPath, fernetKeys, cfg.TTL); err != nil {
			log.Fatalf("Error bootstraping keys: Error writing keys to %s : %v", v.Client.Address(), err)
		}
	}
	fmt.Println("Bootstrap done")
}
