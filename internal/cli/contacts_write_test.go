//go:build !readonly

package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestContactsCreateAcceptsOrgOnly(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"contact":{"url":"https://api.sandbox.freeagent.com/v2/contacts/1","organisation_name":"Acme Ltd"}}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "contacts", "create", "--organisation", "Acme Ltd"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestContactsCreateAcceptsFirstAndLastName(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"contact":{"url":"https://api.sandbox.freeagent.com/v2/contacts/2","first_name":"Jane","last_name":"Doe"}}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "contacts", "create", "--first-name", "Jane", "--last-name", "Doe"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestContactsCreateRejectsMissingNameAndOrg(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "contacts", "create", "--email", "x@y.test"})
	})
	if err == nil || !strings.Contains(err.Error(), "organisation_name") {
		t.Errorf("expected name/org disjunction error, got %v", err)
	}
}

func TestContactsCreateRejectsFirstNameWithoutLastName(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "contacts", "create", "--first-name", "Jane"})
	})
	if err == nil || !strings.Contains(err.Error(), "organisation_name") {
		t.Errorf("expected disjunction error when last_name missing, got %v", err)
	}
}

func TestContactsUpdate(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{
		"contact": map[string]any{"organisation_name": "Renamed Co"},
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
		return NewApp("").Run([]string{"freeagent", "contacts", "update", "--id", "1", "--body", bodyFile})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/v2/contacts/1" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
	contact, _ := gotBody["contact"].(map[string]any)
	if contact["organisation_name"] != "Renamed Co" {
		t.Errorf("body: %#v", contact)
	}
}

func TestContactsUpdateRequiresIdentifier(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{"contact": map[string]any{"organisation_name": "X"}})
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "contacts", "update", "--body", bodyFile})
	})
	if err == nil || !strings.Contains(err.Error(), "id or url") {
		t.Errorf("expected id-or-url error, got %v", err)
	}
}

func TestContactsDelete(t *testing.T) {
	var gotMethod, gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "contacts", "delete", "--id", "1", "--yes"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/v2/contacts/1" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
}
