package cli

import (
	"net/http"
	"strings"
	"testing"
)

func TestBillsList(t *testing.T) {
	var gotPath, gotQuery string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_, _ = w.Write(mustFixture(t, "bills_list.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp().Run([]string{
			"freeagent", "bills", "list",
			"--from", "2026-01-01", "--contact", "1",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/bills" {
		t.Errorf("path = %q", gotPath)
	}
	for _, want := range []string{"from_date=2026-01-01", "contact="} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
	if !strings.Contains(out, "INV-001") {
		t.Errorf("output: %q", out)
	}
}

func TestBillsGet(t *testing.T) {
	var gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write(mustFixture(t, "bills_get.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp().Run([]string{"freeagent", "bills", "get", "--id", "100"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/bills/100" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(out, "INV-001") {
		t.Errorf("output: %q", out)
	}
}

func TestBillsGetRequiresIdentifier(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp().Run([]string{"freeagent", "bills", "get"})
	})
	if err == nil || !strings.Contains(err.Error(), "id or url") {
		t.Errorf("expected id-or-url error, got %v", err)
	}
}
