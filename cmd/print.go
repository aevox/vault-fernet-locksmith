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

	"github.com/aevox/vault-fernet-locksmith/pkg/vault"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// printCmd represents the print command
var printCmd = &cobra.Command{
	Use:   "print",
	Short: "Print secrets stored in Vault(s)",
	Run: func(cmd *cobra.Command, args []string) {
		vaultClients, err := createVaultClients()
		if err != nil {
			log.Fatalf("Error creating vault clients: %v", err)
		}
		printSecrets(vaultClients)
	},
}

func init() {
	rootCmd.AddCommand(printCmd)
}

func printSecrets(vaultClients []*vault.Vault) {
	vaultClients, err := createVaultClients()
	if err != nil {
		log.Fatalf("Error creating vault clients: %v", err)
	}
	for _, v := range vaultClients {
		s, err := v.Read(cfg.SecretPath)
		if err != nil {
			log.Errorf("Error reading secret in %s: %v", v.Client.Address(), err)
		}
		fmt.Printf("%s:\n%s", v.Client.Address(), s)
	}
}
