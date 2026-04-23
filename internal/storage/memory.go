package storage

import "sync"

type MemoryStore struct {
	mu     sync.Mutex
	tokens map[string]*Token
}

func NewMemoryStore(profile string, token Token) *MemoryStore {
	m := &MemoryStore{tokens: map[string]*Token{}}
	tokenCopy := token
	m.tokens[profile] = &tokenCopy
	return m
}

func (m *MemoryStore) Get(profile string) (*Token, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tok, ok := m.tokens[profile]
	if !ok {
		return nil, ErrNotFound
	}
	copy := *tok
	return &copy, nil
}

func (m *MemoryStore) Set(profile string, token *Token) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if token == nil {
		delete(m.tokens, profile)
		return nil
	}
	tokenCopy := *token
	m.tokens[profile] = &tokenCopy
	return nil
}

func (m *MemoryStore) Delete(profile string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tokens, profile)
	return nil
}
