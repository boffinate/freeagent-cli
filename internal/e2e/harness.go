//go:build e2e

// Package e2e contains the end-to-end test harness that drives the freeagent
// CLI's HTTP client against a real FreeAgent sandbox account. The package is
// gated behind the `e2e` build tag so it is invisible to the normal `go test
// ./...` run, and every helper file in it carries the same tag.
//
// The harness is opt-in: tests Skip when the four FREEAGENT_E2E_* env vars
// are not set, so a developer without sandbox credentials can still run
// `go test -tags e2e ./internal/e2e/...` and see PASS-with-SKIP. CI without
// secrets will see the same behaviour.
package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/boffinate/freeagent-cli/internal/freeagent"
	"github.com/boffinate/freeagent-cli/internal/storage"
)

// Compile-time guard: singleFileStore must satisfy storage.TokenStore so
// the harness's *freeagent.Client can use it. Catches drift at build time.
var _ storage.TokenStore = (*singleFileStore)(nil)

// Environment-variable names that drive the harness. Kept as exported
// constants so docs and follow-up tests can reference them by symbol.
const (
	EnvTokenFile    = "FREEAGENT_E2E_TOKEN_FILE"
	EnvClientID     = "FREEAGENT_E2E_CLIENT_ID"
	EnvClientSecret = "FREEAGENT_E2E_CLIENT_SECRET"
	EnvBaseURL      = "FREEAGENT_E2E_BASE_URL"

	// allowedE2EHost is the only host the harness will target. Production
	// (api.freeagent.com) is deliberately excluded: the harness creates
	// and deletes resources, and a fat-fingered FREEAGENT_E2E_BASE_URL
	// must not be able to mutate real customer data. Mirrors the readonly
	// binary's hardcoded host allowlist in safemode_readonly.go.
	allowedE2EHost = "api.sandbox.freeagent.com"

	// DefaultBaseURL is used when EnvBaseURL is not set.
	DefaultBaseURL = "https://" + allowedE2EHost + "/v2"

	e2eProfile = "e2e"

	// refreshSkew is the buffer subtracted from the token's ExpiresAt before
	// we decide to refresh proactively. Anything tighter than a minute risks
	// the token expiring mid-test on slow CI; five minutes is comfortable.
	refreshSkew = 5 * time.Minute
)

// Harness bundles everything an e2e test needs: a configured *freeagent.Client
// that will refresh tokens automatically, the resolved base URL for sanity
// asserts, and a Cleanup slice that tests can append created-resource URLs to
// for delete-on-teardown via Harness.RegisterCleanup.
type Harness struct {
	Client  *freeagent.Client
	BaseURL string

	mu      sync.Mutex
	Cleanup []string
}

// bootstrap builds a *Harness from the given configuration without any
// *testing.T plumbing. Used by both NewHarness (per-test) and TestMain
// (suite-level entry/exit sweep). The access token is refreshed eagerly
// if it expires within refreshSkew.
func bootstrap(tokenFile, clientID, clientSecret, baseURL string) (*Harness, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse %s=%q: %w", EnvBaseURL, baseURL, err)
	}
	if parsed.Host != allowedE2EHost {
		return nil, fmt.Errorf("e2e harness refuses non-sandbox host %q (set %s to a sandbox URL such as %s)", parsed.Host, EnvBaseURL, DefaultBaseURL)
	}

	store, err := newSingleFileStore(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("token store: %w", err)
	}

	// HTTP nil → Client.httpClient() picks up the build-tagged
	// defaultHTTPClient(), which under -tags readonly carries the
	// CheckRedirect guard.
	client := &freeagent.Client{
		BaseURL:      baseURL,
		UserAgent:    "freeagent-cli-e2e/0.1",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Profile:      e2eProfile,
		Store:        store,
	}

	// AccessToken() refreshes inside a 1-minute window; we want a wider
	// safety margin so a 14-minute test run doesn't trip mid-flight.
	stored, err := store.Get(e2eProfile)
	if err != nil {
		return nil, fmt.Errorf("read token: %w", err)
	}
	if !stored.ExpiresAt.IsZero() && time.Until(stored.ExpiresAt) < refreshSkew && stored.RefreshToken != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		refreshed, err := client.Refresh(ctx, stored.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("proactive refresh: %w", err)
		}
		if refreshed.RefreshToken == "" {
			refreshed.RefreshToken = stored.RefreshToken
		}
		if err := store.Set(e2eProfile, refreshed); err != nil {
			return nil, fmt.Errorf("persist refreshed token: %w", err)
		}
	}

	return &Harness{Client: client, BaseURL: baseURL}, nil
}

