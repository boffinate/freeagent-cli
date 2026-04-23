package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/anjor/freeagent-cli/internal/config"
	"github.com/anjor/freeagent-cli/internal/freeagent"
	"github.com/anjor/freeagent-cli/internal/storage"
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
