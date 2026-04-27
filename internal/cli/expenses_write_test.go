//go:build !readonly

package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestExpensesCreateFromBody(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{
		"expense": map[string]any{
			"user":        "https://api.sandbox.freeagent.com/v2/users/1",
			"category":    "https://api.sandbox.freeagent.com/v2/categories/285",
			"dated_on":    "2026-04-01",
			"gross_value": "12.50",
			"description": "Lunch",
		},
	})

	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_, _ = w.Write([]byte(`{"expense":{"url":"https://api.sandbox.freeagent.com/v2/expenses/9"}}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "expenses", "create", "--body", bodyFile})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2/expenses" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
	exp, _ := gotBody["expense"].(map[string]any)
	if exp["gross_value"] != "12.50" {
		t.Errorf("body missing gross_value: %#v", exp)
	}
	if !strings.Contains(out, "/v2/expenses/9") {
		t.Errorf("output: %q", out)
	}
}

func TestExpensesCreateFlags(t *testing.T) {
	var gotBody map[string]any
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_, _ = w.Write([]byte(`{"expense":{}}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{
			"freeagent", "expenses", "create",
			"--user", "1",
			"--category", "Mileage",
			"--dated-on", "2026-04-01",
			"--description", "Client visit",
			"--mileage", "42.0",
			"--vehicle-type", "Car",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	exp, _ := gotBody["expense"].(map[string]any)
	if !strings.HasSuffix(exp["user"].(string), "/v2/users/1") {
		t.Errorf("user not normalised: %#v", exp)
	}
	if exp["category"] != "Mileage" {
		t.Errorf("Mileage category not preserved: %#v", exp)
	}
	if exp["mileage"] != "42.0" || exp["vehicle_type"] != "Car" {
		t.Errorf("mileage fields missing: %#v", exp)
	}
}

func TestExpensesCreateMileageRequiresVehicleFields(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{
			"freeagent", "expenses", "create",
			"--user", "1",
			"--category", "Mileage",
			"--dated-on", "2026-04-01",
			"--description", "Client visit",
		})
	})
	if err == nil || !strings.Contains(err.Error(), "mileage") {
		t.Errorf("expected mileage error, got %v", err)
	}
}

func TestExpensesCreateRequiresGrossValueWhenNotMileage(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{
			"freeagent", "expenses", "create",
			"--user", "1",
			"--category", "285",
			"--dated-on", "2026-04-01",
			"--description", "Lunch",
		})
	})
	if err == nil || !strings.Contains(err.Error(), "gross_value") {
		t.Errorf("expected gross_value error, got %v", err)
	}
}

func TestExpensesCreateRequiresFields(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "expenses", "create"})
	})
	if err == nil || !strings.Contains(err.Error(), "is required") {
		t.Errorf("expected required-field error, got %v", err)
	}
}

func TestExpensesUpdate(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{"expense": map[string]any{"description": "updated"}})

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
			"freeagent", "expenses", "update",
			"--id", "9",
			"--body", bodyFile,
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/v2/expenses/9" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
	exp, _ := gotBody["expense"].(map[string]any)
	if exp["description"] != "updated" {
		t.Errorf("body description: %#v", exp)
	}
}

func TestExpensesDelete(t *testing.T) {
	var gotMethod, gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "expenses", "delete", "--id", "9", "--yes"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/v2/expenses/9" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
}
