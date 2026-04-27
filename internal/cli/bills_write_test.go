//go:build !readonly

package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBillsCreateFromBody(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{
		"bill": map[string]any{
			"contact":   "https://api.sandbox.freeagent.com/v2/contacts/1",
			"reference": "REF100",
			"dated_on":  "2026-04-01",
			"due_on":    "2026-05-01",
			"bill_items": []any{
				map[string]any{"description": "Line", "total_value": "100.00"},
			},
		},
	})

	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_, _ = w.Write([]byte(`{"bill":{"url":"https://api.sandbox.freeagent.com/v2/bills/12","reference":"REF100"}}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "bills", "create", "--body", bodyFile})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2/bills" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
	bill, _ := gotBody["bill"].(map[string]any)
	if bill["reference"] != "REF100" {
		t.Errorf("body bill missing reference: %#v", bill)
	}
	if !strings.Contains(out, "REF100") {
		t.Errorf("output: %q", out)
	}
}

func TestBillsCreateFlagsOverrideBody(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{
		"bill": map[string]any{
			"contact":   "https://api.sandbox.freeagent.com/v2/contacts/9",
			"reference": "OLD",
			"dated_on":  "2026-01-01",
			"due_on":    "2026-02-01",
			"bill_items": []any{
				map[string]any{"description": "Line", "total_value": "100.00", "category": "https://api.sandbox.freeagent.com/v2/categories/285"},
			},
		},
	})

	var gotBody map[string]any
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_, _ = w.Write([]byte(`{"bill":{"url":"https://api.sandbox.freeagent.com/v2/bills/12","reference":"NEW"}}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{
			"freeagent", "bills", "create",
			"--body", bodyFile,
			"--reference", "NEW",
			"--dated-on", "2026-04-01",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	bill, _ := gotBody["bill"].(map[string]any)
	if bill["reference"] != "NEW" {
		t.Errorf("reference not overridden: %#v", bill)
	}
	if bill["dated_on"] != "2026-04-01" {
		t.Errorf("dated_on not overridden: %#v", bill)
	}
}

func TestBillsCreateRejectsMissingRequired(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "bills", "create", "--reference", "X"})
	})
	if err == nil || !strings.Contains(err.Error(), "is required") {
		t.Errorf("expected required-field error, got %v", err)
	}
}

func TestBillsCreateRejectsMissingItems(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{
		"bill": map[string]any{
			"contact":   "https://api.sandbox.freeagent.com/v2/contacts/1",
			"reference": "REF",
			"dated_on":  "2026-04-01",
			"due_on":    "2026-05-01",
		},
	})
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "bills", "create", "--body", bodyFile})
	})
	if err == nil || !strings.Contains(err.Error(), "bill_items is required") {
		t.Errorf("expected bill_items error, got %v", err)
	}
}

func TestBillsUpdate(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{
		"bill": map[string]any{"reference": "REF200"},
	})

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
			"freeagent", "bills", "update",
			"--id", "100",
			"--body", bodyFile,
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/v2/bills/100" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
	bill, _ := gotBody["bill"].(map[string]any)
	if bill["reference"] != "REF200" {
		t.Errorf("body bill missing reference: %#v", bill)
	}
}

func TestBillsUpdateRequiresIdentifier(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{"bill": map[string]any{"reference": "X"}})
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "bills", "update", "--body", bodyFile})
	})
	if err == nil || !strings.Contains(err.Error(), "id or url") {
		t.Errorf("expected id-or-url error, got %v", err)
	}
}

func TestBillsDelete(t *testing.T) {
	var gotMethod, gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "bills", "delete", "--id", "100", "--yes"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/v2/bills/100" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
}

func writeTempJSON(t *testing.T, payload any) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "body.json")
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}
