package server

import (
	"encoding/json"
	"errors"
	"fmt"
)

type KeysMap map[string]map[string]interface{}

type persistentKeysFile struct {
	// Function that return the contents of the file
	open func() ([]byte, error)
	// Function to dump contents to the file
	dump func([]byte) error

	userAttr string
	roleAttr string
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

func (pkf *persistentKeysFile) Search(username string) (map[string]interface{}, error) {
	if pkf.keys == nil {
		err := pkf.Load()
		if err != nil {
			return nil, err
		}
	}

	data := map[string]interface{}{
		"username": username,
		"password": "",
	}

	sshPublicKeys := []string{}

	found := false
	for key, userData := range pkf.keys {
		u, _ := userData[pkf.userAttr]
		user, _ := u.(string)
		password, _ := userData["password"]
		passwordHash, _ := password.(string)
		if user == username {
			sshPublicKeys = append(sshPublicKeys, key)
			data["password"] = passwordHash
			found = true
		}
	}
	if found {
		data["sshPublicKeys"] = sshPublicKeys
		return data, nil
	}

	return nil, errors.New(fmt.Sprintf("User %s not found!", username))
}

func (pkf *persistentKeysFile) SearchUser(userData map[string]string) (map[string]interface{}, error) {
	return pkf.Search(userData["username"])
}

func (pkf *persistentKeysFile) Modify(username, sshPublicKey string) error {
	pkf.keys[sshPublicKey] = map[string]interface{}{"username": username}

	keysBytes, _ := json.Marshal(pkf.keys)
	return pkf.dump(keysBytes) // Dump contents of keys
}

func NewPersistentKeysFile(open func() ([]byte, error), dump func([]byte) error, userAttr, roleAttr string) KeysFile {
	return &persistentKeysFile{open: open, dump: dump, userAttr: userAttr, roleAttr: roleAttr}
}
