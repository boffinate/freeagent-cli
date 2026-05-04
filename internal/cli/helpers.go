package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/boffinate/freeagent-cli/internal/config"
	"github.com/boffinate/freeagent-cli/internal/freeagent"
	"github.com/boffinate/freeagent-cli/internal/storage"
)

// newTokenStore produces the TokenStore used by newClient. Tests override this
// via installTestHooks to substitute an in-memory store and must restore the
// original via t.Cleanup. The hook is a package-level var, so tests that
// install hooks must not run in parallel.
var newTokenStore = func(rt Runtime) (storage.TokenStore, error) {
	return storage.NewDefaultStore()
}

// newHTTPClient produces the *http.Client used by newClient. Returning nil
// means "use freeagent.Client's default"; production code always returns nil.
// Tests override to route traffic through an httptest server while preserving
// the readonly redirect guard.
var newHTTPClient = func(rt Runtime) *http.Client { return nil }

// loadConfigHook indirects config.Load so tests can inject an in-memory
// config pointing BaseURL at api.sandbox.freeagent.com (the readonly build's
// allowed host) without touching the user's config file.
var loadConfigHook = func(rt Runtime) (*config.Config, string, error) {
	return config.Load(rt.ConfigPath)
}

func loadConfig(rt Runtime) (*config.Config, string, error) {
	return loadConfigHook(rt)
}

func ensureProfile(cfg *config.Config, profileName string, rt Runtime, overrides config.Profile) config.Profile {
	profile := cfg.Profile(profileName)

	if overrides.ClientID != "" {
		profile.ClientID = overrides.ClientID
	}
	if overrides.ClientSecret != "" {
		profile.ClientSecret = overrides.ClientSecret
	}
	if overrides.RedirectURI != "" {
		profile.RedirectURI = overrides.RedirectURI
	}
	if overrides.UserAgent != "" {
		profile.UserAgent = overrides.UserAgent
	}
	if overrides.BaseURL != "" {
		profile.BaseURL = overrides.BaseURL
	}

	if profile.BaseURL == "" {
		profile.BaseURL = rt.BaseURL
	}
	if profile.UserAgent == "" {
		profile.UserAgent = "freeagent-cli/0.1"
	}
	return profile
}

func saveProfile(cfg *config.Config, profileName, cfgPath string, profile config.Profile) error {
	cfg.SetProfile(profileName, profile)
	return cfg.Save(cfgPath)
}

func newClient(ctx context.Context, rt Runtime, profile config.Profile) (*freeagent.Client, storage.TokenStore, error) {
	store, err := newTokenStore(rt)
	if err != nil {
		return nil, nil, err
	}

	client := &freeagent.Client{
		BaseURL:      profile.BaseURL,
		UserAgent:    profile.UserAgent,
		ClientID:     profile.ClientID,
		ClientSecret: profile.ClientSecret,
		RedirectURI:  profile.RedirectURI,
		Profile:      rt.Profile,
		Subdomain:    rt.Subdomain,
		Store:        store,
		HTTP:         newHTTPClient(rt),
	}
	return client, store, nil
}

func normalizeResourceURL(baseURL, resource, value string) (string, error) {
	if value == "" {
		return "", errors.New("value required")
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value, nil
	}
	if strings.HasPrefix(value, "/v2/") {
		base := strings.TrimSuffix(baseURL, "/v2")
		return base + value, nil
	}
	if strings.HasPrefix(value, "/") {
		return strings.TrimSuffix(baseURL, "/v2") + value, nil
	}
	return strings.TrimSuffix(baseURL, "/v2") + "/v2/" + path.Join(resource, value), nil
}

func exitf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

func require(value, name string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", name)
	}
	return nil
}

func writeJSONOutput(data []byte) error {
	_, err := os.Stdout.Write(append(data, '\n'))
	return err
}

// writeRaw is the printOrJSON-fallback shorthand for resources without a
// custom table renderer: emit the bytes verbatim plus newline.
func writeRaw(resp []byte) error {
	_, err := os.Stdout.Write(append(resp, '\n'))
	return err
}

// getAndDecode issues a GET and decodes the JSON body into *T. Returns both
// the decoded value and the raw bytes so callers can choose JSON passthrough.
func getAndDecode[T any](ctx context.Context, client *freeagent.Client, path string) (*T, []byte, error) {
	resp, _, _, err := client.Do(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, resp, err
	}
	var decoded T
	if err := json.Unmarshal(resp, &decoded); err != nil {
		return nil, resp, err
	}
	return &decoded, resp, nil
}

// buildQueryParams URL-encodes a set of query parameters, skipping blanks
// after trimming. Values and keys are passed through verbatim otherwise.
func buildQueryParams(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}
	q := url.Values{}
	for k, v := range params {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		q.Set(k, v)
	}
	return q.Encode()
}

// appendQuery attaches an already-encoded query string to path.
func appendQuery(path, encoded string) string {
	if encoded == "" {
		return path
	}
	if strings.Contains(path, "?") {
		return path + "&" + encoded
	}
	return path + "?" + encoded
}

// printOrJSON writes raw to stdout when rt.JSONOutput is set; otherwise calls
// fallback to render whatever formatted output the caller prefers.
func printOrJSON(rt Runtime, raw []byte, fallback func() error) error {
	if rt.JSONOutput {
		return writeJSONOutput(raw)
	}
	return fallback()
}

// loadResourceObject reads a JSON file at bodyPath and returns the inner
// resource object (e.g. the "bill" / "expense" map). If the file's top-level
// object already contains a key matching wrapper, that value is returned;
// otherwise the whole object is treated as the resource. An empty bodyPath
// returns an empty map so callers can layer flag values on top.
func loadResourceObject(bodyPath, wrapper string) (map[string]any, error) {
	if bodyPath == "" {
		return map[string]any{}, nil
	}
	data, err := os.ReadFile(bodyPath)
	if err != nil {
		return nil, err
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, err
	}
	if inner, ok := decoded[wrapper].(map[string]any); ok {
		return inner, nil
	}
	return decoded, nil
}

// loadItemsArray reads a JSON file holding either {"<wrapper>": [...]} or a
// bare top-level array, and returns the array. Used for line-items flags.
func loadItemsArray(itemsPath, wrapper string) ([]any, error) {
	data, err := os.ReadFile(itemsPath)
	if err != nil {
		return nil, err
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, err
	}
	switch v := decoded.(type) {
	case []any:
		return v, nil
	case map[string]any:
		if items, ok := v[wrapper].([]any); ok {
			return items, nil
		}
		return nil, fmt.Errorf("expected top-level array or %q key", wrapper)
	default:
		return nil, fmt.Errorf("expected JSON array or object with %q key", wrapper)
	}
}

// requireIDOrURL resolves a flag pair (id, url) against a resource collection
// for commands that accept either. Returns the absolute request path.
func requireIDOrURL(baseURL, resource, id, urlValue string) (string, error) {
	if urlValue != "" {
		return urlValue, nil
	}
	if id != "" {
		return normalizeResourceURL(baseURL, resource, id)
	}
	return "", fmt.Errorf("id or url required")
}
