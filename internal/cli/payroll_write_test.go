//go:build !readonly

package cli

import (
	"net/http"
	"testing"
)

func TestPayrollPaymentTransition(t *testing.T) {
	var gotMethod, gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{
			"freeagent", "payroll", "payment-transition",
			"--year", "2025",
			"--payment-date", "2026-04-15",
			"--name", "mark_as_paid",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/v2/payroll/2025/payments/2026-04-15/mark_as_paid" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
}
