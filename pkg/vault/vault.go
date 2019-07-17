package vault

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	vaultapi "github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"
)

// Vault represents a means for interacting with a remote Vault
// instance (unsealed and pre-authenticated) to read and write secrets.
type Vault struct {
	Client     *vaultapi.Client
	RenewToken bool
}

// Reader meant to be used by func that want to read in Vault
type Reader interface {
	Read(path string) ([]byte, error)
}

// Writer meant to be used by func that want to write in Vault
type Writer interface {
	Write(path string, data map[string]interface{}) error
}

// NewClient  creates a new vault client
func NewClient(address string, proxy string, renew bool) (*Vault, error) {
	config := vaultapi.DefaultConfig()

	config.Address = address

	// By default this added the system's CAs
	if err := config.ConfigureTLS(&vaultapi.TLSConfig{Insecure: false}); err != nil {
		return nil, fmt.Errorf("Failed to configure TLS: %v", err)
	}

	// Configure optionnal proxy
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("Error parsing proxy URL: %v", err)
		}
		config.HttpClient = &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	}

	// Create the client
	client, err := vaultapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %s", err)
	}

	return &Vault{Client: client, RenewToken: renew}, nil
}

// Read reads data from vault
func (v *Vault) Read(path string) ([]byte, error) {
	r := v.Client.NewRequest("GET", "/v1/"+path)
	resp, err := v.Client.RawRequest(r)
	if resp != nil {
		defer resp.Body.Close()
	}
	if resp != nil && resp.StatusCode == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error while making request: %v", err)
	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading from buffer: %v", err)
	}
	return buf.Bytes(), nil
}

// Write writes a secret in vault
func (v *Vault) Write(path string, data map[string]interface{}) error {
	_, err := v.Client.Logical().Write(path, data)
	if err != nil {
		return fmt.Errorf("Error writing secret %s to vault: %v", path, err)
	}
	return nil
}

// Delete a secret in vault
func (v *Vault) Delete(path string) error {
	_, err := v.Client.Logical().Delete(path)
	if err != nil {
		return fmt.Errorf("Error Deleting secret %s in vault: %v", path, err)
	}
	return nil
}

// SelfRenew renews the vault client token
func (v *Vault) SelfRenew() error {
	vaultName := v.Client.Address()

	// Renew the auth.
	renewal, err := v.Client.Auth().Token().RenewSelf(0)
	if err != nil {
		return fmt.Errorf("error renewing token of vault %s :%v", vaultName, err)
	}

	// Somehow, sometimes, this happens.
	if renewal == nil || renewal.Auth == nil {
		return errors.New("returned empty secret data")
	}

	// Do nothing if we are not renewable
	if !renewal.Auth.Renewable {
		return errors.New("secret is not renewable")
	}
	return nil
}

// SelfRenewer starts a goroutine that periodically renew the vault client token
func (v *Vault) SelfRenewer() {
	vaultSecret := &vaultapi.Secret{
		Auth: &vaultapi.SecretAuth{
			ClientToken:   v.Client.Token(),
			Renewable:     true,
			LeaseDuration: 1,
		},
	}
	go v.Renewer(vaultSecret)
}

// Renewer creates a secret renewer
func (v *Vault) Renewer(secret *vaultapi.Secret) {
	renewer, err := v.Client.NewRenewer(&vaultapi.RenewerInput{
		Secret: secret,
	})
	if err != nil {
		log.Errorf("Error creating secret renewer: %v", err)
	}

	go renewer.Renew()
	defer renewer.Stop()

	for {
		select {
		case err := <-renewer.DoneCh():
			if err != nil {
				log.Errorf("Error renewing lease: %v", err)
			}

			// Renewal is now over
		case <-renewer.RenewCh():
			log.Debugf("Successfully renewed secret")
		}
	}
}
