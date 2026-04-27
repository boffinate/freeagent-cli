package cli

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"testing"

	urfavecli "github.com/urfave/cli/v2"
)

// versionRoundTripFunc is a tiny RoundTripper for stubbing GitHub responses
// in version_test. Re-declared here to keep this file self-contained — the
// cli package's transports (testTransport in testsupport_test.go) speak HTTPS
// against an httptest TLS server, but we want plain unit-level control here.
type versionRoundTripFunc func(req *http.Request) (*http.Response, error)

func (f versionRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// withTempConfigDir mirrors the helper in update/cache_test.go so cache reads
// land in a per-test temp directory.
func withTempConfigDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
}

// stubVersionTransport swaps updateHTTPClient with one that uses fn. Restores
// the previous value via t.Cleanup.
func stubVersionTransport(t *testing.T, fn versionRoundTripFunc) {
	t.Helper()
	prev := updateHTTPClient
	t.Cleanup(func() { updateHTTPClient = prev })
	updateHTTPClient = func() *http.Client {
		return &http.Client{Transport: fn}
	}
}

// suppressOSExit overrides urfave-cli's OsExiter so cli.Exit() returned from
// our action does not actually call os.Exit during tests.
func suppressOSExit(t *testing.T) {
	t.Helper()
	prev := urfavecli.OsExiter
	t.Cleanup(func() { urfavecli.OsExiter = prev })
	urfavecli.OsExiter = func(int) {}
}

// captureBoth captures stdout AND stderr for the duration of fn.
func captureBoth(t *testing.T, fn func() error) (stdout, stderr string, err error) {
	t.Helper()

	origOut, origErr := os.Stdout, os.Stderr
	rOut, wOut, perr := os.Pipe()
	if perr != nil {
		t.Fatalf("pipe stdout: %v", perr)
	}
	rErr, wErr, perr := os.Pipe()
	if perr != nil {
		t.Fatalf("pipe stderr: %v", perr)
	}
	os.Stdout = wOut
	os.Stderr = wErr

	outCh := make(chan []byte, 1)
	errCh := make(chan []byte, 1)
	go func() { b, _ := io.ReadAll(rOut); outCh <- b }()
	go func() { b, _ := io.ReadAll(rErr); errCh <- b }()

	fnErr := fn()
	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = origOut
	os.Stderr = origErr

	return string(<-outCh), string(<-errCh), fnErr
}

