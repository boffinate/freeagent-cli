package cli

import (
	"net/http"
	"strings"
	"testing"
)

func TestCreditNotesList(t *testing.T) {
	var gotPath string
	srv := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write(mustFixture(t, "credit_notes_list.json"))
	})
	installTestHooks(t, srv)

	out, err := captureStdout(t, func() error {
		return NewApp().Run([]string{"freeagent", "credit-notes", "list"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotPath != "/v2/credit_notes" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(out, "CN-001") {
		t.Errorf("output: %q", out)
	}
}
