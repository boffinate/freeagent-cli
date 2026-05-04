package cli

import (
	"encoding/json"
	"net/http"
	"net/url"
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

func TestAPClientsList_Success(t *testing.T) {
	var gotPath, gotQuery, gotMethod string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"clients":[
			{"name":"Acme Ltd","subdomain":"acme","account_manager":"https://api.freeagent.com/v2/account_managers/123","url":"https://api.freeagent.com/v2/clients/1"},
			{"name":"Globex","subdomain":"globex","account_manager":"https://api.freeagent.com/v2/account_managers/456","url":"https://api.freeagent.com/v2/clients/2"}
		]}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "ap", "clients", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/v2/clients" {
		t.Errorf("path = %q, want /v2/clients", gotPath)
	}
	// Auto-paginate adds per_page=100 by default; nothing else should be in
	// the query when no filter flags were passed.
	if gotQuery != "per_page=100" {
		t.Errorf("query = %q, want %q", gotQuery, "per_page=100")
	}
	for _, want := range []string{"Acme Ltd", "acme", "Globex", "Subdomain", "Account Manager"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, out)
		}
	}
}

func TestAPClientsList_Filters(t *testing.T) {
	var gotQuery string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"clients":[]}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{
			"freeagent", "ap", "clients", "list",
			"--view", "active",
			"--sort", "-created_at",
			"--from-date", "2020-01-01",
			"--to-date", "2021-03-31",
			"--updated-since", "2021-05-22T09:00:00Z",
			"--minimal",
			"--per-page", "500",
			"--page", "2",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	parsed, err := url.ParseQuery(gotQuery)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	expect := map[string]string{
		"view":          "active",
		"sort":          "-created_at",
		"from_date":     "2020-01-01",
		"to_date":       "2021-03-31",
		"updated_since": "2021-05-22T09:00:00Z",
		"minimal_data":  "true",
		"per_page":      "500",
		"page":          "2",
	}
	for k, want := range expect {
		if got := parsed.Get(k); got != want {
			t.Errorf("query %s = %q, want %q (full: %q)", k, got, want, gotQuery)
		}
	}
}

func TestAPClientsList_Empty(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"clients":[]}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "ap", "clients", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out, "No clients found") {
		t.Errorf("expected empty-state message, got %q", out)
	}
}

func TestSubdomainFlag_InjectsHeader(t *testing.T) {
	var gotSubdomain string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotSubdomain = r.Header.Get("X-Subdomain")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"contacts":[]}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "--subdomain", "acme", "contacts", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotSubdomain != "acme" {
		t.Errorf("X-Subdomain = %q, want %q", gotSubdomain, "acme")
	}
}

func TestSubdomainFlag_AliasClient(t *testing.T) {
	var gotSubdomain string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotSubdomain = r.Header.Get("X-Subdomain")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"contacts":[]}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "--client", "globex", "contacts", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotSubdomain != "globex" {
		t.Errorf("X-Subdomain = %q (via --client), want %q", gotSubdomain, "globex")
	}
}

func TestSubdomainFlag_AbsentByDefault(t *testing.T) {
	var hadHeader bool
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, hadHeader = r.Header["X-Subdomain"]
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"contacts":[]}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "contacts", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if hadHeader {
		t.Errorf("X-Subdomain header should not be set when --subdomain is unused")
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
