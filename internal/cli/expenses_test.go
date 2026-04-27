package cli

import (
	"net/http"
	"strings"
	"testing"
)

func TestExpensesList(t *testing.T) {
	var gotPath, gotQuery string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_, _ = w.Write(mustFixture(t, "expenses_list.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{
			"freeagent", "expenses", "list",
			"--user", "7", "--project", "5", "--from", "2026-01-01",
		})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/expenses" {
		t.Errorf("path = %q", gotPath)
	}
	for _, want := range []string{"user=", "project=", "from_date=2026-01-01"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
	if !strings.Contains(out, "Taxi to airport") {
		t.Errorf("output: %q", out)
	}
}
