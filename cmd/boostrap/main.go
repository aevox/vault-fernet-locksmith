package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/aevox/vault-fernet-locksmith/pkg/config"
	"github.com/aevox/vault-fernet-locksmith/pkg/locksmith"
	"github.com/aevox/vault-fernet-locksmith/pkg/vault"
)

var (
	cfg              config.Configuration
	opts             options
	locksmithVersion string
)

type options struct {
	numKeys int
	period  int64
}

func init() {
	config.DefineCmdFlags(&cfg)
	flag.IntVar(&opts.numKeys, "n", 3, "Number of fernet keys")
	flag.Int64Var(&opts.period, "period", 3600, "Period of rotation")
}

func main() {
	if err := config.GetConfig(&cfg); err != nil {
		fmt.Printf("Error getting configuration: %v", err)
		os.Exit(1)
	}

	if cfg.Version {
		fmt.Println(locksmithVersion)
		os.Exit(0)
	}

	if opts.numKeys < 3 {
		fmt.Println("Keys number must be superior to 3")
		os.Exit(1)
	}

	var vaultClients []*vault.Vault
	vaultsConfig := append([]config.VaultConfiguration{cfg.PrimaryVault}, cfg.SecondaryVaults...)
	for _, vaultConfig := range vaultsConfig {
		vaultClient, err := vault.NewClient(vaultConfig.Address, vaultConfig.ProxyURL)
		if err != nil {
			fmt.Printf("Failed to create vault client for %s: %v", vaultConfig.Address, err)
			os.Exit(1)
		}
		//Set vault client token
		var vaultToken string
		if vaultConfig.Token != "" {
			vaultToken = vaultConfig.Token
		} else if vaultConfig.TokenFile != "" {
			data, err := ioutil.ReadFile(vaultConfig.TokenFile)
			if err != nil {
				fmt.Printf("Cannot read vault token file: %v", err)
				os.Exit(1)
			}
			vaultToken = string(data)
		} else {
			fmt.Printf("No vault token provided for vault %s", vaultClient.Client.Address())
			os.Exit(1)
		}
		vaultClient.Client.SetToken(vaultToken)

		k, err := vaultClient.Client.Logical().Read(cfg.SecretPath)
		if err != nil {
			fmt.Printf("Cannot read secret %s: %v", cfg.SecretPath, err)
			os.Exit(1)
		}

		if k != nil {
			fmt.Println("Doing nothing, a secret exists")
			os.Exit(1)
		}

		vaultClients = append(vaultClients, vaultClient)
	}

	fernetKeys, err := locksmith.NewFernetKeys(opts.period, opts.numKeys)
	if err != nil {
		fmt.Printf("Error creating new fernet keys: %v", err)
		os.Exit(1)
	}

	ls := &locksmith.LockSmith{
		VaultList: vaultClients,
		KeyPath:   cfg.SecretPath,
		TTL:       cfg.TTL,
	}

	if err := ls.WriteKeys(fernetKeys); err != nil {
		fmt.Printf("Error bootstraping keys: %v", err)
	}

	fmt.Println("Success! New keys written in vault")
}
