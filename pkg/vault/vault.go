package vault

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"

	"github.com/golang/glog"
	vaultapi "github.com/hashicorp/vault/api"
)

// Vault represents a means for interacting with a remote Vault
// instance (unsealed and pre-authenticated) to read and write secrets.
type Vault struct {
	Client *vaultapi.Client
}

// NewClient  creates a new vault client
func NewClient(address string, proxy string) (*Vault, error) {
	config := vaultapi.DefaultConfig()

	config.Address = address

	// By default this added the system's CAs
	if err := config.ConfigureTLS(&vaultapi.TLSConfig{Insecure: false}); err != nil {
		return nil, fmt.Errorf("Failed to configureTLS: %v", err)
	}

	// Configure optionnal proxy
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			fmt.Errorf("Error parsing proxy URL: %v", err)
		}
		config.HttpClient = &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	}

	// Create the client
	client, err := vaultapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %s", err)
	}

	return &Vault{Client: client}, nil
}

// RenewToken starts a goroutine that periodically renew the vault client token
func (v *Vault) RenewToken() {
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
		glog.Errorf("Error creating secret renewer: %v", err)
	}

	go renewer.Renew()
	defer renewer.Stop()

	for {
		select {
		case err := <-renewer.DoneCh():
			if err != nil {
				glog.Errorf("Error renewing lease: %v", err)
			}

			// Renewal is now over
		case <-renewer.RenewCh():
			glog.V(1).Info("Successfully renewed secret")
		}
	}
}

// ReadSecret reads a secret and starts a renewer to renew its lease.
func (v *Vault) ReadSecret(path string) (*vaultapi.Secret, error) {
	secret, err := v.Client.Logical().Read(path)
	if err != nil {
		return nil, fmt.Errorf("Error reading secret from vault: %v", err)
	}
	return secret, nil
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
