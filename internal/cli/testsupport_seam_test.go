package cli

import (
	"net/http"
	"strings"
	"testing"
)

// TestSeamRoutesThroughTestServer is a smoke test for the phase-0 testability
// seam: it runs an unmodified command (contacts list) and verifies the
// request reached the httptest server with the expected path, method, and
// Authorization header, and that the fixture data showed up in stdout.
func TestSeamRoutesThroughTestServer(t *testing.T) {
	var gotPath, gotMethod, gotAuth, gotHost string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		gotHost = r.Host
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mustFixture(t, "contacts_list.json"))
	})
	installTestHooks(t, srv)

	app := NewApp("")
	out, err := captureStdout(t, func() error {
		return app.Run([]string{"freeagent", "contacts", "list"})
	})
	if err != nil {
		t.Fatalf("app.Run: %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/v2/contacts" {
		t.Errorf("path = %q, want /v2/contacts", gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("auth = %q, want Bearer test-token", gotAuth)
	}
	if gotHost != "api.sandbox.freeagent.com" {
		t.Errorf("host = %q, want api.sandbox.freeagent.com (enforceReadOnly sees this)", gotHost)
	}
	if !strings.Contains(out, "Acme Ltd") {
		t.Errorf("output missing fixture data: %q", out)
	}
}
