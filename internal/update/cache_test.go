package update

import (
	"testing"
	"time"
)

// withTempConfigDir redirects os.UserConfigDir to a fresh temp directory by
// setting XDG_CONFIG_HOME (Linux) and HOME (macOS fallback). The tests don't
// need to cover Windows here — the Go runtime handles APPDATA lookup, and the
// CI matrix exercises both POSIX paths.
func withTempConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	return dir
}

func TestCacheRoundTrip(t *testing.T) {
	withTempConfigDir(t)

	now := time.Now().UTC().Truncate(time.Second)
	entry := CacheEntry{
		CheckedAt: now,
		LatestTag: "v0.5.0",
		HTMLURL:   "https://github.com/boffinate/freeagent-cli/releases/tag/v0.5.0",
	}
	if err := SaveCache(entry); err != nil {
		t.Fatalf("SaveCache: %v", err)
	}

	got, fresh := LoadCache(now.Add(time.Minute))
	if !fresh {
		t.Error("expected cache to be fresh just after writing")
	}
	if got.LatestTag != entry.LatestTag {
		t.Errorf("LatestTag = %q, want %q", got.LatestTag, entry.LatestTag)
	}
	if !got.CheckedAt.Equal(entry.CheckedAt) {
		t.Errorf("CheckedAt = %v, want %v", got.CheckedAt, entry.CheckedAt)
	}
	if got.HTMLURL != entry.HTMLURL {
		t.Errorf("HTMLURL = %q, want %q", got.HTMLURL, entry.HTMLURL)
	}
}

func TestCacheExpires(t *testing.T) {
	withTempConfigDir(t)

	written := time.Now().UTC().Add(-25 * time.Hour)
	if err := SaveCache(CacheEntry{CheckedAt: written, LatestTag: "v0.0.1"}); err != nil {
		t.Fatalf("SaveCache: %v", err)
	}

	got, fresh := LoadCache(time.Now().UTC())
	if fresh {
		t.Error("expected cache to be stale after 25h")
	}
	if got.LatestTag != "v0.0.1" {
		t.Error("entry should still be returned even when stale, so callers can show it on error")
	}
}

func TestCacheMissingFile(t *testing.T) {
	withTempConfigDir(t)
	got, fresh := LoadCache(time.Now())
	if fresh {
		t.Error("missing cache file should not be fresh")
	}
	if got.LatestTag != "" {
		t.Errorf("missing cache should yield zero entry, got %+v", got)
	}
}
