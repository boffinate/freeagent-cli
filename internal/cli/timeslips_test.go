package cli

import (
	"net/http"
	"strings"
	"testing"
)

func TestTimeslipsListRequiresDateFilter(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "timeslips", "list"})
	})
	if err == nil || !strings.Contains(err.Error(), "from") {
		t.Errorf("expected date-filter required error, got %v", err)
	}
}

func TestTimeslipsList(t *testing.T) {
	var gotQuery string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write(mustFixture(t, "timeslips_list.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{
			"freeagent", "timeslips", "list",
			"--from", "2026-01-01", "--to", "2026-01-31", "--user", "7",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, want := range []string{"from_date=2026-01-01", "to_date=2026-01-31", "user="} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
	if !strings.Contains(out, "4.0") {
		t.Errorf("output: %q", out)
	}
}
