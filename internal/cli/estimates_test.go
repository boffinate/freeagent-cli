package cli

import (
	"net/http"
	"strings"
	"testing"
)

func TestEstimatesList(t *testing.T) {
	var gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write(mustFixture(t, "estimates_list.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp().Run([]string{"freeagent", "estimates", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/estimates" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(out, "EST-001") {
		t.Errorf("output: %q", out)
	}
}
