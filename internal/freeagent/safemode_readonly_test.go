//go:build readonly

package freeagent

import (
	"net/http"
	"net/url"
	"testing"
)

func TestReadonlyGuardRejectsMutations(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		if err := enforceReadOnly(method, "https://api.freeagent.com/v2/invoices"); err == nil {
			t.Errorf("readonly guard should reject %s on /v2/invoices", method)
		}
	}
}

func TestReadonlyGuardAllowsTokenEndpoint(t *testing.T) {
	urls := []string{
		"https://api.freeagent.com/v2/token_endpoint",
		"https://api.sandbox.freeagent.com/v2/token_endpoint",
	}
	for _, u := range urls {
		if err := enforceReadOnly(http.MethodPost, u); err != nil {
			t.Errorf("readonly guard should allow POST %s: %v", u, err)
		}
	}
}

func TestReadonlyGuardRejectsLookalikeTokenPaths(t *testing.T) {
	urls := []string{
		"https://api.freeagent.com/v2/foo/token_endpoint/bar",
		"https://api.freeagent.com/v2/token_endpoint_extra",
		"https://api.freeagent.com/v2/token_endpoint/something",
		"https://api.freeagent.com/token_endpoint",
	}
	for _, u := range urls {
		if err := enforceReadOnly(http.MethodPost, u); err == nil {
			t.Errorf("readonly guard should reject POST %s (path must match exactly)", u)
		}
	}
}

func TestReadonlyGuardAllowsReads(t *testing.T) {
	target := "https://api.freeagent.com/v2/invoices"
	for _, method := range []string{http.MethodGet, http.MethodHead, ""} {
		if err := enforceReadOnly(method, target); err != nil {
			t.Errorf("readonly guard should allow %q %s: %v", method, target, err)
		}
	}
	if err := enforceReadOnly(http.MethodOptions, target); err == nil {
		t.Error("readonly guard should reject OPTIONS: allow-list is GET/HEAD only")
	}
}

func TestReadonlyGuardRejectsForeignHost(t *testing.T) {
	// GET to a non-FreeAgent host must be refused: otherwise the bearer
	// token would be sent there, enabling exfiltration.
	cases := []struct {
		method string
		url    string
	}{
		{http.MethodGet, "https://attacker.example/v2/invoices"},
		{http.MethodGet, "https://evil.com/anything"},
		{http.MethodGet, "http://localhost:9999/v2/invoices"},
		{http.MethodPost, "https://attacker.example/v2/invoices"},
		// Lookalike domains: endpoint path + path allow-listing is worthless
		// without host allow-listing.
		{http.MethodPost, "https://api.freeagent.com.attacker.example/v2/token_endpoint"},
		{http.MethodPost, "https://freeagent.com/v2/token_endpoint"},
	}
	for _, c := range cases {
		if err := enforceReadOnly(c.method, c.url); err == nil {
			t.Errorf("readonly guard should reject %s %s (foreign host)", c.method, c.url)
		}
	}
}

func TestReadonlyGuardRejectsUnparseableOrEmptyHost(t *testing.T) {
	cases := []string{
		"",
		"not-a-url",
		"/v2/invoices", // relative: empty host
		"https:///v2/invoices",
	}
	for _, u := range cases {
		if err := enforceReadOnly(http.MethodGet, u); err == nil {
			t.Errorf("readonly guard should reject unparseable/empty-host URL %q", u)
		}
	}
}

func TestReadonlyGuardRejectsNonHTTPS(t *testing.T) {
	// Plaintext HTTP must be refused even against an allowed host: otherwise
	// bearer tokens (and, on the token endpoint, client credentials) would
	// traverse the network unencrypted.
	cases := []struct {
		method string
		url    string
	}{
		{http.MethodGet, "http://api.freeagent.com/v2/invoices"},
		{http.MethodGet, "http://api.sandbox.freeagent.com/v2/invoices"},
		{http.MethodPost, "http://api.freeagent.com/v2/token_endpoint"},
		{http.MethodGet, "ftp://api.freeagent.com/v2/invoices"},
	}
	for _, c := range cases {
		if err := enforceReadOnly(c.method, c.url); err == nil {
			t.Errorf("readonly guard should reject non-https %s %s", c.method, c.url)
		}
	}
}

func TestReadonlyGuardRejectsNonPOSTOnTokenEndpoint(t *testing.T) {
	// The token endpoint exception must be scoped to POST only. A DELETE or
	// PUT to /v2/token_endpoint is not a legitimate OAuth flow and must be
	// refused just like any other mutation.
	target := "https://api.freeagent.com/v2/token_endpoint"
	for _, method := range []string{http.MethodPut, http.MethodDelete, http.MethodPatch} {
		if err := enforceReadOnly(method, target); err == nil {
			t.Errorf("readonly guard should reject %s %s (only POST is permitted at the token endpoint)", method, target)
		}
	}
}

func TestReadonlyHTTPClientCheckRedirect(t *testing.T) {
	client := defaultHTTPClient()
	if client.CheckRedirect == nil {
		t.Fatal("readonly http.Client must have CheckRedirect set (to re-enforce the guard on redirect targets)")
	}

	mustURL := func(s string) *url.URL {
		u, err := url.Parse(s)
		if err != nil {
			t.Fatalf("parse %q: %v", s, err)
		}
		return u
	}

	// Redirect targets that must be refused.
	bad := []struct {
		method string
		url    string
	}{
		{http.MethodGet, "http://api.freeagent.com/v2/invoices"},              // downgrade to plaintext
		{http.MethodGet, "https://attacker.example/v2/invoices"},              // foreign host
		{http.MethodGet, "https://api.freeagent.com.attacker.example/foo"},    // lookalike host
		{http.MethodPost, "https://api.freeagent.com/v2/invoices"},            // POST outside token endpoint
		{http.MethodDelete, "https://api.freeagent.com/v2/token_endpoint"},    // wrong method at token endpoint
	}
	for _, c := range bad {
		req := &http.Request{Method: c.method, URL: mustURL(c.url)}
		if err := client.CheckRedirect(req, nil); err == nil {
			t.Errorf("CheckRedirect must refuse %s %s", c.method, c.url)
		}
	}

	// Legitimate same-host https redirects must be permitted (FreeAgent
	// occasionally redirects e.g. resource URLs; we should not break that).
	good := []struct {
		method string
		url    string
	}{
		{http.MethodGet, "https://api.freeagent.com/v2/invoices/1"},
		{http.MethodHead, "https://api.sandbox.freeagent.com/v2/contacts"},
	}
	for _, c := range good {
		req := &http.Request{Method: c.method, URL: mustURL(c.url)}
		if err := client.CheckRedirect(req, nil); err != nil {
			t.Errorf("CheckRedirect must permit %s %s: %v", c.method, c.url, err)
		}
	}

	// Redirect chain length limit.
	via := make([]*http.Request, 10)
	req := &http.Request{Method: http.MethodGet, URL: mustURL("https://api.freeagent.com/v2/x")}
	if err := client.CheckRedirect(req, via); err == nil {
		t.Error("CheckRedirect must stop after 10 redirects")
	}
}
