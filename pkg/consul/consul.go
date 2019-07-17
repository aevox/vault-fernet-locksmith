package consul

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	consulapi "github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
)

// Consul represents a means for interacting with a remote consul client.
type Consul struct {
	Client *consulapi.Client
}

// NewClient creates a new consul client
func NewClient(address string, proxy string, token string) (*Consul, error) {
	if address == "" {
		return nil, errors.New("Error creating consul client, consul address is empty")
	}
	config := consulapi.DefaultConfig()
	config.Address = address
	config.Token = token

	// Configure optionnal proxy
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("Error parsing proxy URL: %v", err)
		}
		config.HttpClient = &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	}

	client, err := consulapi.NewClient(config)
	if err != nil {
		return nil, err
	}
	_, err = client.Status().Leader()
	if err != nil {
		return nil, fmt.Errorf("ERROR communicating with consul server: %v", err)
	}

	return &Consul{Client: client}, nil
}

// CleanLock attempts to release a lock and destroy it
func CleanLock(lock *consulapi.Lock) error {
	// Release the lock
	log.Infof("Attempting to release lock")
	if err := lock.Unlock(); err != nil {
		return fmt.Errorf("Lock release failed : %s", err)
	}
	log.Infof("Lock released")
	// Cleanup the lock if no longer in use
	log.Infof("Cleaning lock entry")
	if err := lock.Destroy(); err != nil {
		if err != consulapi.ErrLockInUse {
			return fmt.Errorf("Lock cleanup failed: %s", err)
		}
		log.Info("Cleanup aborted, lock in use")
		return nil
	}
	log.Infof("Cleanup succeeded")
	return nil
}
