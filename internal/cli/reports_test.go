package cli

import (
	"net/http"
	"strings"
	"testing"
)

func TestReportsBalanceSheet(t *testing.T) {
	var gotPath, gotQuery string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_, _ = w.Write(mustFixture(t, "balance_sheet.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp().Run([]string{"freeagent", "reports", "balance-sheet", "--as-at", "2026-03-31"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/balance_sheet" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(gotQuery, "as_at_date=2026-03-31") {
		t.Errorf("query %q missing as_at_date", gotQuery)
	}
	if !strings.Contains(out, "2026-03-31") {
		t.Errorf("output: %q", out)
	}
}

func TestReportsProfitAndLossRequiresFilter(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp().Run([]string{"freeagent", "reports", "profit-and-loss"})
	})
	if err == nil || !strings.Contains(err.Error(), "accounting-year") {
		t.Errorf("expected filter required error, got %v", err)
	}
}

func TestReportsProfitAndLossFromTo(t *testing.T) {
	var gotPath, gotQuery string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"profit_and_loss": []}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp().Run([]string{
			"freeagent", "reports", "profit-and-loss",
			"--from", "2026-01-01", "--to", "2026-03-31",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/profit_and_loss" {
		t.Errorf("path = %q", gotPath)
	}
	for _, want := range []string{"from_date=2026-01-01", "to_date=2026-03-31"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
}
