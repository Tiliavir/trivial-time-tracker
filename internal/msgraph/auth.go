package msgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

var requiredScopes = []string{
	"https://graph.microsoft.com/Calendars.Read",
	"offline_access",
}

func msEndpoint(tenantID, path string) string {
	return "https://login.microsoftonline.com/" + tenantID + "/oauth2/v2.0/" + path
}

// tokenFilePath returns the path to the stored token file.
func tokenFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".ttt", "auth", "msgraph_tokens.json"), nil
}

// oauth2Config returns the oauth2.Config for Microsoft Graph using the
// provided tenant and client IDs.
func oauth2Config(tenantID, clientID string) *oauth2.Config {
	return &oauth2.Config{
		ClientID: clientID,
		Scopes:   requiredScopes,
		Endpoint: oauth2.Endpoint{
			DeviceAuthURL: msEndpoint(tenantID, "devicecode"),
			TokenURL:      msEndpoint(tenantID, "token"),
			AuthStyle:     oauth2.AuthStyleInParams,
		},
	}
}

// loadToken loads a previously saved token from disk.
func loadToken() (*oauth2.Token, error) {
	path, err := tokenFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading token file: %w", err)
	}
	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("corrupt token file (delete %s to re-authenticate): %w", path, err)
	}
	return &tok, nil
}

// saveToken persists a token to disk.
func saveToken(tok *oauth2.Token) error {
	path, err := tokenFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating auth directory: %w", err)
	}
	data, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling token: %w", err)
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("writing token file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("saving token file: %w", err)
	}
	return nil
}

// GetHTTPClient returns an authenticated HTTP client for Microsoft Graph.
// It loads saved tokens, refreshes them if needed, or initiates a new
// device code flow if no valid token is available.
// tenantID and clientID are read from ~/.ttt/config.json.
func GetHTTPClient(ctx context.Context, tenantID, clientID string) (*oauth2.Token, *oauth2.Config, error) {
	cfg := oauth2Config(tenantID, clientID)

	tok, err := loadToken()
	if err != nil {
		// Corrupt token — warn and re-auth.
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		tok = nil
	}

	if tok != nil && tok.Valid() {
		return tok, cfg, nil
	}

	// Try to refresh.
	if tok != nil && tok.RefreshToken != "" {
		ts := cfg.TokenSource(ctx, tok)
		refreshed, err := ts.Token()
		if err == nil {
			if err2 := saveToken(refreshed); err2 != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not save refreshed token: %v\n", err2)
			}
			return refreshed, cfg, nil
		}
		fmt.Fprintf(os.Stderr, "Token refresh failed (%v), re-authenticating...\n", err)
	}

	// Device code flow.
	resp, err := cfg.DeviceAuth(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("device auth request failed: %w", err)
	}

	fmt.Println()
	fmt.Println("To sign in, use a web browser to open the page:")
	fmt.Printf("  %s\n", resp.VerificationURI)
	fmt.Printf("Enter the code: %s\n", resp.UserCode)
	fmt.Println()

	newTok, err := cfg.DeviceAccessToken(ctx, resp)
	if err != nil {
		return nil, nil, fmt.Errorf("device authentication failed: %w", err)
	}

	if err := saveToken(newTok); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save token: %v\n", err)
	}

	return newTok, cfg, nil
}
