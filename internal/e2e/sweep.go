//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// fixtureMarker is the literal prefix every e2e-created resource's name
// or reference must start with. Sweep refuses to delete anything whose
// identifying field does not begin with this — a foot-gun guard so a
// misconfigured FREEAGENT_E2E_BASE_URL pointing at production cannot
// nuke real data.
const fixtureMarker = "e2e-"

// Sweep removes leftover e2e-prefixed resources from the sandbox. It is
// best-effort: every API failure is logged via t.Logf and the sweep
// continues. Tests must not call t.Fatal/t.Error from inside Sweep, which
// is why every helper here only logs.
//
// In this scaffolding PR only the contacts list is swept end-to-end; the
// other resource types are stubbed with TODO markers so the follow-up PRs
// have an obvious slot to fill. The list is taken straight from #8:
// invoices, estimates, projects, bank-accounts, tasks, timeslips, bills,
// expenses, credit-notes, price-list-items, stock-items.
func Sweep(t *testing.T, h *Harness) {
	t.Helper()
	if h == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	sweepContacts(ctx, t, h)

	// TODO(#8 follow-up): sweep invoices    (/v2/invoices,           reference field)
	// TODO(#8 follow-up): sweep estimates   (/v2/estimates,          reference field)
	// TODO(#8 follow-up): sweep projects    (/v2/projects,           name field)
	// TODO(#8 follow-up): sweep bank-accounts (/v2/bank_accounts,    name field) — read-only in most sandboxes; gate carefully
	// TODO(#8 follow-up): sweep tasks       (/v2/tasks,              name field; per-project)
	// TODO(#8 follow-up): sweep timeslips   (/v2/timeslips,          comment field; date-window scoped)
	// TODO(#8 follow-up): sweep bills       (/v2/bills,              reference field)
	// TODO(#8 follow-up): sweep expenses    (/v2/expenses,           description field)
	// TODO(#8 follow-up): sweep credit-notes (/v2/credit_notes,      reference field)
	// TODO(#8 follow-up): sweep price-list-items (/v2/price_list_items, item_type/description)
	// TODO(#8 follow-up): sweep stock-items (/v2/stock_items,        item_type/description)
}

// sweepContacts pages through /v2/contacts and DELETEs every contact whose
// organisation_name starts with fixtureMarker. Soft-deletes that the
// FreeAgent API may translate into a hide-rather-than-purge are fine —
// the next sweep will not see them since FreeAgent omits hidden contacts
// from the default listing.
func sweepContacts(ctx context.Context, t *testing.T, h *Harness) {
	t.Helper()
	const path = "/v2/contacts?per_page=100"

	body, status, _, err := h.Client.Do(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		t.Logf("e2e sweep: list contacts: status=%d err=%v", status, err)
		return
	}

	var page struct {
		Contacts []struct {
			URL              string `json:"url"`
			OrganisationName string `json:"organisation_name"`
			FirstName        string `json:"first_name"`
			LastName         string `json:"last_name"`
		} `json:"contacts"`
	}
	if err := json.Unmarshal(body, &page); err != nil {
		t.Logf("e2e sweep: decode contacts: %v", err)
		return
	}

	for _, c := range page.Contacts {
		if !looksLikeFixture(c.OrganisationName, c.FirstName, c.LastName) {
			continue
		}
		if c.URL == "" {
			continue
		}
		if _, dstatus, _, derr := h.Client.Do(ctx, http.MethodDelete, c.URL, nil, ""); derr != nil {
			t.Logf("e2e sweep: DELETE %s -> %d: %v", c.URL, dstatus, derr)
		}
	}
}

// looksLikeFixture returns true if any of the supplied identifying fields
// starts with the e2e- marker. Belt-and-braces: even a single match is
// enough to flag the resource for deletion, but at least one field must
// match — never delete a record whose name does not bear the marker.
func looksLikeFixture(fields ...string) bool {
	for _, f := range fields {
		if strings.HasPrefix(f, fixtureMarker) {
			return true
		}
	}
	return false
}
