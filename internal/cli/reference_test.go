package cli

import (
	"net/http"
	"strings"
	"testing"
)

func TestCompanyShow(t *testing.T) {
	var gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write(mustFixture(t, "company_show.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "company", "show"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/company" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(out, "Test Ltd") {
		t.Errorf("output: %q", out)
	}
}

func TestUsersList(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(mustFixture(t, "users_list.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "users", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out, "Ada") {
		t.Errorf("output: %q", out)
	}
}

func TestUsersMe(t *testing.T) {
	var gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write(mustFixture(t, "users_me.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "users", "me"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/users/me" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(out, "Ada") {
		t.Errorf("output: %q", out)
	}
}

func TestPriceListItemsList(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(mustFixture(t, "price_list_items_list.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "price-list-items", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	// Columns should pull from `code` and `description`, not the non-existent
	// `item_code` / `item_name` fields.
	if !strings.Contains(out, "A001") || !strings.Contains(out, "Apple") {
		t.Errorf("output missing code/description: %q", out)
	}
}

func TestStockItemsList(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(mustFixture(t, "stock_items_list.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "stock-items", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	// Real API fields: description + stock_on_hand (not item_name / stock_level).
	if !strings.Contains(out, "Apple") || !strings.Contains(out, "10.0") {
		t.Errorf("output missing description/stock_on_hand: %q", out)
	}
}

func TestCategoriesList(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(mustFixture(t, "categories_list.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "categories", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out, "Travel") || !strings.Contains(out, "Sales") {
		t.Errorf("output missing grouped rows: %q", out)
	}
}
