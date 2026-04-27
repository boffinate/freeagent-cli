package cli

import (
	"net/http"
	"strings"
	"testing"
)

func TestProjectsList(t *testing.T) {
	var gotPath, gotQuery string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_, _ = w.Write(mustFixture(t, "projects_list.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp("").Run([]string{"freeagent", "projects", "list", "--view", "active"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/projects" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(gotQuery, "view=active") {
		t.Errorf("query %q missing view=active", gotQuery)
	}
	if !strings.Contains(out, "Website redesign") {
		t.Errorf("output: %q", out)
	}
}
