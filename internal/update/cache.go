package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// cacheTTL is how long a successful update check is treated as fresh. 24h
// matches GitHub's anonymous rate limit window and avoids hammering the API
// when users invoke `freeagent version --check` from shell prompts.
const cacheTTL = 24 * time.Hour

// CacheEntry is the persisted shape of a single update-check result.
type CacheEntry struct {
	CheckedAt time.Time `json:"checked_at"`
	LatestTag string    `json:"latest_tag"`
	HTMLURL   string    `json:"html_url"`
}

// cachePath returns the on-disk location of the update-check cache, anchored
// under os.UserConfigDir (which honours $XDG_CONFIG_HOME on Linux/macOS).
func cachePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "freeagent", "update-check.json"), nil
}

// LoadCache reads the persisted update-check entry. fresh is true only when
// CheckedAt is within cacheTTL of now. A missing file is not an error: it
// returns a zero entry with fresh=false.
func LoadCache(now time.Time) (entry CacheEntry, fresh bool) {
	path, err := cachePath()
	if err != nil {
		return CacheEntry{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return CacheEntry{}, false
	}
	if err := json.Unmarshal(data, &entry); err != nil {
		return CacheEntry{}, false
	}
	fresh = !entry.CheckedAt.IsZero() && now.Sub(entry.CheckedAt) < cacheTTL
	return entry, fresh
}

// SaveCache writes entry to the cache path with 0600 permissions and creates
// the parent directory if necessary. The cache is best-effort: callers that
// can't persist still get a working update check, just without the 24h
// throttle.
func SaveCache(entry CacheEntry) error {
	path, err := cachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
