// Package update implements an opt-in check against the GitHub Releases API
// for newer freeagent-cli releases. It is designed to be invoked only by the
// `freeagent version --check` subcommand: there is no passive/background
// polling and no auto-install.
package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// releasesURL is the GitHub Releases API endpoint for the boffinate fork.
// Hard-coded because the readonly safemode allowlists this exact host and
// because we ship pre-built binaries from this repo only.
const releasesURL = "https://api.github.com/repos/boffinate/freeagent-cli/releases/latest"

// defaultTimeout bounds a single update check so an unresponsive GitHub does
// not stall an interactive `version --check`. Applied only when the caller
// hasn't supplied its own context deadline.
const defaultTimeout = 5 * time.Second

// ErrRateLimited is returned by LatestRelease when GitHub responds with 403
// and X-RateLimit-Remaining: 0. Callers can use it to render an actionable
// message including when the limit resets, instead of a generic HTTP error.
type ErrRateLimited struct {
	Reset time.Time
}

func (e *ErrRateLimited) Error() string {
	if e.Reset.IsZero() {
		return "github rate limit exceeded"
	}
	return fmt.Sprintf("github rate limit exceeded (resets at %s)", e.Reset.Format(time.RFC3339))
}

// LatestRelease queries the GitHub Releases API for the latest published
// release and returns its tag and html_url. httpClient may be nil; in that
// case http.DefaultClient is used. If ctx has no deadline, a 5s timeout is
// applied.
func LatestRelease(ctx context.Context, httpClient *http.Client) (tag, htmlURL string, err error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	// GitHub requires a User-Agent on all API requests; without one it returns 403.
	req.Header.Set("User-Agent", "freeagent-cli")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("github request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden && resp.Header.Get("X-RateLimit-Remaining") == "0" {
		reset := parseResetHeader(resp.Header.Get("X-RateLimit-Reset"))
		return "", "", &ErrRateLimited{Reset: reset}
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("github returned status %d", resp.StatusCode)
	}

	var payload struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", "", fmt.Errorf("decode github response: %w", err)
	}
	if payload.TagName == "" {
		return "", "", errors.New("github response missing tag_name")
	}
	return payload.TagName, payload.HTMLURL, nil
}

// parseResetHeader converts X-RateLimit-Reset (unix seconds) into a time.Time.
// Returns the zero value on any parse failure; callers must handle that.
func parseResetHeader(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	secs, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(secs, 0).UTC()
}
