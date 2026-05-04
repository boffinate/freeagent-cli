package cli

// Test support for per-command tests in the cli package.
//
// The helpers here swap three package-level var hooks — newTokenStore,
// newHTTPClient, and loadConfigHook — so that app.Run takes the real command
// plumbing (including the readonly enforcement layer) all the way down to an
// httptest server instead of the real FreeAgent API. Because those hooks are
// package-level vars, tests that call installTestHooks MUST NOT call
// t.Parallel(): concurrent mutation of the hooks would race regardless of
// t.Cleanup ordering. If parallel tests become necessary later, guard the
// hook writes with a sync.Mutex and revisit.

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/boffinate/freeagent-cli/internal/config"
	"github.com/boffinate/freeagent-cli/internal/freeagent"
	"github.com/boffinate/freeagent-cli/internal/storage"
)

const testBaseURL = "https://api.sandbox.freeagent.com/v2"

func setupTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// installTestHooks swaps newTokenStore, newHTTPClient, and loadConfigHook so
// that requests driven by app.Run route through srv. Restores all three via
// t.Cleanup. Callers must not run in parallel; see file header.
func installTestHooks(t *testing.T, srv *httptest.Server) {
	t.Helper()

	prevTokenStore := newTokenStore
	prevHTTPClient := newHTTPClient
	prevConfig := loadConfigHook
	t.Cleanup(func() {
		newTokenStore = prevTokenStore
		newHTTPClient = prevHTTPClient
		loadConfigHook = prevConfig
	})

	store := storage.NewMemoryStore("default", storage.Token{
		AccessToken: "test-token",
		ExpiresAt:   time.Now().Add(time.Hour),
	})
	newTokenStore = func(rt Runtime) (storage.TokenStore, error) { return store, nil }

	transport := testTransport(t, srv)
	newHTTPClient = func(rt Runtime) *http.Client {
		return freeagent.DefaultHTTPClientWithTransport(transport)
	}

	loadConfigHook = func(rt Runtime) (*config.Config, string, error) {
		cfg := &config.Config{Profiles: map[string]config.Profile{
			"default": {
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				BaseURL:      testBaseURL,
				UserAgent:    "freeagent-cli-test/0.1",
			},
		}}
		return cfg, "", nil
	}
}

// testTransport builds an *http.Transport whose DialContext always lands on
// srv, and whose TLS config trusts srv's certificate. ServerName is set to
// "example.com" because httptest.NewTLSServer's cert covers that SAN but not
// api.sandbox.freeagent.com — the URL itself still reads as the allowed host
// so enforceReadOnly sees what it would see in production.
func testTransport(t *testing.T, srv *httptest.Server) *http.Transport {
	t.Helper()

	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())

	serverAddr := srv.Listener.Addr().String()

	return &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, network, serverAddr)
		},
		TLSClientConfig: &tls.Config{
			RootCAs:    pool,
			ServerName: "example.com",
		},
	}
}

// captureStdout redirects os.Stdout for the duration of fn, returning both the
// captured bytes and the error fn produced. Useful for asserting on rendered
// tables or JSON passthrough.
func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan []byte, 1)
	go func() {
		data, _ := io.ReadAll(r)
		done <- data
	}()

	fnErr := fn()
	_ = w.Close()
	os.Stdout = original
	data := <-done
	return string(data), fnErr
}

// mustFixture reads a JSON fixture from internal/cli/testdata/<name>.
func mustFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}
