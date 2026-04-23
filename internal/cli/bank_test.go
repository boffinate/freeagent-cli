package cli

import (
	"net/http"
	"strings"
	"testing"
)

func TestBankAccountsList(t *testing.T) {
	var gotPath, gotMethod string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_, _ = w.Write(mustFixture(t, "bank_accounts_list.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp().Run([]string{"freeagent", "bank", "accounts", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/v2/bank_accounts" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
	if !strings.Contains(out, "Current") || !strings.Contains(out, "GBP") {
		t.Errorf("output missing fields: %q", out)
	}
}

func TestBankAccountsListJSON(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(mustFixture(t, "bank_accounts_list.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp().Run([]string{"freeagent", "--json", "bank", "accounts", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out, `"bank_accounts"`) {
		t.Errorf("expected raw JSON passthrough, got %q", out)
	}
}

func TestBankAccountsGet(t *testing.T) {
	var gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write(mustFixture(t, "bank_accounts_get.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp().Run([]string{"freeagent", "bank", "accounts", "get", "--id", "1"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/bank_accounts/1" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(out, "Current") {
		t.Errorf("output missing: %q", out)
	}
}

func TestBankTransactionsListRequiresBankAccount(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp().Run([]string{"freeagent", "bank", "transactions", "list"})
	})
	if err == nil || !strings.Contains(err.Error(), "bank-account") {
		t.Errorf("expected bank-account required error, got %v", err)
	}
}

func TestBankTransactionsList(t *testing.T) {
	var gotQuery string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write(mustFixture(t, "bank_transactions_list.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp().Run([]string{
			"freeagent", "bank", "transactions", "list",
			"--bank-account", "42", "--from", "2026-01-01", "--to", "2026-01-31",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, want := range []string{"from_date=2026-01-01", "to_date=2026-01-31", "bank_account="} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
	if !strings.Contains(out, "Coffee shop") {
		t.Errorf("output: %q", out)
	}
}

func TestBankExplanationsListRequiresBankAccount(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp().Run([]string{"freeagent", "bank", "explanations", "list"})
	})
	if err == nil || !strings.Contains(err.Error(), "bank-account") {
		t.Errorf("expected bank-account required error, got %v", err)
	}
}

func TestBankExplanationsList(t *testing.T) {
	var gotQuery string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write(mustFixture(t, "bank_explanations_list.json"))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp().Run([]string{
			"freeagent", "bank", "explanations", "list",
			"--bank-account", "1",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(gotQuery, "bank_account=") {
		t.Errorf("query %q missing bank_account", gotQuery)
	}
}

func TestBankExplanationsGet(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(mustFixture(t, "bank_explanations_get.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp().Run([]string{"freeagent", "bank", "explanations", "get", "--id", "21"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out, "Coffee shop") {
		t.Errorf("output: %q", out)
	}
}
