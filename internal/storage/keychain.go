package storage

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

type KeychainStore struct {
	Service string
}

func (s *KeychainStore) Get(profile string) (*Token, error) {
	secret, err := keyring.Get(s.Service, profile)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("keychain get: %w", err)
	}
	var token Token
	if err := json.Unmarshal([]byte(secret), &token); err != nil {
		return nil, fmt.Errorf("keychain decode: %w", err)
	}
	return &token, nil
}

func (s *KeychainStore) Set(profile string, token *Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	if err := keyring.Set(s.Service, profile, string(data)); err != nil {
		return fmt.Errorf("keychain set: %w", err)
	}
	return nil
}

func (s *KeychainStore) Delete(profile string) error {
	if err := keyring.Delete(s.Service, profile); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("keychain delete: %w", err)
	}
	return nil
}
