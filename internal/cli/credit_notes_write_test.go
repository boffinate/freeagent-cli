//go:build !readonly

package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCreditNotesCreate(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{
		"credit_note": map[string]any{
			"contact":   "https://api.sandbox.freeagent.com/v2/contacts/1",
			"reference": "CN-001",
			"dated_on":  "2026-04-01",
			"currency":  "GBP",
		},
	})

	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_, _ = w.Write([]byte(`{"credit_note":{"url":"https://api.sandbox.freeagent.com/v2/credit_notes/4","reference":"CN-001"}}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "credit-notes", "create", "--body", bodyFile})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2/credit_notes" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
	cn, _ := gotBody["credit_note"].(map[string]any)
	if cn["reference"] != "CN-001" {
		t.Errorf("body reference: %#v", cn)
	}
	if !strings.Contains(out, "CN-001") {
		t.Errorf("output: %q", out)
	}
}

func TestCreditNotesCreateRequiresContact(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "credit-notes", "create", "--reference", "X"})
	})
	if err == nil || !strings.Contains(err.Error(), "contact") {
		t.Errorf("expected contact error, got %v", err)
	}
}

func TestCreditNotesSendWithFlags(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{
			"freeagent", "credit-notes", "send",
			"--id", "4",
			"--email-to", "client@example.com",
			"--subject", "Your credit note",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2/credit_notes/4/send_email" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
	cn, _ := gotBody["credit_note"].(map[string]any)
	email, _ := cn["email"].(map[string]any)
	if email["to"] != "client@example.com" || email["subject"] != "Your credit note" {
		t.Errorf("email payload: %#v", email)
	}
}

func TestCreditNotesTransition(t *testing.T) {
	var gotMethod, gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "credit-notes", "transition", "--id", "4", "--name", "mark_as_sent"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/v2/credit_notes/4/transitions/mark_as_sent" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
}

func TestCreditNotesDelete(t *testing.T) {
	var gotMethod, gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "credit-notes", "delete", "--id", "4", "--yes"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/v2/credit_notes/4" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
}
