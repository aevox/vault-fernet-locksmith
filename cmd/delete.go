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
	"os"

	"github.com/aevox/vault-fernet-locksmith/pkg/vault"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var forceDelete bool

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete fernet keys secret in Vault(s)",
	Run: func(cmd *cobra.Command, args []string) {
		vaultClients, err := createVaultClients()
		if err != nil {
			log.Fatalf("Error creating vault clients: %v", err)
		}
		deleteSecrets(vaultClients)
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().BoolVar(&forceDelete, "force", false, "force deletion")
}

func deleteSecrets(vaultClients []*vault.Vault) {
	var input string
	if !forceDelete {
		fmt.Printf("Delete %s (y/N):", cfg.SecretPath)
		fmt.Scanln(&input)
	}
	if input == "y" || input == "Y" || input == "yes" || forceDelete {
		for _, v := range vaultClients {
			if err := v.Delete(cfg.SecretPath); err != nil {
				log.Errorf("Error Deleting secret in %s: %v", v.Client.Address(), err)
			} else {
				fmt.Printf("%s deleted in vault %s\n", cfg.SecretPath, v.Client.Address())
			}
		}
	} else {
		fmt.Println("Doing nothing")
	}
	os.Exit(0)
}
