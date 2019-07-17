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
	"github.com/aevox/vault-fernet-locksmith/pkg/locksmith"
	"github.com/aevox/vault-fernet-locksmith/pkg/vault"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rotateCmdPeriod int64

// rotateCmd represents the rotate command
var rotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Force a fernet keys rotation",
	Run: func(cmd *cobra.Command, args []string) {
		vaultClients, err := createVaultClients()
		if err != nil {
			log.Fatalf("Error creating vault clients: %v", err)
		}
		rotate(vaultClients)
	},
}

func init() {
	rootCmd.AddCommand(rotateCmd)

	rotateCmd.Flags().Int64VarP(&rotateCmdPeriod, "period", "p", 0, "period between each key rotation. Do not change the period if it is 0")
}

func rotate(vaultClients []*vault.Vault) {
	fkeys, err := locksmith.GetFernetKeys(vaultClients, cfg.SecretPath)
	if err != nil {
		log.Fatalf("Cannot rotate keys: %v", err)
	}

	if err := fkeys.Rotate(rotateCmdPeriod); err != nil {
		log.Fatalf("Error rotating keys: %v", err)
	}

	for _, v := range vaultClients {
		vaultName := v.Client.Address()
		log.Infof("Writing keys to %s", vaultName)
		if err := locksmith.WriteFernetKeys(v, cfg.SecretPath, fkeys, cfg.TTL); err != nil {
			log.Fatalf("Cannot write fernet keys to %s : %v", vaultName, err)
		}
		log.Debugf("Keys written to %s", vaultName)
	}
	log.Info("Rotation complete")
}
