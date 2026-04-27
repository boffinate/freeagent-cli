//go:build !readonly

package cli

import (
	"net/http"
	"reflect"
	"testing"
)

func TestPropertiesCRUD(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{"property": map[string]any{"address": "1 The Street"}})
	calls := []string{}
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"property":{"url":"https://api.sandbox.freeagent.com/v2/properties/1"}}`))
	})
	installTestHooks(t, srv)

	for _, args := range [][]string{
		{"freeagent", "properties", "create", "--body", bodyFile},
		{"freeagent", "properties", "update", "--id", "1", "--body", bodyFile},
		{"freeagent", "properties", "delete", "--id", "1", "--yes"},
	} {
		if _, err := captureStdout(t, func() error { return NewApp("").Run(args) }); err != nil {
			t.Fatalf("%v: %v", args[1:], err)
		}
	}
	want := []string{
		"POST /v2/properties",
		"PUT /v2/properties/1",
		"DELETE /v2/properties/1",
	}
	if !reflect.DeepEqual(calls, want) {
		t.Errorf("calls = %v, want %v", calls, want)
	}
}

func TestCapitalAssetTypesCRUD(t *testing.T) {
	bodyFile := writeTempJSON(t, map[string]any{"capital_asset_type": map[string]any{"name": "Vehicles"}})
	calls := []string{}
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"capital_asset_type":{"url":"https://api.sandbox.freeagent.com/v2/capital_asset_types/2"}}`))
	})
	installTestHooks(t, srv)

	for _, args := range [][]string{
		{"freeagent", "capital-asset-types", "create", "--body", bodyFile},
		{"freeagent", "capital-asset-types", "update", "--id", "2", "--body", bodyFile},
		{"freeagent", "capital-asset-types", "delete", "--id", "2", "--yes"},
	} {
		if _, err := captureStdout(t, func() error { return NewApp("").Run(args) }); err != nil {
			t.Fatalf("%v: %v", args[1:], err)
		}
	}
	want := []string{
		"POST /v2/capital_asset_types",
		"PUT /v2/capital_asset_types/2",
		"DELETE /v2/capital_asset_types/2",
	}
	if !reflect.DeepEqual(calls, want) {
		t.Errorf("calls = %v, want %v", calls, want)
	}
}
