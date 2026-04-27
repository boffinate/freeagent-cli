//go:build e2e

package e2e

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"
)

// FixturePrefix returns a deterministic-ish unique string the calling test
// can stamp onto every resource it creates. The shape is
// `e2e-<unix-ts>-<short-uuid>-<test-shortname>`, e.g.
// `e2e-1719500000-a1b2c3d4-contacts_create`. The literal `e2e-` prefix is
// load-bearing: Sweep relies on it to find leftover resources.
//
// The test shortname is derived from t.Name() with `/` replaced by `_` so
// subtest names round-trip safely into FreeAgent fields that disallow
// slashes (e.g. invoice references).
func FixturePrefix(t *testing.T) string {
	t.Helper()
	ts := time.Now().Unix()
	short := shortUUID()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	return fmt.Sprintf("e2e-%d-%s-%s", ts, short, name)
}

func shortUUID() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Effectively impossible on a working system, but if it ever
		// happens we still want a non-empty string so the prefix
		// remains parseable. Fall back to the timestamp's nanos.
		return fmt.Sprintf("%08x", time.Now().UnixNano()&0xffffffff)
	}
	return hex.EncodeToString(b[:])
}

// Per-resource helpers (MustCreateContact, MustCreateInvoice, ...) live in
// follow-up PRs once a sandbox account is provisioned and we can exercise
// each endpoint. The signatures are intentionally not stubbed here: an
// uncalled stub that compiles is worse than no stub at all because
// reviewers cannot tell whether a placeholder is intentional. See #8 for
// the planned per-group breakdown.
