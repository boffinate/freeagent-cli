//go:build e2e

package e2e

import (
	"os"
	"strings"
	"testing"
)

// TestE2E_Smoke_HarnessSkipsWithoutEnv proves two things at once:
//  1. The e2e package builds under `-tags e2e`.
//  2. NewHarness() invokes t.Skip (rather than t.Fatal) when the
//     FREEAGENT_E2E_* env vars are absent, so contributors without
//     sandbox credentials can still run `go test -tags e2e ./...` and
//     see PASS-with-SKIP rather than red CI.
//
// We cannot assert on Skip directly from the same test that triggers it
// (t.Skip halts execution), so we wrap the call in a subtest and inspect
// the subtest's outcome via the parent. The subtest must Skip; if it
// fails or runs to completion, our skip wiring is broken.
func TestE2E_Smoke_HarnessSkipsWithoutEnv(t *testing.T) {
	// Belt-and-braces: clear the env vars for the duration of this test
	// even if the developer happens to have them exported. We restore
	// in t.Cleanup so the rest of the suite is unaffected.
	for _, k := range []string{EnvTokenFile, EnvClientID, EnvClientSecret, EnvBaseURL} {
		prev, had := os.LookupEnv(k)
		os.Unsetenv(k)
		if had {
			t.Cleanup(func() { os.Setenv(k, prev) })
		}
	}

	// Probe via a subtest. We capture inner so we can interrogate
	// inner.Skipped()/inner.Failed() AFTER the subtest has finished —
	// the *testing.T outlives the closure for read-only inspection.
	//
	// Expected outcome: NewHarness calls inner.Skip(...), which exits
	// the closure immediately, leaves inner.Skipped() == true, and
	// makes t.Run return true (Skip is not a failure). If NewHarness
	// returns normally (regression: env-check missing), the closure
	// records a failure via inner.Errorf and t.Run returns false.
	var inner *testing.T
	t.Run("inner_should_skip", func(it *testing.T) {
		inner = it
		_ = NewHarness(it)
		// Reached only if NewHarness did NOT Skip.
		it.Errorf("NewHarness did not Skip with FREEAGENT_E2E_* unset")
	})
	if inner == nil {
		t.Fatalf("inner subtest never ran")
	}
	if inner.Failed() {
		t.Fatalf("inner subtest failed; expected Skip when FREEAGENT_E2E_* is unset")
	}
	if !inner.Skipped() {
		t.Fatalf("inner subtest did not Skip; NewHarness env-var check is broken")
	}
}

// TestE2E_Smoke_FixturePrefix exercises the FixturePrefix helper without
// needing sandbox credentials. The function is pure, so we can validate
// its contract here and catch shape regressions early.
func TestE2E_Smoke_FixturePrefix(t *testing.T) {
	got := FixturePrefix(t)
	if !strings.HasPrefix(got, "e2e-") {
		t.Fatalf("FixturePrefix() = %q; want e2e- prefix", got)
	}
	if !strings.Contains(got, t.Name()) {
		t.Fatalf("FixturePrefix() = %q; want it to embed the test name %q", got, t.Name())
	}
	if strings.Contains(got, "/") {
		t.Fatalf("FixturePrefix() = %q; must not contain '/' (would break invoice references etc.)", got)
	}
}
