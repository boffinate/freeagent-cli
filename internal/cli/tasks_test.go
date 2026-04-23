package cli

import (
	"net/http"
	"strings"
	"testing"
)

func TestTasksList(t *testing.T) {
	var gotQuery string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write(mustFixture(t, "tasks_list.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp().Run([]string{"freeagent", "tasks", "list", "--project", "5"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(gotQuery, "project=") {
		t.Errorf("query %q missing project= (normalizeResourceURL result)", gotQuery)
	}
	if !strings.Contains(out, "Layout") {
		t.Errorf("output: %q", out)
	}
}
