package cli

import (
	"net/http"
	"strings"
	"testing"
)

func TestPayrollList(t *testing.T) {
	var gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"payroll":{"year":"2025","periods":[]}}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "payroll", "list", "--year", "2025"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/payroll/2025" {
		t.Errorf("path: %s", gotPath)
	}
}

func TestPayrollGet(t *testing.T) {
	var gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"payroll":{}}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "payroll", "get", "--year", "2025", "--period", "3"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/payroll/2025/3" {
		t.Errorf("path: %s", gotPath)
	}
}

func TestPayrollProfilesList(t *testing.T) {
	var gotPath, gotQuery string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"payroll_profiles":[]}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "payroll-profiles", "list", "--year", "2025", "--user", "1"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/payroll_profiles/2025" {
		t.Errorf("path: %s", gotPath)
	}
	if !strings.Contains(gotQuery, "%2Fusers%2F1") {
		t.Errorf("query: %q", gotQuery)
	}
}