func TestVersionDefault(t *testing.T) {
	stdout, _, err := captureBoth(t, func() error {
		return NewApp("v1.2.3").Run([]string{"freeagent", "version"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	wantLine1 := "freeagent version v1.2.3"
	wantLine2 := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	if !strings.Contains(stdout, wantLine1) {
		t.Errorf("stdout missing %q\ngot:\n%s", wantLine1, stdout)
	}
	if !strings.Contains(stdout, wantLine2) {
		t.Errorf("stdout missing %q\ngot:\n%s", wantLine2, stdout)
	}
}

func TestVersionCheckSkipsForDevBuild(t *testing.T) {
	withTempConfigDir(t)
	called := false
	stubVersionTransport(t, func(req *http.Request) (*http.Response, error) {
		called = true
		return nil, errors.New("must not call")
	})

	stdout, _, err := captureBoth(t, func() error {
		return NewApp("dev").Run([]string{"freeagent", "version", "--check"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if called {
		t.Error("dev build must not call GitHub")
	}
	if !strings.Contains(stdout, "update check skipped") {
		t.Errorf("expected skipped message; got: %s", stdout)
	}
}

func TestVersionCheckSkipsForGitDescribeBuild(t *testing.T) {
	withTempConfigDir(t)
	called := false
	stubVersionTransport(t, func(req *http.Request) (*http.Response, error) {
		called = true
		return nil, errors.New("must not call")
	})

	// Typical local-build version string from `git describe --tags --always`
	// when there is no tag at all: a bare commit SHA. Not valid semver, so we
	// must skip the network call.
	_, _, err := captureBoth(t, func() error {
		return NewApp("a42f454").Run([]string{"freeagent", "version", "--check"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if called {
		t.Error("non-semver build must not call GitHub")
	}
}

func TestVersionCheckNewerAvailable(t *testing.T) {
	withTempConfigDir(t)
	stubVersionTransport(t, func(req *http.Request) (*http.Response, error) {
		body := `{"tag_name":"v2.0.0","html_url":"https://github.com/boffinate/freeagent-cli/releases/tag/v2.0.0"}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{},
		}, nil
	})

	stdout, _, err := captureBoth(t, func() error {
		return NewApp("v1.0.0").Run([]string{"freeagent", "version", "--check"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(stdout, "v2.0.0 available") {
		t.Errorf("expected newer-available message; got: %s", stdout)
	}
	if !strings.Contains(stdout, "https://github.com/boffinate/freeagent-cli/releases/tag/v2.0.0") {
		t.Errorf("expected html_url in output; got: %s", stdout)
	}
}

func TestVersionCheckUpToDate(t *testing.T) {
	withTempConfigDir(t)
	stubVersionTransport(t, func(req *http.Request) (*http.Response, error) {
		body := `{"tag_name":"v1.0.0","html_url":"https://x"}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{},
		}, nil
	})

	stdout, _, err := captureBoth(t, func() error {
		return NewApp("v1.0.0").Run([]string{"freeagent", "version", "--check"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(stdout, "(latest)") {
		t.Errorf("expected (latest); got: %s", stdout)
	}
}

func TestVersionCheckRateLimited(t *testing.T) {
	withTempConfigDir(t)
	suppressOSExit(t)
	stubVersionTransport(t, func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 403,
			Body:       io.NopCloser(strings.NewReader("rate limit")),
			Header: http.Header{
				"X-Ratelimit-Remaining": []string{"0"},
				"X-Ratelimit-Reset":     []string{"1700000000"},
			},
		}, nil
	})

	_, stderr, err := captureBoth(t, func() error {
		return NewApp("v1.0.0").Run([]string{"freeagent", "version", "--check"})
	})

	if err == nil {
		t.Fatal("expected error exit")
	}
	if !strings.Contains(stderr, "rate-limited") {
		t.Errorf("expected rate-limited stderr; got: %q", stderr)
	}

	// Verify exit code is 2.
	if ec, ok := err.(urfavecli.ExitCoder); !ok || ec.ExitCode() != 2 {
		t.Errorf("want ExitCoder with code 2, got %T %v", err, err)
	}
}

func TestVersionCheckNetworkError(t *testing.T) {
	withTempConfigDir(t)
	suppressOSExit(t)
	stubVersionTransport(t, func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("dns failure")
	})

	_, stderr, err := captureBoth(t, func() error {
		return NewApp("v1.0.0").Run([]string{"freeagent", "version", "--check"})
	})
	if err == nil {
		t.Fatal("expected error exit")
	}
	if !strings.Contains(stderr, "could not check for updates") {
		t.Errorf("expected generic error message; got: %q", stderr)
	}
	if ec, ok := err.(urfavecli.ExitCoder); !ok || ec.ExitCode() != 2 {
		t.Errorf("want ExitCoder with code 2, got %T %v", err, err)
	}
}

func TestVersionCheckUsesCacheUnlessForce(t *testing.T) {
	withTempConfigDir(t)

	calls := 0
	stubVersionTransport(t, func(req *http.Request) (*http.Response, error) {
		calls++
		body := `{"tag_name":"v1.0.0","html_url":"https://x"}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{},
		}, nil
	})

	// First call — populates cache.
	if _, _, err := captureBoth(t, func() error {
		return NewApp("v1.0.0").Run([]string{"freeagent", "version", "--check"})
	}); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if calls != 1 {
		t.Fatalf("first call should hit network once, got %d", calls)
	}

	// Second call without --force should hit cache.
	if _, _, err := captureBoth(t, func() error {
		return NewApp("v1.0.0").Run([]string{"freeagent", "version", "--check"})
	}); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if calls != 1 {
		t.Errorf("second call should reuse cache, calls=%d", calls)
	}

	// Third call with --force should bypass cache.
	if _, _, err := captureBoth(t, func() error {
		return NewApp("v1.0.0").Run([]string{"freeagent", "version", "--check", "--force"})
	}); err != nil {
		t.Fatalf("third run: %v", err)
	}
	if calls != 2 {
		t.Errorf("--force should bypass cache, calls=%d", calls)
	}
}
