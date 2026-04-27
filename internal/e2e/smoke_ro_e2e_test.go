//go:build e2e && readonly

package e2e

import (
	"net/http"
	"testing"

	"github.com/anjor/freeagent-cli/internal/freeagent"
)

// TestE2E_Smoke_ReadonlyBuild proves that the e2e package compiles AND
// links against the readonly safemode. Specifically: under -tags readonly
// the freeagent package's enforceReadOnly hook is wired in, so
// constructing a Client and asking it to issue an outright DELETE should
// be refused before any network I/O happens.
//
// We don't depend on env vars here: the readonly guard is a pure check
// against (method, url) and runs before AccessToken() is even consulted.
// That makes this safe to run in CI without sandbox creds.
func TestE2E_Smoke_ReadonlyBuild(t *testing.T) {
	c := &freeagent.Client{
		BaseURL: DefaultBaseURL,
		HTTP:    &http.Client{},
	}

	// DELETE against the sandbox host must be refused by the readonly
	// guard. We never reach the token lookup, so the absence of a Store
	// would NOT trip first; if it did, we'd be looking at a regression
	// in the safemode wiring.
	_, _, _, err := c.Do(t.Context(), http.MethodDelete, "/v2/contacts/123", nil, "")
	if err == nil {
		t.Fatalf("readonly build allowed DELETE; safemode_readonly.go is not linked in")
	}
}
