//go:build !readonly

package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestEstimatesCreate(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{
		"estimate": map[string]any{
			"contact":   "https://api.sandbox.freeagent.com/v2/contacts/1",
			"reference": "EST-100",
			"dated_on":  "2026-04-01",
			"currency":  "GBP",
		},
	})
	var gotMethod, gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"estimate":{"url":"https://api.sandbox.freeagent.com/v2/estimates/9","reference":"EST-100"}}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "estimates", "create", "--body", bodyFile})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2/estimates" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
	if !strings.Contains(out, "EST-100") {
		t.Errorf("output: %q", out)
	}
}

func TestEstimatesTransitionAndDuplicate(t *testing.T) {
	calls := []string{}
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"estimate":{"url":"https://api.sandbox.freeagent.com/v2/estimates/10"}}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "estimates", "transition", "--id", "9", "--name", "mark_as_approved"})
	})
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "estimates", "duplicate", "--id", "9"})
	})
	if err != nil {
		t.Fatalf("duplicate: %v", err)
	}
	if !strings.Contains(out, "/v2/estimates/10") {
		t.Errorf("duplicate output: %q", out)
	}
	want := []string{"PUT /v2/estimates/9/transitions/mark_as_approved", "POST /v2/estimates/9/duplicate"}
	if len(calls) != 2 || calls[0] != want[0] || calls[1] != want[1] {
		t.Errorf("calls = %v, want %v", calls, want)
	}
}

func TestEstimatesSendWithBodyFile(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{
		"estimate": map[string]any{"email": map[string]any{"to": "x@y.test"}},
	})
	var gotPath string
	var gotBody map[string]any
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "estimates", "send", "--id", "9", "--body", bodyFile})
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if gotPath != "/v2/estimates/9/send_email" {
		t.Errorf("path: %s", gotPath)
	}
	est, _ := gotBody["estimate"].(map[string]any)
	email, _ := est["email"].(map[string]any)
	if email["to"] != "x@y.test" {
		t.Errorf("body file not used: %#v", email)
	}
}

func TestEstimatesUpdateAndDelete(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{"estimate": map[string]any{"reference": "EST-200"}})
	calls := 0
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "estimates", "update", "--id", "9", "--body", bodyFile})
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	_, err = captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "estimates", "delete", "--id", "9", "--yes"})
	})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}
