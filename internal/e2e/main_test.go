//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/anjor/freeagent-cli/internal/freeagent"
	"github.com/anjor/freeagent-cli/internal/storage"
)

// TestMain wraps the e2e test run with an entry sweep and an exit sweep so
// that leftover `e2e-*` resources from a crashed previous run don't pile
// up in the sandbox. Both sweeps are best-effort: we log failures but
// never block the test binary.
//
// When the FREEAGENT_E2E_* env vars are not set the wrapper does nothing
// extra — individual tests will Skip via NewHarness, and the binary still
// exits cleanly with PASS.
func TestMain(m *testing.M) {
	if envConfigured() {
		if h, err := bootstrapForSweep(); err != nil {
			log.Printf("e2e: entry sweep skipped: %v", err)
		} else {
			runStandaloneSweep("entry", h)
			defer runStandaloneSweep("exit", h)
		}
	}
	os.Exit(m.Run())
}

// envConfigured returns true iff every required FREEAGENT_E2E_* variable
// is set. We deliberately do NOT default any of them here: the harness
// must be opt-in.
func envConfigured() bool {
	return os.Getenv(EnvTokenFile) != "" &&
		os.Getenv(EnvClientID) != "" &&
		os.Getenv(EnvClientSecret) != ""
}

// bootstrapForSweep mirrors NewHarness but without a *testing.T (TestMain
// runs before any test exists). Errors are returned rather than logged
// here so the caller can decide whether to abort the sweep — usually we
// just log and carry on.
func bootstrapForSweep() (*Harness, error) {
	tokenFile := os.Getenv(EnvTokenFile)
	clientID := os.Getenv(EnvClientID)
	clientSecret := os.Getenv(EnvClientSecret)
	baseURL := os.Getenv(EnvBaseURL)
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	store, err := newSingleFileStore(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("token store: %w", err)
	}
	client := &freeagent.Client{
		BaseURL:      baseURL,
		UserAgent:    "freeagent-cli-e2e/0.1",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Profile:      e2eProfile,
		Store:        store,
		HTTP:         &http.Client{Timeout: 30 * time.Second},
	}

	// Probe + proactive refresh, same logic as NewHarness.
	stored, err := store.Get(e2eProfile)
	if err != nil {
		return nil, fmt.Errorf("read token: %w", err)
	}
	if !stored.ExpiresAt.IsZero() && time.Until(stored.ExpiresAt) < refreshSkew && stored.RefreshToken != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		refreshed, err := client.Refresh(ctx, stored.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("refresh: %w", err)
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

// runStandaloneSweep invokes the same sweep logic as Sweep, but without a
// *testing.T (so failures go to log.Printf instead of t.Logf). Kept
// separate so the per-test path stays pure and testable.
func runStandaloneSweep(label string, h *Harness) {
	if h == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	body, status, _, err := h.Client.Do(ctx, http.MethodGet, "/v2/contacts?per_page=100", nil, "")
	if err != nil {
		log.Printf("e2e %s sweep: list contacts: status=%d err=%v", label, status, err)
		return
	}
	var page struct {
		Contacts []struct {
			URL              string `json:"url"`
			OrganisationName string `json:"organisation_name"`
			FirstName        string `json:"first_name"`
			LastName         string `json:"last_name"`
		} `json:"contacts"`
	}
	if err := json.Unmarshal(body, &page); err != nil {
		log.Printf("e2e %s sweep: decode contacts: %v", label, err)
		return
	}
	deleted := 0
	for _, c := range page.Contacts {
		if !(strings.HasPrefix(c.OrganisationName, fixtureMarker) ||
			strings.HasPrefix(c.FirstName, fixtureMarker) ||
			strings.HasPrefix(c.LastName, fixtureMarker)) {
			continue
		}
		if c.URL == "" {
			continue
		}
		if _, dstatus, _, derr := h.Client.Do(ctx, http.MethodDelete, c.URL, nil, ""); derr != nil {
			log.Printf("e2e %s sweep: DELETE %s -> %d: %v", label, c.URL, dstatus, derr)
			continue
		}
		deleted++
	}
	if deleted > 0 {
		log.Printf("e2e %s sweep: removed %d e2e-prefixed contact(s)", label, deleted)
	}
}

// Compile-time guard: the harness's singleFileStore must satisfy the
// storage.TokenStore interface that *freeagent.Client.Store demands. If
// storage.TokenStore drifts, this fails at build time rather than at the
// first sandbox call.
var _ storage.TokenStore = (*singleFileStore)(nil)
