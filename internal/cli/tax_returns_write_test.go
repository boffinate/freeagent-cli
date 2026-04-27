//go:build !readonly

package cli

import (
	"net/http"
	"reflect"
	"testing"
)

func TestVatReturnsTransition(t *testing.T) {
	var gotMethod, gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{
			"freeagent", "vat-returns", "transition",
			"--period-ends-on", "2026-03-31",
			"--name", "mark_as_filed",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/v2/vat_returns/2026-03-31/mark_as_filed" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
}

func TestVatReturnsPaymentTransition(t *testing.T) {
	var gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{
			"freeagent", "vat-returns", "payment-transition",
			"--period-ends-on", "2026-03-31",
			"--payment-date", "2026-04-15",
			"--name", "mark_as_paid",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/vat_returns/2026-03-31/payments/2026-04-15/mark_as_paid" {
		t.Errorf("path: %s", gotPath)
	}
}

func TestSelfAssessmentReturnsTransition(t *testing.T) {
	var gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{
			"freeagent", "self-assessment-returns", "transition",
			"--user", "1",
			"--period-ends-on", "2026-04-05",
			"--name", "mark_as_filed",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/users/1/self_assessment_returns/2026-04-05/mark_as_filed" {
		t.Errorf("path: %s", gotPath)
	}
}

func TestSalesTaxPeriodsCRUD(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{
		"sales_tax_period": map[string]any{"period_starts_on": "2026-01-01", "period_ends_on": "2026-03-31"},
	})

	calls := []string{}
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"sales_tax_period":{"url":"https://api.sandbox.freeagent.com/v2/sales_tax_periods/9"}}`))
	})
	installTestHooks(t, srv)

	for _, args := range [][]string{
		{"freeagent", "sales-tax-periods", "create", "--body", bodyFile},
		{"freeagent", "sales-tax-periods", "update", "--id", "9", "--body", bodyFile},
		{"freeagent", "sales-tax-periods", "delete", "--id", "9", "--yes"},
	} {
		if _, err := captureStdout(t, func() error { return NewApp("").Run(args) }); err != nil {
			t.Fatalf("%v: %v", args[1:], err)
		}
	}
	want := []string{
		"POST /v2/sales_tax_periods",
		"PUT /v2/sales_tax_periods/9",
		"DELETE /v2/sales_tax_periods/9",
	}
	if !reflect.DeepEqual(calls, want) {
		t.Errorf("calls = %v, want %v", calls, want)
	}
}
