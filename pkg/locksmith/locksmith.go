package locksmith

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/fernet/fernet-go"

	"github.com/aevox/vault-fernet-locksmith/pkg/vault"
)

// FernetKeys represents the fernet keys and their metadata
type FernetKeys struct {
	Keys         []string `json:"keys"`
	CreationTime int64    `json:"creation_time"`
	Period       int64    `json:"period"`
}

// KeysSecret is used to unmarshal the secret from Vault
type KeysSecret struct {
	Data FernetKeys `json:"data"`
}

// GenerateKey generates a base64 url safe fernet key string
func GenerateKey() (string, error) {
	var key fernet.Key
	if err := key.Generate(); err != nil {
		return "", fmt.Errorf("Error generating key: %v", err)
	}
	return key.Encode(), nil
}

// NewFernetKeys creates a new set of fernet keys
func NewFernetKeys(period int64, numKeys int) (*FernetKeys, error) {
	keys := make([]string, numKeys, numKeys)
	for i := 0; i < numKeys; i++ {
		key, err := GenerateKey()
		if err != nil {
			return nil, fmt.Errorf("Error creating new fernet secret: %v", err)
		}
		keys[i] = key
	}
	return &FernetKeys{
		Keys:         keys,
		CreationTime: time.Now().Unix(),
		Period:       period}, nil
}

// CheckFormat checks that a struct FernetKey is coherent
func (fk FernetKeys) CheckFormat() error {
	if fk.Keys == nil {
		return errors.New("Keys list is nil")
	}
	if len(fk.Keys) < 3 {
		return errors.New("Not enough keys")
	}
	if fk.CreationTime == 0 {
		return errors.New("Creation time is nil")
	}
	if fk.Period == 0 {
		return errors.New("Period is nil")
	}
	return nil
}

// Rotate creates a new staging key (Keys[0]), deletes the oldest key in the slice,
// and update the creation time
// If period is 0, keep the same period
func (fk *FernetKeys) Rotate(period int64) error {
	newStaging, err := GenerateKey()
	if err != nil {
		return fmt.Errorf("Error generating new staging key: %v", err)
	}
	newPrimary, keys := fk.Keys[0], fk.Keys[2:]
	keys = append(keys, newPrimary)
	keys = append([]string{newStaging}, keys...)
	fk.Keys = keys
	fk.CreationTime = time.Now().Unix()
	if period > 0 {
		fk.Period = period
	}
	return nil
}

// ReadFernetKeys reads a fernet secret from Vault
func ReadFernetKeys(v vault.Reader, path string) (*FernetKeys, error) {
	var ks KeysSecret
	b, err := v.Read(path)
	if err != nil {
		return nil, fmt.Errorf("Error reading fernet keys secret from vault: %v", err)
	}
	if b == nil {
		return nil, fmt.Errorf("No secret in path %s", path)
	}
	// First decode the JSON into a map[string]interface{}
	if err := json.Unmarshal(b, &ks); err != nil {
		return nil, fmt.Errorf("Error decoding json: %v", err)
	}
	fs := ks.Data

	if err := fs.CheckFormat(); err != nil {
		return nil, fmt.Errorf("Keys have wrong format: %v", err)
	}

	return &fs, nil
}

// WriteFernetKeys writes the fernet keys as a secret in Vault
func WriteFernetKeys(v vault.Writer, path string, fs *FernetKeys, ttl int) error {
	ttlstring := strconv.Itoa(ttl) + "s"
	m := map[string]interface{}{
		"keys":          &fs.Keys,
		"creation_time": &fs.CreationTime,
		"period":        &fs.Period,
		"ttl":           ttlstring}

	if err := v.Write(path, m); err != nil {
		return fmt.Errorf("Error writing keys: %v", err)
	}
	return nil
}

// GetFernetKeys get the fernet keys from a list of vault clients.
// It returns an error if it does not get identical keys.
func GetFernetKeys(vlist []*vault.Vault, path string) (*FernetKeys, error) {
	var fkeysRef *FernetKeys
	for i, v := range vlist {
		vaultName := v.Client.Address()
		fkeys, err := ReadFernetKeys(v, path)
		if err != nil {
			return nil, fmt.Errorf("Cannot get keys from vault %s: %v", vaultName, err)
		}

		if i == 0 {
			fkeysRef = fkeys
		} else if !reflect.DeepEqual(fkeysRef, fkeys) {
			return nil, fmt.Errorf("Doing nothing: keys are not identical in each vaults")
		}
	}
	return fkeysRef, nil
}
