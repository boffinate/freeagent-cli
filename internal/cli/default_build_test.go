//go:build !readonly

package cli

import "testing"

var expectedFullCommands = map[string][]string{
	"auth":             {"configure", "login", "status", "refresh", "logout"},
	"bank":             {"accounts", "transactions", "explanations", "approve"},
	"bills":            {"list", "get"},
	"categories":       {"list"},
	"company":          {"show"},
	"contacts":         {"list", "search", "get", "create"},
	"credit-notes":     {"list", "get"},
	"estimates":        {"list", "get"},
	"expenses":         {"list", "get"},
	"invoices":         {"list", "get", "delete", "create", "send"},
	"price-list-items": {"list", "get"},
	"projects":         {"list", "get"},
	"reports":          {"balance-sheet", "profit-and-loss", "trial-balance", "cashflow"},
	"stock-items":      {"list", "get"},
	"tasks":            {"list", "get"},
	"timeslips":        {"list", "get"},
	"users":            {"list", "get", "me"},
	"version":          nil,
	"raw":              nil,
}

func TestDefaultBuildRegistersAllCommands(t *testing.T) {
	app := NewApp("")
	got := map[string][]string{}
	for _, cmd := range app.Commands {
		var subs []string
		for _, sub := range cmd.Subcommands {
			subs = append(subs, sub.Name)
		}
		got[cmd.Name] = subs
	}
	for name, wantSubs := range expectedFullCommands {
		gotSubs, ok := got[name]
		if !ok {
			t.Errorf("missing top-level command %q", name)
			continue
		}
		want := map[string]struct{}{}
		for _, s := range wantSubs {
			want[s] = struct{}{}
		}
		have := map[string]struct{}{}
		for _, s := range gotSubs {
			have[s] = struct{}{}
			if _, ok := want[s]; !ok {
				t.Errorf("%q has unexpected subcommand %q", name, s)
			}
		}
		for s := range want {
			if _, ok := have[s]; !ok {
				t.Errorf("%q missing expected subcommand %q", name, s)
			}
		}
	}
	for name := range got {
		if _, ok := expectedFullCommands[name]; !ok {
			t.Errorf("unexpected top-level command %q", name)
		}
	}
}
