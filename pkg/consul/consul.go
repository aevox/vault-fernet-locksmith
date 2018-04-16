package consul

import (
	"errors"
	"fmt"

	"github.com/golang/glog"
	consulapi "github.com/hashicorp/consul/api"
)

// Consul represents a means for interacting with a remote consul client.
type Consul struct {
	Client *consulapi.Client
}

// NewClient creates a new consul client
func NewClient(address string, token string) (*Consul, error) {
	if address == "" {
		return nil, errors.New("Error creating consul client, consul address is empty")
	}
	config := consulapi.DefaultConfig()
	config.Address = address
	config.Token = token

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
	glog.Infof("Attempting to release lock")
	if err := lock.Unlock(); err != nil {
		return fmt.Errorf("Lock release failed : %s", err)
	}
	glog.Infof("Lock released")
	// Cleanup the lock if no longer in use
	glog.Infof("Cleaning lock entry")
	if err := lock.Destroy(); err != nil {
		if err != consulapi.ErrLockInUse {
			return fmt.Errorf("Lock cleanup failed: %s", err)
		}
		glog.Info("Cleanup aborted, lock in use")
		return nil
	}
	glog.Infof("Cleanup succeeded")
	return nil
}
