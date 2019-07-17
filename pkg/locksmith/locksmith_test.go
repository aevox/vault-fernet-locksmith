package locksmith

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeVault struct {
}

var (
	fdata = `
		{"data": {
			"keys": [
				"_lCo9aIptB7q5qb8boRVs99FEbFFFOssbDDo6zUYDXU=",
				"dpLsGHWSu23w3uc1CVWLdgeWMNothoBLcYxh4u0V_7Y=",
				"jhPlbcDhWU1GD7UTDp4snD8F9Id2xgowK8hptctENto="],
			"period": 3600,
			"creation_time": 1,
			"ttl": 120}}`

	fkeys = FernetKeys{
		Keys: []string{
			"_lCo9aIptB7q5qb8boRVs99FEbFFFOssbDDo6zUYDXU=",
			"dpLsGHWSu23w3uc1CVWLdgeWMNothoBLcYxh4u0V_7Y=",
			"jhPlbcDhWU1GD7UTDp4snD8F9Id2xgowK8hptctENto="},
		CreationTime: 1,
		Period:       3600}
)

func (v *fakeVault) Read(path string) ([]byte, error) {
	if path == "secret/fernet-keys" {
		return []byte(fdata), nil
	}
	return []byte{}, nil
}

func (v *fakeVault) Write(path string, data map[string]interface{}) error {
	return nil
}

func TestNewFernetKeys(t *testing.T) {
	newKeys, err := NewFernetKeys(3600, 3)
	if err != nil {
		t.Errorf("Error creating new fernet keys: %v", err)
	}
	if newKeys.CheckFormat() != nil {
		t.Errorf("Keys do not have expected format: %v", err)
	}
}

func TestRotate(t *testing.T) {
	assert := assert.New(t)
	keys := fkeys
	if err := keys.Rotate(1800); err != nil {
		t.Errorf("Error rotating keys: %v", err)
	}
	assert.NotEqual(keys.Keys, fkeys.Keys, "Keys expected to change")
	assert.NotEqual(keys.Keys[0], fkeys.Keys[0], "Staging key expected to have changed")
	assert.Equal(keys.Keys[len(keys.Keys)-1], fkeys.Keys[0], "New primary expected to be old staging")
	assert.Equal(keys.Keys[1:len(keys.Keys)-1], fkeys.Keys[2:], "Keys expected to shift")
	assert.Equal(keys.Period, int64(1800), "Period expected to change")
}

func TestReadFernetKeys(t *testing.T) {
	fvault := fakeVault{}
	fkeysRead, err := ReadFernetKeys(&fvault, "secret/fernet-keys")
	if err != nil {
		t.Errorf("Error reading fernet keys: %v", err)
	}
	assert.Equal(t, *fkeysRead, fkeys, "The two structs should be equal")
}
