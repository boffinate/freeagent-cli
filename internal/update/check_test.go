package update

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// roundTripFunc lets tests stub *http.Client transports without spinning up a
// real httptest server. Each call to RoundTrip returns whatever fn produces.
type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func makeResponse(status int, body string, headers map[string]string) *http.Response {
	resp := &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{},
	}
	for k, v := range headers {
		resp.Header.Set(k, v)
	}
	return resp
}

func TestLatestRelease200OK(t *testing.T) {
	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("missing Accept header, got %q", req.Header.Get("Accept"))
		}
		if req.Header.Get("User-Agent") == "" {
			t.Error("missing User-Agent header")
		}
		body := `{"tag_name":"v1.2.3","html_url":"https://github.com/boffinate/freeagent-cli/releases/tag/v1.2.3"}`
		return makeResponse(http.StatusOK, body, nil), nil
	})

	tag, htmlURL, err := LatestRelease(context.Background(), client)
	if err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}
	if tag != "v1.2.3" {
		t.Errorf("tag = %q, want v1.2.3", tag)
	}
	if htmlURL != "https://github.com/boffinate/freeagent-cli/releases/tag/v1.2.3" {
		t.Errorf("htmlURL = %q", htmlURL)
	}
}

func TestLatestReleaseRateLimited(t *testing.T) {
	resetUnix := time.Now().Add(30 * time.Minute).Unix()
	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		return makeResponse(http.StatusForbidden, "rate limit exceeded", map[string]string{
			"X-RateLimit-Remaining": "0",
			"X-RateLimit-Reset":     strconvFormatInt(resetUnix),
		}), nil
	})

	_, _, err := LatestRelease(context.Background(), client)
	var rate *ErrRateLimited
	if !errors.As(err, &rate) {
		t.Fatalf("want *ErrRateLimited, got %T: %v", err, err)
	}
	if rate.Reset.Unix() != resetUnix {
		t.Errorf("reset = %v, want unix %d", rate.Reset, resetUnix)
	}
}

func TestLatestRelease5xx(t *testing.T) {
	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		return makeResponse(http.StatusInternalServerError, "boom", nil), nil
	})

	_, _, err := LatestRelease(context.Background(), client)
	if err == nil {
		t.Fatal("want error on 5xx")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected status code in error, got %v", err)
	}
}

func TestLatestReleaseMalformedJSON(t *testing.T) {
	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		return makeResponse(http.StatusOK, "{not json", nil), nil
	})

	_, _, err := LatestRelease(context.Background(), client)
	if err == nil {
		t.Fatal("want error on malformed JSON")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("want decode-related error, got %v", err)
	}
}

func TestLatestReleaseMissingTag(t *testing.T) {
	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		return makeResponse(http.StatusOK, `{"html_url":"x"}`, nil), nil
	})
	_, _, err := LatestRelease(context.Background(), client)
	if err == nil {
		t.Fatal("want error on missing tag_name")
	}
}

func TestLatestReleaseNetworkError(t *testing.T) {
	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("dial: refused")
	})
	_, _, err := LatestRelease(context.Background(), client)
	if err == nil {
		t.Fatal("want error on network failure")
	}
}

// strconvFormatInt is a tiny shim so the test file doesn't need to import
// strconv just for one call. Keeps the test imports tight.
func strconvFormatInt(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
