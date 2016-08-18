package server

import (
	"encoding/json"
)

type KeysMap map[string]map[string]interface{}

type persistentKeysFile struct {
	// Function that return the contents of the file
	open func() ([]byte, error)
	// Function to dump contents to the file
	dump func([]byte) error
	// Map from public ssh keys to a list of roles
	keys KeysMap
}

func (pkf *persistentKeysFile) Load() error {
	fileContent, err := pkf.open()
	if err != nil {
		return err
	}

	var keys KeysMap

	if err := json.Unmarshal(fileContent, &keys); err != nil {
		return err
	}

	pkf.keys = keys

	return nil
}

func (pkf *persistentKeysFile) Keys() (KeysMap, error) {
	if pkf.keys == nil {
		err := pkf.Load()
		if err != nil {
			return nil, err
		}
	}
	return pkf.keys, nil
}

func (pkf *persistentKeysFile) Search(sshKey string) (map[string]interface{}, error) {
	if pkf.keys == nil {
		err := pkf.Load()
		if err != nil {
			return nil, err
		}
	}
	return pkf.keys[sshKey], nil
}

func NewPersistentKeysFile(open func() ([]byte, error), dump func([]byte) error) *persistentKeysFile {
	return &persistentKeysFile{open: open, dump: dump}
}