// NewHarness builds a *Harness for the given test. If any required env var
// is missing it calls t.Skip — never t.Fatal — so the suite degrades to a
// no-op for contributors without sandbox credentials.
func NewHarness(t *testing.T) *Harness {
	t.Helper()

	tokenFile := os.Getenv(EnvTokenFile)
	clientID := os.Getenv(EnvClientID)
	clientSecret := os.Getenv(EnvClientSecret)
	if tokenFile == "" || clientID == "" || clientSecret == "" {
		t.Skipf("e2e harness disabled: set %s, %s, %s (and optionally %s) to enable",
			EnvTokenFile, EnvClientID, EnvClientSecret, EnvBaseURL)
		return nil
	}

	h, err := bootstrap(tokenFile, clientID, clientSecret, os.Getenv(EnvBaseURL))
	if err != nil {
		t.Fatalf("e2e bootstrap: %v", err)
	}

	// Per-test cleanup: best-effort delete of any URL the test registered.
	// Leftover resources will be caught by the next Sweep on entry/exit.
	t.Cleanup(func() {
		h.mu.Lock()
		urls := append([]string(nil), h.Cleanup...)
		h.mu.Unlock()
		if len(urls) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		for _, u := range urls {
			if _, status, _, derr := h.Client.Do(ctx, http.MethodDelete, u, nil, ""); derr != nil {
				t.Logf("e2e cleanup: DELETE %s -> %d: %v", u, status, derr)
			}
		}
	})

	return h
}

// RegisterCleanup queues a resource URL for best-effort DELETE on test
// teardown. Safe for concurrent use, though e2e tests run with -parallel 1
// today.
func (h *Harness) RegisterCleanup(url string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Cleanup = append(h.Cleanup, url)
}

// singleFileStore is a TokenStore that reads/writes a single JSON file
// whose path was passed in via FREEAGENT_E2E_TOKEN_FILE — independent of
// the user's regular ~/.config/freeagent/tokens directory. Built on top of
// storage.FileStore by treating the file's basename (sans .json) as the
// profile name and its directory as the store dir, so we reuse the
// existing JSON marshalling without copy-pasting it.
type singleFileStore struct {
	inner   *storage.FileStore
	profile string
}

func newSingleFileStore(path string) (*singleFileStore, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(abs)
	base := filepath.Base(abs)
	const ext = ".json"
	if filepath.Ext(base) != ext {
		return nil, fmt.Errorf("token file must have .json extension: %s", abs)
	}
	profile := base[:len(base)-len(ext)]

	// Verify the file exists and parses as a Token now, rather than
	// failing later on the first Get with a less obvious error.
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("read token file: %w", err)
	}
	var probe storage.Token
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("decode token file %s: %w", abs, err)
	}
	if probe.AccessToken == "" {
		return nil, errors.New("token file is missing access_token")
	}

	return &singleFileStore{
		inner:   &storage.FileStore{Dir: dir},
		profile: profile,
	}, nil
}

func (s *singleFileStore) Get(profile string) (*storage.Token, error) {
	// Ignore caller-supplied profile: this store is bound to one file.
	return s.inner.Get(s.profile)
}

func (s *singleFileStore) Set(profile string, t *storage.Token) error {
	return s.inner.Set(s.profile, t)
}

func (s *singleFileStore) Delete(profile string) error {
	return s.inner.Delete(s.profile)
}
