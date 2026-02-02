package storage

import (
	"errors"
)

var ErrNotFound = errors.New("token not found")

type TokenStore interface {
	Get(profile string) (*Token, error)
	Set(profile string, token *Token) error
	Delete(profile string) error
}

type Store struct {
	primary  TokenStore
	fallback TokenStore
}

func NewStore(primary TokenStore, fallback TokenStore) *Store {
	return &Store{primary: primary, fallback: fallback}
}

func (s *Store) Get(profile string) (*Token, error) {
	if s.primary != nil {
		token, err := s.primary.Get(profile)
		if err == nil {
			return token, nil
		}
		if !errors.Is(err, ErrNotFound) && s.fallback == nil {
			return nil, err
		}
		if s.fallback == nil {
			return nil, ErrNotFound
		}
		token, err = s.fallback.Get(profile)
		if err == nil {
			_ = s.primary.Set(profile, token)
			return token, nil
		}
		return nil, err
	}
	if s.fallback != nil {
		return s.fallback.Get(profile)
	}
	return nil, ErrNotFound
}

func (s *Store) Set(profile string, token *Token) error {
	if s.primary != nil {
		if err := s.primary.Set(profile, token); err == nil {
			return nil
		}
	}
	if s.fallback != nil {
		return s.fallback.Set(profile, token)
	}
	return nil
}

func (s *Store) Delete(profile string) error {
	var errOut error
	if s.primary != nil {
		if err := s.primary.Delete(profile); err != nil && !errors.Is(err, ErrNotFound) {
			errOut = err
		}
	}
	if s.fallback != nil {
		if err := s.fallback.Delete(profile); err != nil && !errors.Is(err, ErrNotFound) {
			errOut = err
		}
	}
	return errOut
}
