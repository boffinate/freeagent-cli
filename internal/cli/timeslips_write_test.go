//go:build !readonly

package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestTimeslipsCreate(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_, _ = w.Write([]byte(`{"timeslip":{"url":"https://api.sandbox.freeagent.com/v2/timeslips/3"}}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{
			"freeagent", "timeslips", "create",
			"--user", "1",
			"--project", "77",
			"--task", "9",
			"--dated-on", "2026-04-01",
			"--hours", "1.5",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2/timeslips" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
	ts, _ := gotBody["timeslip"].(map[string]any)
	if ts["hours"] != "1.5" {
		t.Errorf("hours: %#v", ts)
	}
	if !strings.HasSuffix(ts["task"].(string), "/v2/tasks/9") {
		t.Errorf("task not normalised: %#v", ts)
	}
	if !strings.Contains(out, "/v2/timeslips/3") {
		t.Errorf("output: %q", out)
	}
}

func TestTimeslipsCreateRequiresFields(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "timeslips", "create", "--user", "1"})
	})
	if err == nil || !strings.Contains(err.Error(), "is required") {
		t.Errorf("expected required-field error, got %v", err)
	}
}

func TestTimeslipsTimerStartStop(t *testing.T) {
	calls := []string{}
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "timeslips", "timer-start", "--id", "3"})
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	_, err = captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "timeslips", "timer-stop", "--id", "3"})
	})
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	want := []string{"POST /v2/timeslips/3/timer", "DELETE /v2/timeslips/3/timer"}
	if len(calls) != 2 || calls[0] != want[0] || calls[1] != want[1] {
		t.Errorf("calls = %v, want %v", calls, want)
	}
}

func TestTimeslipsDelete(t *testing.T) {
	var gotMethod, gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "timeslips", "delete", "--id", "3", "--yes"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/v2/timeslips/3" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
}
