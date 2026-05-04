//go:build readonly

package cli

import (
	"net/http"
	"strings"
	"testing"

	"github.com/boffinate/freeagent-cli/internal/freeagent"
)

// TestReadonlyBlocksCrossHostRedirect exercises the RO CheckRedirect guard by
// making the test server return a 301 to an attacker host. The command must
// fail with the readonly error, proving that commands go through the
// CheckRedirect-carrying http.Client.
func TestReadonlyBlocksCrossHostRedirect(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://attacker.example/hijack")
		w.WriteHeader(http.StatusMovedPermanently)
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "bills", "list"})
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "readonly build") || !strings.Contains(err.Error(), "non-FreeAgent host") {
		t.Errorf("want readonly non-FreeAgent host error, got: %v", err)
	}
}

// TestReadonlyDefaultClientHasCheckRedirect guards against refactors that
// accidentally drop the redirect guard from the base client factory.
func TestReadonlyDefaultClientHasCheckRedirect(t *testing.T) {
	client := freeagent.DefaultHTTPClientWithTransport(nil)
	if client.CheckRedirect == nil {
		t.Fatal("readonly default client missing CheckRedirect hook")
	}
}
