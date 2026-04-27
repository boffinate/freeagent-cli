//go:build !readonly

package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPriceListItemsCreate(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_, _ = w.Write([]byte(`{"price_list_item":{"url":"https://api.sandbox.freeagent.com/v2/price_list_items/4","code":"WIDGET"}}`))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{
			"freeagent", "price-list-items", "create",
			"--code", "WIDGET",
			"--description", "Widget",
			"--price", "9.99",
			"--quantity", "1",
			"--item-type", "Products",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2/price_list_items" {
		t.Errorf("got %s %s", gotMethod, gotPath)
	}
	item, _ := gotBody["price_list_item"].(map[string]any)
	if item["code"] != "WIDGET" || item["price"] != "9.99" {
		t.Errorf("body: %#v", item)
	}
	if item["quantity"] != "1" || item["item_type"] != "Products" {
		t.Errorf("body missing quantity/item_type: %#v", item)
	}
	if !strings.Contains(out, "WIDGET") {
		t.Errorf("output: %q", out)
	}
}

func TestPriceListItemsCreateRequiresFields(t *testing.T) {
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not hit server")
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "price-list-items", "create"})
	})
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Errorf("expected required-field error, got %v", err)
	}
}

func TestPriceListItemsUpdateAndDelete(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{"price_list_item": map[string]any{"price": "12.00"}})
	calls := 0
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/v2/price_list_items/4" {
			t.Errorf("path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	installTestHooks(t, srv)

	_, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "price-list-items", "update", "--id", "4", "--body", bodyFile})
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	_, err = captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "price-list-items", "delete", "--id", "4", "--yes"})
	})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}
