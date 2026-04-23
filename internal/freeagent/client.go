package freeagent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/anjor/freeagent-cli/internal/storage"
)

const (
	timeoutDefault = 30 * time.Second
)

type Client struct {
	BaseURL      string
	UserAgent    string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Profile      string
	Store        storage.TokenStore
	HTTP         *http.Client
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return defaultHTTPClient()
}

func (c *Client) ResolveURL(path string) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path, nil
	}
	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", err
	}
	origin := fmt.Sprintf("%s://%s", base.Scheme, base.Host)
	if strings.HasPrefix(path, "/v2") {
		return origin + path, nil
	}
	if strings.HasPrefix(path, "/") {
		return strings.TrimRight(c.BaseURL, "/") + path, nil
	}
	return strings.TrimRight(c.BaseURL, "/") + "/" + path, nil
}

func (c *Client) ApproveURL(state string) (string, error) {
	u, err := url.Parse(strings.TrimRight(c.BaseURL, "/") + "/approve_app")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("client_id", c.ClientID)
	q.Set("response_type", "code")
	if c.RedirectURI != "" {
		q.Set("redirect_uri", c.RedirectURI)
	}
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (c *Client) ExchangeCode(ctx context.Context, code string) (*storage.Token, error) {
	payload := url.Values{}
	payload.Set("grant_type", "authorization_code")
	payload.Set("code", code)
	if c.RedirectURI != "" {
		payload.Set("redirect_uri", c.RedirectURI)
	}
	return c.tokenRequest(ctx, payload)
}

func (c *Client) Refresh(ctx context.Context, refreshToken string) (*storage.Token, error) {
	payload := url.Values{}
	payload.Set("grant_type", "refresh_token")
	payload.Set("refresh_token", refreshToken)
	return c.tokenRequest(ctx, payload)
}

func (c *Client) tokenRequest(ctx context.Context, payload url.Values) (*storage.Token, error) {
	u := strings.TrimRight(c.BaseURL, "/") + "/token_endpoint"
	if err := enforceReadOnly(http.MethodPost, u); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(payload.Encode()))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.ClientID, c.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("token request failed (%d): %s", resp.StatusCode, string(body))
	}

	var decoded struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(time.Duration(decoded.ExpiresIn) * time.Second)
	return &storage.Token{
		AccessToken:  decoded.AccessToken,
		RefreshToken: decoded.RefreshToken,
		TokenType:    decoded.TokenType,
		ExpiresAt:    expiresAt,
	}, nil
}

func (c *Client) AccessToken(ctx context.Context) (*storage.Token, error) {
	if c.Store == nil {
		return nil, errors.New("token store not configured")
	}
	stored, err := c.Store.Get(c.Profile)
	if err != nil {
		return nil, err
	}
	if stored.AccessToken == "" {
		return nil, errors.New("access token missing")
	}

	if stored.ExpiresAt.IsZero() || time.Now().Before(stored.ExpiresAt.Add(-1*time.Minute)) {
		return stored, nil
	}

	if stored.RefreshToken == "" {
		return stored, nil
	}

	refreshed, err := c.Refresh(ctx, stored.RefreshToken)
	if err != nil {
		return nil, err
	}
	if refreshed.RefreshToken == "" {
		refreshed.RefreshToken = stored.RefreshToken
	}
	if err := c.Store.Set(c.Profile, refreshed); err != nil {
		return nil, err
	}
	return refreshed, nil
}

func (c *Client) DoJSON(ctx context.Context, method, path string, payload any) ([]byte, int, http.Header, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, nil, err
		}
		body = bytes.NewReader(data)
	}
	return c.Do(ctx, method, path, body, "application/json")
}

func (c *Client) Do(ctx context.Context, method, path string, body io.Reader, contentType string) ([]byte, int, http.Header, error) {
	urlStr, err := c.ResolveURL(path)
	if err != nil {
		return nil, 0, nil, err
	}
	if err := enforceReadOnly(method, urlStr); err != nil {
		return nil, 0, nil, err
	}

	var payload []byte
	if body != nil {
		data, err := io.ReadAll(body)
		if err != nil {
			return nil, 0, nil, err
		}
		payload = data
	}

	token, err := c.AccessToken(ctx)
	if err != nil {
		return nil, 0, nil, err
	}

	respBody, status, headers, err := c.doRequest(ctx, method, urlStr, payload, contentType, token.AccessToken)
	if err == nil && status != http.StatusUnauthorized {
		return respBody, status, headers, nil
	}

	if status == http.StatusUnauthorized && token.RefreshToken != "" {
		refreshed, refreshErr := c.Refresh(ctx, token.RefreshToken)
		if refreshErr == nil {
			if refreshed.RefreshToken == "" {
				refreshed.RefreshToken = token.RefreshToken
			}
			if err := c.Store.Set(c.Profile, refreshed); err == nil {
				respBody, status, headers, err = c.doRequest(ctx, method, urlStr, payload, contentType, refreshed.AccessToken)
				return respBody, status, headers, err
			}
		}
	}

	return respBody, status, headers, err
}

func (c *Client) doRequest(ctx context.Context, method, urlStr string, body []byte, contentType, accessToken string) ([]byte, int, http.Header, error) {
	retry := 0
	for {
		if err := enforceReadOnly(method, urlStr); err != nil {
			return nil, 0, nil, err
		}
		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, urlStr, reader)
		if err != nil {
			return nil, 0, nil, err
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		req.Header.Set("Accept", "application/json")
		if c.UserAgent != "" {
			req.Header.Set("User-Agent", c.UserAgent)
		}
		if accessToken != "" {
			req.Header.Set("Authorization", "Bearer "+accessToken)
		}

		resp, err := c.httpClient().Do(req)
		if err != nil {
			return nil, 0, nil, err
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, resp.StatusCode, resp.Header, err
		}
		if resp.StatusCode == http.StatusTooManyRequests && retry == 0 {
			retry++
			if wait := resp.Header.Get("Retry-After"); wait != "" {
				if secs, convErr := time.ParseDuration(wait + "s"); convErr == nil {
					time.Sleep(secs)
					continue
				}
			}
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return data, resp.StatusCode, resp.Header, fmt.Errorf("request failed (%d): %s", resp.StatusCode, string(data))
		}
		return data, resp.StatusCode, resp.Header, nil
	}
}
