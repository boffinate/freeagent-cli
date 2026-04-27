//go:build !readonly

package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestProjectsCreateFromBody(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{
		"project": map[string]any{
			"contact":      "https://api.sandbox.freeagent.com/v2/contacts/1",
			"name":         "Site rebuild",
			"status":       "Active",
			"currency":     "GBP",
			"budget_units": "Hours",
		},
	})

	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_, _ = w.Write([]byte(`{"project":{"url":"https://api.sandbox.freeagent.com/v2/projects/77","name":"Site rebuild"}}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "projects", "create", "--body", bodyFile})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2/projects" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
	proj, _ := gotBody["project"].(map[string]any)
	if proj["name"] != "Site rebuild" {
		t.Errorf("body name: %#v", proj)
	}
	if !strings.Contains(out, "Site rebuild") {
		t.Errorf("output: %q", out)
	}
}

func TestProjectsCreateRequiresFields(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "projects", "create", "--name", "X"})
	})
	if err == nil || !strings.Contains(err.Error(), "is required") {
		t.Errorf("expected required-field error, got %v", err)
	}
}

func TestProjectsUpdate(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{"project": map[string]any{"status": "Completed"}})

	var gotMethod, gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "projects", "update", "--id", "77", "--body", bodyFile})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/v2/projects/77" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
}

func TestProjectsDelete(t *testing.T) {
	var gotMethod, gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "projects", "delete", "--id", "77", "--yes"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/v2/projects/77" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
}
