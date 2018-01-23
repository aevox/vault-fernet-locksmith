package locksmith

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/fernet/fernet-go"
	"github.com/golang/glog"

	"github.com/aevox/vault-fernet-locksmith/vault"
)

// FernetKeys represents the fernet keys and their metadata
type FernetKeys struct {
	Keys         []string `json:"keys"`
	CreationTime int64    `json:"creation_time"`
	Period       int64    `json:"period"`
}

// KeysSecret is the represents how fernet keys are stored in vault
type KeysSecret struct {
	Data FernetKeys `json:"data"`
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

// GenerateKey generates a base64 url safe fernet key string
func GenerateKey() (string, error) {
	var key fernet.Key
	if err := key.Generate(); err != nil {
		return "", fmt.Errorf("Error generating key: %v", err)
	}
	return key.Encode(), nil
}

// Rotate creates a new set of ferenet keys. It creates a new staging key
// ( Keys[0] ) and deletes the oldest key in the slice
func (fk *FernetKeys) Rotate() error {
	newStaging, err := GenerateKey()
	if err != nil {
		return fmt.Errorf("Error generating new staging key: %v", err)
	}
	newPrimary, keys := fk.Keys[0], fk.Keys[2:]
	keys = append(keys, newPrimary)
	keys = append([]string{newStaging}, keys...)
	fk.Keys = keys
	fk.CreationTime = time.Now().Unix()
	return nil
}

// ReadFernetKeys reads a fernet secret from vault
func ReadFernetKeys(v *vault.Vault, path string) (*FernetKeys, error) {
	var ks KeysSecret
	b, err := v.Read(path)
	if err != nil {
		return nil, fmt.Errorf("Error reading fernet keys secret from vault: %v", err)
	}
	if b == nil {
		return nil, nil
	}
	// First decode the JSON into a map[string]interface{}
	if err := json.Unmarshal(b, &ks); err != nil {
		return nil, fmt.Errorf("Error decoding json: %v", err)
	}
	fs := ks.Data

	return &fs, nil
}

// CheckKeysIntegrity checks the keys integrity
func CheckKeysIntegrity(fk *FernetKeys) error {
	if fk.Keys == nil {
		return errors.New("Keys list is nil")
	}
	if len(fk.Keys) < 3 {
		return errors.New("Not enoug keys")
	}
	if fk.CreationTime == 0 {
		return errors.New("Creation time is nil")
	}
	if fk.Period == 0 {
		return errors.New("Period is nil")
	}
	return nil
}

// WriteKeys writes the secret in vault
func WriteKeys(v *vault.Vault, fs *FernetKeys, path string, ttl int) error {
	ttlstring := strconv.Itoa(ttl) + "s"
	m := map[string]interface{}{
		"keys":          &fs.Keys,
		"creation_time": &fs.CreationTime,
		"period":        &fs.Period,
		"ttl":           ttlstring}

	if err := v.Write(path, m); err != nil {
		return fmt.Errorf("Error writing keys to vault :%v", err)
	}
	return nil
}

// Smith reads the fernet keys in vault and rotates them when their age is about
// to be equal to the period of rotation.
func Smith(v *vault.Vault, path string, ttl int) error {
	glog.V(1).Info("Reading secret")
	fkeys, err := ReadFernetKeys(v, path)
	if err != nil {
		return fmt.Errorf("Cannot read secret: %v", err)
	}
	if err := CheckKeysIntegrity(fkeys); err != nil {
		return fmt.Errorf("Doing nothing: Keys have wrong format: %v", err)
	}
	glog.V(2).Infof("Keys read: %v", *fkeys)

	if fkeys == nil {
		return errors.New("Doing nothing: No fernet keys in vault")
	}

	if time.Now().Unix() > (fkeys.CreationTime + fkeys.Period - int64(ttl)) {
		glog.Info("Rotating keys")
		if err := fkeys.Rotate(); err != nil {
			return fmt.Errorf("Error rotating keys: %v", err)
		}
		glog.V(2).Infof("new keys: %v", *fkeys)
		if err := WriteKeys(v, fkeys, path, ttl); err != nil {
			return fmt.Errorf("Error writing keys to vault: %v", err)
		}
		glog.Info("Rotation complete")
	}
	return nil
}

// Run executes the Smith function every tt/2
func Run(v *vault.Vault, path string, ttl int) {
	for c := time.Tick(time.Duration(ttl) * time.Second); ; <-c {
		if err := Smith(v, path, ttl); err != nil {
			glog.Error(err)
			continue
		}
	}
}
