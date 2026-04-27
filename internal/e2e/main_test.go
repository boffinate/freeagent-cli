//go:build e2e

package e2e

import (
	"log"
	"os"
	"testing"
)

// TestMain wraps the e2e test run with an entry sweep and an exit sweep so
// that leftover `e2e-*` resources from a crashed previous run don't pile up
// in the sandbox. Both sweeps are best-effort — failures go to log.Printf.
//
// When the FREEAGENT_E2E_* env vars are not set the wrapper does nothing
// extra: individual tests Skip via NewHarness and the binary exits PASS.
func TestMain(m *testing.M) {
	h := tryBootstrap()
	if h != nil {
		Sweep(log.Printf, h)
	}
	code := m.Run()
	if h != nil {
		Sweep(log.Printf, h)
	}
	os.Exit(code)
}

// tryBootstrap returns a Harness if all FREEAGENT_E2E_* vars are set and
// the token file is usable; nil otherwise. Errors are logged, never fatal —
// TestMain must run regardless so the smoke tests can Skip cleanly.
func tryBootstrap() *Harness {
	tokenFile := os.Getenv(EnvTokenFile)
	clientID := os.Getenv(EnvClientID)
	clientSecret := os.Getenv(EnvClientSecret)
	if tokenFile == "" || clientID == "" || clientSecret == "" {
		return nil
	}
	h, err := bootstrap(tokenFile, clientID, clientSecret, os.Getenv(EnvBaseURL))
	if err != nil {
		log.Printf("e2e: TestMain bootstrap failed (sweep skipped): %v", err)
		return nil
	}
	return h
}
