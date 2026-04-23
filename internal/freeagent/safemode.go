//go:build !readonly

package freeagent

import "net/http"

func enforceReadOnly(method, urlStr string) error { return nil }

func defaultHTTPClient() *http.Client {
	return &http.Client{Timeout: timeoutDefault}
}

// DefaultHTTPClientWithTransport returns a client built from defaultHTTPClient
// with Transport replaced. The full build has no redirect policy to preserve,
// so only Transport needs swapping. Intended for tests inside the cli package.
func DefaultHTTPClientWithTransport(t http.RoundTripper) *http.Client {
	c := defaultHTTPClient()
	c.Transport = t
	return c
}
