//go:build !readonly

package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestTasksCreate(t *testing.T) {
	var gotMethod, gotPath, gotQuery string
	var gotBody map[string]any
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_, _ = w.Write([]byte(`{"task":{"url":"https://api.sandbox.freeagent.com/v2/tasks/9","name":"Design"}}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{
			"freeagent", "tasks", "create",
			"--project", "77",
			"--name", "Design",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2/tasks" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
	if !strings.Contains(gotQuery, "project=") || !strings.Contains(gotQuery, "%2Fprojects%2F77") {
		t.Errorf("query missing project URL: %q", gotQuery)
	}
	task, _ := gotBody["task"].(map[string]any)
	if task["name"] != "Design" {
		t.Errorf("body name: %#v", task)
	}
	if !strings.Contains(out, "Design") {
		t.Errorf("output: %q", out)
	}
}

func TestTasksCreateRequiresProject(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "tasks", "create", "--name", "X"})
	})
	if err == nil || !strings.Contains(err.Error(), "project") {
		t.Errorf("expected project error, got %v", err)
	}
}

func TestTasksCreateRequiresName(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "tasks", "create", "--project", "77"})
	})
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Errorf("expected name-required error, got %v", err)
	}
}

func TestTasksUpdateAndDelete(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{"task": map[string]any{"name": "Renamed"}})

	calls := 0
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch r.Method {
		case http.MethodPut:
			if r.URL.Path != "/v2/tasks/9" {
				t.Errorf("put path: %s", r.URL.Path)
			}
		case http.MethodDelete:
			if r.URL.Path != "/v2/tasks/9" {
				t.Errorf("delete path: %s", r.URL.Path)
			}
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "tasks", "update", "--id", "9", "--body", bodyFile})
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	_, err = captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "tasks", "delete", "--id", "9", "--yes"})
	})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}
