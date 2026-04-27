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
		return NewApp("").Run([]string{"freeagent", "reports", "balance-sheet", "--as-at", "2026-03-31"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/accounting/balance_sheet" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(gotQuery, "as_at_date=2026-03-31") {
		t.Errorf("query %q missing as_at_date", gotQuery)
	}
	if !strings.Contains(out, "2026-03-31") {
		t.Errorf("output: %q", out)
	}
}

func TestReportsProfitAndLossPath(t *testing.T) {
	var gotPath, gotQuery string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"profit_and_loss_summary":{}}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{
			"freeagent", "reports", "profit-and-loss",
			"--from", "2026-01-01", "--to", "2026-03-31",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/accounting/profit_and_loss/summary" {
		t.Errorf("path = %q", gotPath)
	}
	for _, want := range []string{"from_date=2026-01-01", "to_date=2026-03-31"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
}

func TestReportsProfitAndLossDefaultsWithNoFilter(t *testing.T) {
	var called bool
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, _ = w.Write([]byte(`{"profit_and_loss_summary":{}}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "reports", "profit-and-loss"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !called {
		t.Fatal("no request reached server; API defaults to current accounting year to date when no filter")
	}
}

func TestReportsProfitAndLossAcceptsAccountingPeriod(t *testing.T) {
	var gotQuery string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"profit_and_loss_summary":{}}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{
			"freeagent", "reports", "profit-and-loss",
			"--accounting-period", "2022/23",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(gotQuery, "accounting_period=2022") {
		t.Errorf("query %q missing accounting_period", gotQuery)
	}
}

func TestReportsTrialBalancePath(t *testing.T) {
	var gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"trial_balance_summary":{}}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "reports", "trial-balance"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/accounting/trial_balance/summary" {
		t.Errorf("path = %q", gotPath)
	}
}

func TestReportsCashflowRequiresDates(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "reports", "cashflow"})
	})
	if err == nil || !strings.Contains(err.Error(), "from") {
		t.Errorf("expected from/to required error, got %v", err)
	}
}

func TestReportsCashflow(t *testing.T) {
	var gotPath, gotQuery string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"cashflow":{}}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{
			"freeagent", "reports", "cashflow",
			"--from", "2026-01-01", "--to", "2026-03-31",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/cashflow" {
		t.Errorf("path = %q", gotPath)
	}
	for _, want := range []string{"from_date=2026-01-01", "to_date=2026-03-31"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
}
