package cli

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestAPPracticeShow_Success(t *testing.T) {
	var gotPath, gotMethod string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"My Practice","subdomain":"mypracticesubdomain"}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "ap", "practice", "show"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/v2/practice" {
		t.Errorf("path = %q, want /v2/practice", gotPath)
	}
	for _, want := range []string{"My Practice", "mypracticesubdomain"} {
		if !strings.Contains(out, want) {
			t.Errorf("output %q missing %q", out, want)
		}
	}
}

func TestAPPracticeShow_HTTPError(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errors":{"error":{"message":"Access denied"}}}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "ap", "practice", "show"})
	})
	if err == nil {
		t.Fatalf("expected error from 401, got nil")
	}
}

func TestAPPracticeShow_JSON(t *testing.T) {
	body := `{"name":"My Practice","subdomain":"mypracticesubdomain"}`
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "--json", "ap", "practice", "show"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	// Must be valid JSON matching the upstream payload, not the table output.
	var decoded map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %q", err, out)
	}
	if decoded["name"] != "My Practice" || decoded["subdomain"] != "mypracticesubdomain" {
		t.Errorf("unexpected JSON payload: %v", decoded)
	}
	if strings.Contains(out, "Name:") {
		t.Errorf("--json mode should not render table; got %q", out)
	}
}

func TestAPAccountManagersList_Success(t *testing.T) {
	var gotPath, gotMethod string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"account_managers":[
			{"url":"https://api.freeagent.com/v2/account_managers/123","name":"Bobson Dugnutt","email":"bobson@example.com"},
			{"url":"https://api.freeagent.com/v2/account_managers/456","name":"Sleve McDichael","email":"sleve@example.com"}
		]}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "ap", "account-managers", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/v2/account_managers" {
		t.Errorf("path = %q, want /v2/account_managers", gotPath)
	}
	for _, want := range []string{"Bobson Dugnutt", "bobson@example.com", "Sleve McDichael", "Name", "Email", "URL"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, out)
		}
	}
}

func TestAPAccountManagersList_Empty(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"account_managers":[]}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "ap", "account-managers", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out, "No account managers found") {
		t.Errorf("expected empty-state message, got %q", out)
	}
}

func TestAPAccountManagersShow_ByID(t *testing.T) {
	var gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"account_manager":{"url":"https://api.sandbox.freeagent.com/v2/account_managers/123","name":"Bobson Dugnutt","email":"bobson@example.com"}}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "ap", "account-managers", "show", "--id", "123"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/account_managers/123" {
		t.Errorf("path = %q, want /v2/account_managers/123", gotPath)
	}
	if !strings.Contains(out, "Bobson Dugnutt") || !strings.Contains(out, "bobson@example.com") {
		t.Errorf("output missing expected fields:\n%s", out)
	}
}

func TestAPAccountManagersShow_ByURL(t *testing.T) {
	var gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"account_manager":{"url":"https://api.sandbox.freeagent.com/v2/account_managers/456","name":"Sleve McDichael","email":"sleve@example.com"}}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "ap", "account-managers", "show", "--url", "https://api.sandbox.freeagent.com/v2/account_managers/456"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/account_managers/456" {
		t.Errorf("path = %q, want /v2/account_managers/456", gotPath)
	}
}

func TestAPAccountManagersShow_RequiresIDOrURL(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be hit when neither --id nor --url is set")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "ap", "account-managers", "show"})
	})
	if err == nil {
		t.Fatalf("expected error for missing --id/--url, got nil")
	}
}
