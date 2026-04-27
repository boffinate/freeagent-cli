package cli

import (
	"net/http"
	"strings"
	"testing"
)

func TestVatReturnsList(t *testing.T) {
	var gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"vat_returns":[{"period_ends_on":"2026-03-31","status":"Open","url":"https://api.sandbox.freeagent.com/v2/vat_returns/2026-03-31"}]}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "vat-returns", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/vat_returns" {
		t.Errorf("path: %s", gotPath)
	}
	if !strings.Contains(out, "2026-03-31") {
		t.Errorf("output: %q", out)
	}
}

func TestVatReturnsGetByPeriod(t *testing.T) {
	var gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"vat_return":{"period_ends_on":"2026-03-31"}}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "vat-returns", "get", "--period-ends-on", "2026-03-31"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/vat_returns/2026-03-31" {
		t.Errorf("path: %s", gotPath)
	}
}

func TestSalesTaxPeriodsList(t *testing.T) {
	var gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"sales_tax_periods":[{"period_starts_on":"2026-01-01","period_ends_on":"2026-03-31","url":"https://api.sandbox.freeagent.com/v2/sales_tax_periods/9"}]}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "sales-tax-periods", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/sales_tax_periods" {
		t.Errorf("path: %s", gotPath)
	}
	if !strings.Contains(out, "2026-03-31") {
		t.Errorf("output: %q", out)
	}
}

func TestCisBandsList(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"cis_bands":[{"name":"Standard","rate":"20.0","url":"https://api.sandbox.freeagent.com/v2/cis_bands/1"}]}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "cis-bands", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out, "Standard") {
		t.Errorf("output: %q", out)
	}
}

func TestSelfAssessmentReturnsList(t *testing.T) {
	var gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"income_tax_returns":[]}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "self-assessment-returns", "list", "--user", "1"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/users/1/self_assessment_returns" {
		t.Errorf("path: %s", gotPath)
	}
}
