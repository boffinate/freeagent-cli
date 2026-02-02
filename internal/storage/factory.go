package storage

import (
	"os"
	"path/filepath"
)

const DefaultServiceName = "freegant"

func DefaultFileDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "freegant", "tokens"), nil
}

func NewDefaultStore() (*Store, error) {
	tokenDir, err := DefaultFileDir()
	if err != nil {
		return nil, err
	}
	primary := &KeychainStore{Service: DefaultServiceName}
	fallback := &FileStore{Dir: tokenDir}
	return NewStore(primary, fallback), nil
}
