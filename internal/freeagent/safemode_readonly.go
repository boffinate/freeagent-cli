//go:build readonly

package freeagent

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// defaultHTTPClient returns a readonly-aware http.Client. Alongside Client.Do's
// pre-flight enforceReadOnly call, the CheckRedirect hook re-applies the guard
// to every redirect target: without it, a 30x response from an allowed host
// could forward the bearer token to a foreign host or over plaintext http.
// Go does strip Authorization on cross-host redirects, but not on same-host
// https->http downgrades, and we want one coherent policy regardless.
func defaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: timeoutDefault,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			return enforceReadOnly(req.Method, req.URL.String())
		},
	}
}

// DefaultHTTPClientWithTransport returns a client built from defaultHTTPClient
// with Transport replaced. The readonly CheckRedirect hook is preserved so
// tests continue to exercise the redirect guard. Intended for tests inside the
// cli package.
func DefaultHTTPClientWithTransport(t http.RoundTripper) *http.Client {
	c := defaultHTTPClient()
	c.Transport = t
	return c
}

// tokenEndpointPath is the exact URL path of FreeAgent's OAuth token endpoint.
// Only POST at this path (combined with an allowed host and https scheme) is
// permitted, because hitting it drives the OAuth flow and never touches
// FreeAgent business data.
const tokenEndpointPath = "/v2/token_endpoint"

// allowedReadonlyHosts is the closed set of hosts the readonly binary may
// talk to. It is deliberately not user-configurable: the whole point of the
// readonly build is that bearer tokens cannot be sent to attacker-controlled
// hosts, regardless of flag values or response content.
var allowedReadonlyHosts = map[string]struct{}{
	"api.freeagent.com":         {},
	"api.sandbox.freeagent.com": {},
}

// enforceReadOnly refuses, in order:
//   - any request whose URL is unparseable or has an empty host,
//   - any request whose scheme is not https (prevents plaintext transport of
//     bearer tokens and OAuth client credentials),
//   - any request to a host outside allowedReadonlyHosts (prevents token
//     exfiltration via --base-url or server-returned absolute URLs),
//   - any method other than GET/HEAD, unless the request is a POST to the
//     exact OAuth token endpoint path.
//
// An empty method string is treated as GET because net/http defaults to GET
// on empty.
func enforceReadOnly(method, urlStr string) error {
	parsed, err := url.Parse(urlStr)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("readonly build: refusing request with unparseable or empty host: %s", urlStr)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("readonly build: refusing non-https request (scheme=%q): %s", parsed.Scheme, urlStr)
	}
	if _, ok := allowedReadonlyHosts[parsed.Host]; !ok {
		return fmt.Errorf("readonly build: refusing request to non-FreeAgent host %s", parsed.Host)
	}
	switch method {
	case http.MethodGet, http.MethodHead, "":
		return nil
	}
	if method == http.MethodPost && parsed.Path == tokenEndpointPath {
		return nil
	}
	return fmt.Errorf("readonly build: refusing %s %s", method, urlStr)
}
