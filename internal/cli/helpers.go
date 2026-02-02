package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"freegant-cli/internal/config"
	"freegant-cli/internal/freeagent"
	"freegant-cli/internal/storage"
)

func loadConfig(rt Runtime) (*config.Config, string, error) {
	cfg, cfgPath, err := config.Load(rt.ConfigPath)
	if err != nil {
		return nil, "", err
	}
	return cfg, cfgPath, nil
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
		profile.UserAgent = "freegant-cli/0.1"
	}
	return profile
}

func saveProfile(cfg *config.Config, profileName, cfgPath string, profile config.Profile) error {
	cfg.SetProfile(profileName, profile)
	return cfg.Save(cfgPath)
}

func newClient(ctx context.Context, rt Runtime, profile config.Profile) (*freeagent.Client, *storage.Store, error) {
	store, err := storage.NewDefaultStore()
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
