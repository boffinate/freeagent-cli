package cli

import "testing"

func TestNewApp_SetsVersion(t *testing.T) {
	app := NewApp("v1.2.3")
	if app.Version != "v1.2.3" {
		t.Fatalf("expected app.Version = %q, got %q", "v1.2.3", app.Version)
	}
}
