package cli

import (
	"net/http"
	"strings"
	"testing"
)

func TestTransactions_List_Success(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(mustFixture(t, "transactions_list.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp().Run([]string{
			"freeagent", "transactions", "list",
			"--from-date", "2026-01-01", "--to-date", "2026-01-31",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, want := range []string{"Sale", "Books and Journals", "750-1"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q: %s", want, out)
		}
	}
}

func TestTransactions_List_PassesFilters(t *testing.T) {
	var gotPath, gotQuery string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_, _ = w.Write(mustFixture(t, "transactions_list.json"))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp().Run([]string{
			"freeagent", "transactions", "list",
			"--from-date", "2026-01-01",
			"--to-date", "2026-03-31",
			"--nominal-code", "750-1",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/accounting/transactions" {
		t.Errorf("path = %q", gotPath)
	}
	for _, want := range []string{
		"from_date=2026-01-01",
		"to_date=2026-03-31",
		"nominal_code=750-1",
	} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
}

func TestTransactions_Get_Success(t *testing.T) {
	var gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write(mustFixture(t, "transactions_get.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp().Run([]string{"freeagent", "transactions", "get", "--id", "1"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/accounting/transactions/1" {
		t.Errorf("path = %q", gotPath)
	}
	for _, want := range []string{"Sale", "Bank Account", "750-1"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q: %s", want, out)
		}
	}
}

func TestTransactions_Get_RequiresIdentifier(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp().Run([]string{"freeagent", "transactions", "get"})
	})
	if err == nil || !strings.Contains(err.Error(), "id or url") {
		t.Errorf("expected id-or-url error, got %v", err)
	}
}

func TestTransactions_List_HTTPError(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errors":{"error":{"message":"boom"}}}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp().Run([]string{"freeagent", "transactions", "list"})
	})
	if err == nil {
		t.Fatal("expected error from non-200 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected 500 in error, got %v", err)
	}
}
