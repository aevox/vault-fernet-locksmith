package main

import (
	"errors"
	"fmt"

	"github.com/aevox/vault-fernet-locksmith/pkg/vault"
	consulapi "github.com/hashicorp/consul/api"

	health "github.com/docker/go-healthcheck"
)

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
