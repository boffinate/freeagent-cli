package cli

import (
	"net/http"
	"strings"
	"testing"
)

func TestNotesListRequiresFilter(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "notes", "list"})
	})
	if err == nil || !strings.Contains(err.Error(), "contact") {
		t.Errorf("expected contact/project error, got %v", err)
	}
}

func TestNotesListWithContact(t *testing.T) {
	var gotPath, gotQuery string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"notes":[{"created_at":"2026-04-01","note":"Hello","url":"https://api.sandbox.freeagent.com/v2/notes/1"}]}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "notes", "list", "--contact", "1"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/notes" || !strings.Contains(gotQuery, "%2Fcontacts%2F1") {
		t.Errorf("got %s ? %s", gotPath, gotQuery)
	}
	if !strings.Contains(out, "Hello") {
		t.Errorf("output: %q", out)
	}
}

func TestEmailAddressesList(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"email_addresses":[{"email":"a@b","status":"verified","url":"https://api.sandbox.freeagent.com/v2/email_addresses/1"}]}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "email-addresses", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out, "a@b") {
		t.Errorf("output: %q", out)
	}
}

func TestRecurringInvoicesList(t *testing.T) {
	var gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"recurring_invoices":[{"reference":"R-1","status":"Active","url":"https://api.sandbox.freeagent.com/v2/recurring_invoices/1"}]}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "recurring-invoices", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/recurring_invoices" {
		t.Errorf("path: %s", gotPath)
	}
	if !strings.Contains(out, "R-1") {
		t.Errorf("output: %q", out)
	}
}
