package main

import (
	"errors"
	"fmt"

	"github.com/aevox/vault-fernet-locksmith/vault"
	consulapi "github.com/hashicorp/consul/api"

	health "github.com/docker/go-healthcheck"
)

func vaultChecker(v *vault.Vault, path string) health.Checker {
	return health.CheckFunc(func() error {
		_, err := v.Read(path)
		if err != nil {
			return fmt.Errorf("Cannot access vault: %v", err)
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
