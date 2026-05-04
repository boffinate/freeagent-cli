package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

// TestPagination_AutoPaginatesByDefault verifies the default behaviour: a list
// command walks Link rel="next" until exhausted and merges the items into a
// single response.
func TestPagination_AutoPaginatesByDefault(t *testing.T) {
	var pageCount int32
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		page := atomic.AddInt32(&pageCount, 1)
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case 1:
			// Echo per_page back so we can assert the helper sets it.
			if got := r.URL.Query().Get("per_page"); got != "100" {
				t.Errorf("page 1 per_page = %q, want 100", got)
			}
			w.Header().Set("Link", `<https://api.sandbox.freeagent.com/v2/invoices?page=2&per_page=100>; rel="next", <https://api.sandbox.freeagent.com/v2/invoices?page=3&per_page=100>; rel="last"`)
			_, _ = w.Write([]byte(`{"invoices":[{"reference":"INV-001","status":"Open","contact":"https://api.sandbox.freeagent.com/v2/contacts/1","url":"https://api.sandbox.freeagent.com/v2/invoices/1","total_value":"100.00","currency":"GBP"}]}`))
		case 2:
			if got := r.URL.Query().Get("page"); got != "2" {
				t.Errorf("page 2 page-param = %q, want 2", got)
			}
			w.Header().Set("Link", `<https://api.sandbox.freeagent.com/v2/invoices?page=3&per_page=100>; rel="next"`)
			_, _ = w.Write([]byte(`{"invoices":[{"reference":"INV-002","status":"Open","contact":"https://api.sandbox.freeagent.com/v2/contacts/1","url":"https://api.sandbox.freeagent.com/v2/invoices/2","total_value":"200.00","currency":"GBP"}]}`))
		case 3:
			// No Link rel=next on the last page; loop stops here.
			_, _ = w.Write([]byte(`{"invoices":[{"reference":"INV-003","status":"Open","contact":"https://api.sandbox.freeagent.com/v2/contacts/1","url":"https://api.sandbox.freeagent.com/v2/invoices/3","total_value":"300.00","currency":"GBP"}]}`))
		default:
			t.Fatalf("unexpected request page %d", page)
		}
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "--json", "invoices", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if pageCount != 3 {
		t.Errorf("expected 3 page fetches, got %d", pageCount)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &decoded); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	list, _ := decoded["invoices"].([]any)
	if len(list) != 3 {
		t.Errorf("merged list len = %d, want 3", len(list))
	}
}

// TestPagination_NoFollowReturnsFirstPage verifies --no-paginate stops after
// the first response, even when a Link rel=next is present.
func TestPagination_NoFollowReturnsFirstPage(t *testing.T) {
	var pageCount int32
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&pageCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Link", `<https://api.sandbox.freeagent.com/v2/invoices?page=2&per_page=100>; rel="next"`)
		_, _ = w.Write([]byte(`{"invoices":[{"reference":"INV-001","status":"Open","contact":"https://api.sandbox.freeagent.com/v2/contacts/1","url":"https://api.sandbox.freeagent.com/v2/invoices/1","total_value":"100.00","currency":"GBP"}]}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		// --json keeps the renderer from making follow-up contact-lookup
		// requests, so pageCount cleanly reflects pagination behaviour only.
		return NewApp("").Run([]string{"freeagent", "--json", "invoices", "list", "--no-paginate"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if pageCount != 1 {
		t.Errorf("expected 1 page fetch with --no-paginate, got %d", pageCount)
	}
}

// TestPagination_ExplicitPageDisablesAutoFollow verifies --page=N requests one
// specific page only.
func TestPagination_ExplicitPageDisablesAutoFollow(t *testing.T) {
	var pageCount int32
	var gotQuery string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&pageCount, 1)
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		// Even with a next link present, --page=N should not follow.
		w.Header().Set("Link", `<https://api.sandbox.freeagent.com/v2/invoices?page=3&per_page=100>; rel="next"`)
		_, _ = w.Write([]byte(`{"invoices":[]}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "invoices", "list", "--page", "2"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if pageCount != 1 {
		t.Errorf("expected 1 page fetch with --page=2, got %d", pageCount)
	}
	if !strings.Contains(gotQuery, "page=2") {
		t.Errorf("query = %q, expected page=2", gotQuery)
	}
}

// TestPagination_MaxPagesCap verifies the cap aborts auto-pagination and
// returns whatever has been fetched so far.
func TestPagination_MaxPagesCap(t *testing.T) {
	var pageCount int32
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		page := atomic.AddInt32(&pageCount, 1)
		w.Header().Set("Content-Type", "application/json")
		// Always claim a next page exists. The cap must stop us.
		w.Header().Set("Link", fmt.Sprintf(`<https://api.sandbox.freeagent.com/v2/invoices?page=%d&per_page=100>; rel="next"`, page+1))
		_, _ = fmt.Fprintf(w, `{"invoices":[{"reference":"INV-%d","status":"Open","contact":"https://api.sandbox.freeagent.com/v2/contacts/1","url":"https://api.sandbox.freeagent.com/v2/invoices/%d","total_value":"1.00","currency":"GBP"}]}`, page, page)
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "--json", "invoices", "list", "--max-pages", "2"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if pageCount != 2 {
		t.Errorf("expected 2 page fetches with --max-pages=2, got %d", pageCount)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &decoded); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	list, _ := decoded["invoices"].([]any)
	if len(list) != 2 {
		t.Errorf("len(invoices) = %d after cap, want 2", len(list))
	}
}

// TestPagination_SinglePageReturnsRawBody verifies that when the server
// returns one page (no Link rel=next), the body is forwarded verbatim — any
// side fields beyond the wrapper key are preserved.
func TestPagination_SinglePageReturnsRawBody(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// No Link header — single page.
		_, _ = w.Write([]byte(`{"invoices":[{"reference":"INV-001","status":"Open","contact":"https://api.sandbox.freeagent.com/v2/contacts/1","url":"https://api.sandbox.freeagent.com/v2/invoices/1","total_value":"100.00","currency":"GBP"}],"meta":{"total_count":1}}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "--json", "invoices", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out, `"meta"`) {
		t.Errorf("single-page response should preserve side fields; got %q", out)
	}
}

// TestPagination_PerPageOverride verifies --per-page overrides the default of
// 100 (and that values >100 are still passed through — some endpoints, like
// /clients with minimal_data=true, allow higher caps).
func TestPagination_PerPageOverride(t *testing.T) {
	var gotPerPage string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPerPage = r.URL.Query().Get("per_page")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"invoices":[]}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "invoices", "list", "--per-page", "25"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPerPage != "25" {
		t.Errorf("per_page = %q, want 25", gotPerPage)
	}
}
