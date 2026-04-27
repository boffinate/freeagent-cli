//go:build !readonly

package cli

import "testing"

var expectedFullCommands = map[string][]string{
	"auth":                    {"configure", "login", "status", "refresh", "logout"},
	"ap":                      {"practice"},
	"account-locks":           {"list", "set", "delete"},
	"attachments":             {"get", "delete"},
	"bank":                    {"accounts", "transactions", "explanations", "approve"},
	"bills":                   {"list", "get", "create", "update", "delete"},
	"capital-assets":          {"list", "get"},
	"capital-asset-types":     {"list", "get", "create", "update", "delete"},
	"categories":              {"list"},
	"cis-bands":               {"list"},
	"company":                 {"show"},
	"contacts":                {"list", "search", "get", "create", "update", "delete"},
	"corporation-tax-returns": {"list", "get", "transition"},
	"credit-notes":            {"list", "get", "create", "update", "delete", "send", "transition"},
	"email-addresses":         {"list"},
	"estimates":               {"list", "get", "create", "update", "delete", "send", "transition", "duplicate"},
	"expenses":                {"list", "get", "create", "update", "delete"},
	"final-accounts-reports":  {"list", "get", "transition"},
	"hire-purchases":          {"list", "get"},
	"invoices":                {"list", "get", "delete", "create", "send"},
	"journal-sets":            {"list", "get", "opening-balances", "create", "update", "delete"},
	"notes":                   {"list", "get", "create", "update", "delete"},
	"payroll":                 {"list", "get", "payment-transition"},
	"payroll-profiles":        {"list"},
	"price-list-items":        {"list", "get", "create", "update", "delete"},
	"projects":                {"list", "get", "create", "update", "delete"},
	"properties":              {"list", "get", "create", "update", "delete"},
	"recurring-invoices":      {"list", "get"},
	"reports":                 {"balance-sheet", "profit-and-loss", "trial-balance", "cashflow"},
	"sales-tax-periods":       {"list", "get", "create", "update", "delete"},
	"self-assessment-returns": {"list", "get", "transition", "payment-transition"},
	"stock-items":             {"list", "get"},
	"tasks":                   {"list", "get", "create", "update", "delete"},
	"timeslips":               {"list", "get", "create", "update", "delete", "timer-start", "timer-stop"},
	"transactions":            {"list", "get"},
	"users":                   {"list", "get", "me"},
	"vat-returns":             {"list", "get", "transition", "payment-transition"},
	"version":                 nil,
	"raw":                     nil,
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
