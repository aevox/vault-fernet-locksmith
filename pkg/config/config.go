package config

import (
	"flag"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Configuration holds all the configuration for locksmith
type Configuration struct {
	PrimaryVault    VaultConfiguration   `mapstructure:"primary-vault"`
	SecondaryVaults []VaultConfiguration `mapstructure:"secondary-vaults"`
	TTL             int                  `mapstructure:"ttl"`            // Interval between each poll on vault
	SecretPath      string               `mapstructure:"secret-path"`    // Path in vault for fernet-keys secret
	CfgFile         string               `mapstructure:"config"`         // Path to locksmith config file
	Health          bool                 `mapstructure:"health"`         // Enable health endpoint
	HealthPeriod    int                  `mapstructure:"health-period"`  // Period between each health check in seconds
	Lock            bool                 `mapstructure:"lock"`           // Use consul lock system
	LockKey         string               `mapstructure:"lock-key"`       // What key is used for the consul lock system
	ConsulAddress   string               `mapstructure:"consul-address"` // Consul address
	ConsulToken     string               `mapstructure:"consul-token"`   // Consul token use to access consul to read configuration and write lockKey
	Version         bool                 `mapstructure:"version"`
}

// VaultConfiguration holds all the options to create a vault client
type VaultConfiguration struct {
	Address    string `mapstructure:"address"`     // Vault address
	ProxyURL   string `mapstructure:"proxy"`       // Path to proxy
	Token      string `mapstructure:"token"`       // Vault token used to identify with this vault
	TokenFile  string `mapstructure:"token-file"`  // Path to file containing vault token
	TokenRenew bool   `mapstructure:"token-renew"` // Enable token renewal
}

// DefineCmdFlags define the command line flags.
func DefineCmdFlags(cfg *Configuration) {
	// TODO: make version a cmd
	flag.BoolVar(&cfg.Version, "version", false, "Prints locksmith version, exits")
	flag.StringVar(&cfg.PrimaryVault.Token, "vault-token", "", "Vault token used to authenticate with vault")
	flag.StringVar(&cfg.PrimaryVault.Address, "vault-address", "https://127.0.0.1:8500", "Primary vault address")
	flag.StringVar(&cfg.PrimaryVault.TokenFile, "vault-token-file", "", "File containing the vault token used to authenticate with vault")
	flag.BoolVar(&cfg.PrimaryVault.TokenRenew, "renew-vault-token", false, "Renew vault token")
	flag.StringVar(&cfg.CfgFile, "config", "", "Path to config file")
	flag.IntVar(&cfg.TTL, "ttl", 120, "Interval between each vault secret fetch")
	flag.StringVar(&cfg.SecretPath, "secret-path", "secret/fernet-keys", "Path to the fernet-keys secret in vault")
	flag.BoolVar(&cfg.Lock, "lock", false, "Acquires a lock with consul to ensure that only one instance of locksmith is running")
	flag.StringVar(&cfg.LockKey, "lock-key", "locks/locksmith/.lock", "Key used for locking")
	flag.BoolVar(&cfg.Health, "health", false, "Enable endpoint /health on port 8080")
	flag.IntVar(&cfg.HealthPeriod, "health-period", 30, "Period between each health check in seconds")
	flag.StringVar(&cfg.ConsulAddress, "consul-address", "http://127.0.0.1:8200", "Consul address")
	flag.StringVar(&cfg.ConsulToken, "consul-token", "", "Consul token used to authenticate with consul")
}

// GetConfig aggregates all the configuration and create the configuration file
func GetConfig(cfg *Configuration) error {
	flag.Parse()
	viper.SetEnvPrefix("VFL")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	if cfg.CfgFile != "" {
		viper.SetConfigFile(cfg.CfgFile)

		err := viper.ReadInConfig()
		if err != nil {
			return fmt.Errorf("Error reading configuration: %v", err)
		}
	}

	err := viper.Unmarshal(&cfg)
	if err != nil {
		return fmt.Errorf(" Error creating configuration struct: %v", err)
	}

	return nil
}
