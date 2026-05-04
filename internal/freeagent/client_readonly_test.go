//go:build readonly

package freeagent

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/boffinate/freeagent-cli/internal/storage"
)

type tripwireStore struct{ t *testing.T }

func (s *tripwireStore) Get(profile string) (*storage.Token, error) {
	s.t.Fatalf("readonly guard must block before token store Get")
	return nil, errors.New("unreachable")
}
func (s *tripwireStore) Set(profile string, tok *storage.Token) error {
	s.t.Fatalf("readonly guard must block before token store Set")
	return errors.New("unreachable")
}
func (s *tripwireStore) Delete(profile string) error {
	s.t.Fatalf("readonly guard must block before token store Delete")
	return errors.New("unreachable")
}

type tripwireReader struct{ t *testing.T }

func (r *tripwireReader) Read(p []byte) (int, error) {
	r.t.Fatalf("readonly guard must block before request body is read")
	return 0, errors.New("unreachable")
}

func TestReadonlyGuardShortCircuitsBeforeSideEffects(t *testing.T) {
	c := &Client{
		BaseURL: "https://api.freeagent.com/v2",
		Profile: "test",
		Store:   &tripwireStore{t: t},
	}
	_, _, _, err := c.Do(context.Background(), http.MethodPost, "/invoices",
		&tripwireReader{t: t}, "application/json")
	if err == nil {
		t.Fatal("expected readonly error, got nil")
	}
	if !strings.Contains(err.Error(), "readonly build") {
		t.Errorf("expected readonly guard error, got: %v", err)
	}
}
