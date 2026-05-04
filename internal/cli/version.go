package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/urfave/cli/v2"
	"golang.org/x/mod/semver"

	"github.com/boffinate/freeagent-cli/internal/freeagent"
	"github.com/boffinate/freeagent-cli/internal/update"
)

// updateHTTPClient routes `version --check` through the build-tagged
// readonly-aware client. Without this indirection, update.LatestRelease
// would fall back to http.DefaultClient and a 30x from api.github.com to
// a foreign host would be followed silently under -tags readonly.
var updateHTTPClient = func() *http.Client {
	return freeagent.DefaultHTTPClientWithTransport(http.DefaultTransport)
}

// versionCommand registers the `freeagent version` subcommand. version is the
// build's main.Version (injected via -ldflags); when it isn't a valid semver
// tag (e.g. "dev", "v0.1.0-15-gabcdef-dirty") we skip the network check so
// local/dev builds don't make pointless GitHub calls.
func versionCommand(version string) *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Print version (and optionally check for updates)",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "check",
				Usage: "Check GitHub for a newer release",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Bypass the 24h update-check cache (only with --check)",
			},
		},
		Action: func(c *cli.Context) error {
			return runVersion(c, version)
		},
	}
}

func runVersion(c *cli.Context, version string) error {
	if !c.Bool("check") {
		fmt.Fprintf(os.Stdout, "freeagent version %s\n", version)
		fmt.Fprintf(os.Stdout, "%s/%s\n", runtime.GOOS, runtime.GOARCH)
		return nil
	}

	// Dev / non-tagged builds (anything that doesn't parse as semver) skip the
	// network call entirely. Otherwise we'd compare e.g. "dev" against "v1.0.0"
	// and either confuse the user or look noisy in CI logs.
	if version == "" || version == "dev" || !semver.IsValid(normalizeForSemver(version)) {
		fmt.Fprintf(os.Stdout, "freeagent %s — update check skipped (not a released build)\n", displayVersion(version))
		return nil
	}

	now := time.Now().UTC()
	var latestTag, htmlURL string

	if !c.Bool("force") {
		if entry, fresh := update.LoadCache(now); fresh {
			latestTag = entry.LatestTag
			htmlURL = entry.HTMLURL
		}
	}

	if latestTag == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tag, url, err := update.LatestRelease(ctx, updateHTTPClient())
		if err != nil {
			return reportCheckError(version, err)
		}
		latestTag = tag
		htmlURL = url
		// Best-effort persist; if SaveCache fails the next invocation just
		// re-checks. We don't surface the error to keep the UX clean.
		_ = update.SaveCache(update.CacheEntry{CheckedAt: now, LatestTag: tag, HTMLURL: url})
	}

	current := normalizeForSemver(version)
	latest := normalizeForSemver(latestTag)
	if semver.Compare(latest, current) > 0 {
		fmt.Fprintf(os.Stdout, "freeagent %s — %s available\n%s\n", version, latestTag, htmlURL)
		return nil
	}
	fmt.Fprintf(os.Stdout, "freeagent %s (latest)\n", version)
	return nil
}

// reportCheckError writes the appropriate stderr message for an update-check
// failure and returns a cli.ExitCoder so the process exits 2 without urfave
// also re-printing the error.
func reportCheckError(version string, err error) error {
	var rate *update.ErrRateLimited
	if errors.As(err, &rate) {
		when := "soon"
		if !rate.Reset.IsZero() {
			when = rate.Reset.Format(time.RFC3339)
		}
		fmt.Fprintf(os.Stderr, "freeagent %s — update check rate-limited by GitHub, try again after %s\n", version, when)
		return cli.Exit("", 2)
	}
	fmt.Fprintf(os.Stderr, "freeagent %s — could not check for updates: %v\n", version, err)
	return cli.Exit("", 2)
}

// normalizeForSemver coerces a tag-ish string into the form
// `golang.org/x/mod/semver` expects (leading "v"). The semver package rejects
// anything without the "v" prefix, even though git tags conventionally carry
// it.
func normalizeForSemver(s string) string {
	if s == "" {
		return ""
	}
	if s[0] == 'v' || s[0] == 'V' {
		return s
	}
	return "v" + s
}

// displayVersion returns the version string we want to show to humans. We
// keep the original (possibly non-semver) text so users can see why we
// skipped the check.
func displayVersion(v string) string {
	if v == "" {
		return "(unknown)"
	}
	return v
}
