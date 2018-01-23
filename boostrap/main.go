package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/aevox/vault-fernet-locksmith/locksmith"
	"github.com/aevox/vault-fernet-locksmith/vault"
)

var (
	opts             options
	locksmithVersion string
)

type options struct {
	version        bool
	numKeys        int
	secretPath     string
	period         int64
	vaultToken     string
	vaultTokenFile string
	ttl            int
}

func init() {
	flag.BoolVar(&opts.version, "version", false, "Prints version and exits")
	flag.StringVar(&opts.vaultTokenFile, "vault-token-file", "", "File containing the vault token used to authenticate with vault")
	flag.StringVar(&opts.vaultToken, "vault-token", "", "Vult token used to authenticate with vault")
	flag.StringVar(&opts.secretPath, "secre-path", "secret/fernet-keys", "Path to the fernet-keys secret in vault")
	flag.IntVar(&opts.ttl, "ttl", 120, "Freshness interval of the fernet token")
	flag.IntVar(&opts.numKeys, "n", 3, "Number of fernet keys")
	flag.Int64Var(&opts.period, "period", 3600, "Period of rotation")
}

func main() {
	//Get configuration
	flag.Parse()

	if opts.version {
		fmt.Println(locksmithVersion)
		os.Exit(0)
	}

	if opts.numKeys < 3 {
		fmt.Println("Keys number must be superior to 3")
		os.Exit(1)
	}

	//Create vault client
	vaultClient, err := vault.NewClient()
	if err != nil {
		fmt.Printf("Failed to create vault client: %v", err)
		os.Exit(1)
	}
	//Set vault client token
	var vaultToken string
	if opts.vaultToken != "" {
		vaultToken = opts.vaultToken
	} else if opts.vaultTokenFile != "" {
		data, err := ioutil.ReadFile(opts.vaultTokenFile)
		if err != nil {
			fmt.Printf("Cannot read token file: %v\n", err)
			os.Exit(1)
		}
		vaultToken = string(data)
	} else if e := os.Getenv("VAULT_TOKEN"); e != "" {
		vaultToken = strings.TrimSpace(e)
	} else {
		fmt.Println("No vault token provided")
		os.Exit(1)
	}
	vaultClient.Client.SetToken(vaultToken)

	k, err := vaultClient.Client.Logical().Read(opts.secretPath)
	if err != nil {
		fmt.Printf("Cannot read secret %s: %v", opts.secretPath, err)
		os.Exit(1)
	}
	if k != nil {
		fmt.Println("Doing nothing, a secret exists")
		os.Exit(1)
	}

	fernetKeys, err := locksmith.NewFernetKeys(opts.period, opts.numKeys)
	if err != nil {
		fmt.Printf("Error creating new fernet keys: %v", err)
		os.Exit(1)
	}

	if err := locksmith.WriteKeys(vaultClient, fernetKeys, opts.secretPath, opts.ttl); err != nil {
		fmt.Printf("Error writing fernet keys to vault")
	}
	fmt.Println("Success! New keys written in vault")
}
