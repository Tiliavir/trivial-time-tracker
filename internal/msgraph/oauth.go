package msgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Token holds an OAuth2 access/refresh token pair.
// JSON field names match golang.org/x/oauth2.Token for compatibility
// with existing stored token files (~/.ttt/auth/msgraph_tokens.json).
type Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
}

// Valid reports whether t can be used (non-empty access token, not expired).
// A 10-second safety margin is applied before the expiry time.
func (t *Token) Valid() bool {
	return t != nil && t.AccessToken != "" &&
		(t.Expiry.IsZero() || time.Now().Add(10*time.Second).Before(t.Expiry))
}

// oauthConfig holds the OAuth2 endpoints and client credentials for the
// device code flow (RFC 8628).
type oauthConfig struct {
	ClientID      string
	Scopes        []string
	DeviceAuthURL string
	TokenURL      string
}

// deviceCodeResp is the raw JSON response from the device authorization endpoint.
type deviceCodeResp struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// tokenResp is the raw JSON response from the token endpoint.
type tokenResp struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
}

// requestDeviceCode initiates the device authorization flow and returns the
// device code response.
func requestDeviceCode(ctx context.Context, cfg *oauthConfig) (*deviceCodeResp, error) {
	form := url.Values{
		"client_id": {cfg.ClientID},
		"scope":     {strings.Join(cfg.Scopes, " ")},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.DeviceAuthURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device auth request: %w", err)
	}
	defer resp.Body.Close()
	var dc deviceCodeResp
	if err := json.NewDecoder(resp.Body).Decode(&dc); err != nil {
		return nil, fmt.Errorf("decoding device code response: %w", err)
	}
	return &dc, nil
}

// pollForToken polls the token endpoint at the given interval (seconds) until
// an access token is granted or ctx is cancelled.
func pollForToken(ctx context.Context, cfg *oauthConfig, deviceCode string, interval int) (*Token, error) {
	if interval <= 0 {
		interval = 5
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			tok, pending, err := doTokenRequest(ctx, cfg, url.Values{
				"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
				"device_code": {deviceCode},
				"client_id":   {cfg.ClientID},
			})
			if err != nil {
				return nil, err
			}
			if pending {
				continue
			}
			return tok, nil
		}
	}
}

// refreshAccessToken exchanges a refresh token for a new access token.
func refreshAccessToken(ctx context.Context, cfg *oauthConfig, refreshToken string) (*Token, error) {
	tok, _, err := doTokenRequest(ctx, cfg, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {cfg.ClientID},
		"scope":         {strings.Join(cfg.Scopes, " ")},
	})
	return tok, err
}

// doTokenRequest sends a POST to the token endpoint and returns (token, pending, error).
// pending is true when the server returns authorization_pending or slow_down.
func doTokenRequest(ctx context.Context, cfg *oauthConfig, params url.Values) (*Token, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()
	var tr tokenResp
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, false, fmt.Errorf("decoding token response: %w", err)
	}
	if tr.Error == "authorization_pending" || tr.Error == "slow_down" {
		return nil, true, nil
	}
	if tr.Error != "" {
		return nil, false, fmt.Errorf("token error: %s", tr.Error)
	}
	expiry := time.Time{}
	if tr.ExpiresIn > 0 {
		expiry = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	}
	return &Token{
		AccessToken:  tr.AccessToken,
		TokenType:    tr.TokenType,
		RefreshToken: tr.RefreshToken,
		Expiry:       expiry,
	}, false, nil
}
