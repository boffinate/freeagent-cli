//go:build !readonly

package freeagent

import "net/http"

func enforceReadOnly(method, urlStr string) error { return nil }

func defaultHTTPClient() *http.Client {
	return &http.Client{Timeout: timeoutDefault}
}
