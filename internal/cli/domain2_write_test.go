//go:build !readonly

package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestNotesCreate(t *testing.T) {
	var gotMethod, gotPath, gotQuery string
	var gotBody map[string]any
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_, _ = w.Write([]byte(`{"note":{"url":"https://api.sandbox.freeagent.com/v2/notes/5"}}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{
			"freeagent", "notes", "create",
			"--contact", "1",
			"--note", "Hello world",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2/notes" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
	if !strings.Contains(gotQuery, "%2Fcontacts%2F1") {
		t.Errorf("query: %q", gotQuery)
	}
	note, _ := gotBody["note"].(map[string]any)
	if note["note"] != "Hello world" {
		t.Errorf("body: %#v", note)
	}
}

func TestAttachmentsDelete(t *testing.T) {
	var gotMethod, gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "attachments", "delete", "--id", "9", "--yes"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/v2/attachments/9" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
}

func TestJournalSetsCRUD(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{"journal_set": map[string]any{"description": "X"}})
	calls := []string{}
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"journal_set":{"url":"https://api.sandbox.freeagent.com/v2/journal_sets/3"}}`))
	})
	installTestHooks(t, srv)

	for _, args := range [][]string{
		{"freeagent", "journal-sets", "create", "--body", bodyFile},
		{"freeagent", "journal-sets", "update", "--id", "3", "--body", bodyFile},
		{"freeagent", "journal-sets", "delete", "--id", "3", "--yes"},
	} {
		if _, err := captureStdout(t, func() error { return NewApp("").Run(args) }); err != nil {
			t.Fatalf("%v: %v", args[1:], err)
		}
	}
	want := []string{
		"POST /v2/journal_sets",
		"PUT /v2/journal_sets/3",
		"DELETE /v2/journal_sets/3",
	}
	if !reflect.DeepEqual(calls, want) {
		t.Errorf("calls = %v, want %v", calls, want)
	}
}

func TestAccountLocksSetAndDelete(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{"account_lock": map[string]any{"locked_until": "2026-12-31"}})
	calls := []string{}
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusOK)
	})
	installTestHooks(t, srv)

	if _, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "account-locks", "set", "--body", bodyFile})
	}); err != nil {
		t.Fatalf("set: %v", err)
	}
	if _, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "account-locks", "delete", "--yes"})
	}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	want := []string{"PUT /v2/account_locks", "DELETE /v2/account_locks"}
	if !reflect.DeepEqual(calls, want) {
		t.Errorf("calls = %v, want %v", calls, want)
	}
}

func TestFinalAccountsReportsTransition(t *testing.T) {
	var gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{
			"freeagent", "final-accounts-reports", "transition",
			"--period-ends-on", "2026-03-31",
			"--name", "mark_as_filed",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/final_accounts_reports/2026-03-31/mark_as_filed" {
		t.Errorf("path: %s", gotPath)
	}
}
