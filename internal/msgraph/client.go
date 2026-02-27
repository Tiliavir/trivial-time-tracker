package msgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const graphBaseURL = "https://graph.microsoft.com/v1.0"

// tokenTransport is an http.RoundTripper that injects a Bearer token and
// refreshes it automatically when it expires.
type tokenTransport struct {
	mu  sync.Mutex
	tok *Token
	cfg *oauthConfig
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	tok := t.tok
	t.mu.Unlock()

	if !tok.Valid() && tok.RefreshToken != "" {
		if newTok, err := refreshAccessToken(req.Context(), t.cfg, tok.RefreshToken); err == nil {
			_ = saveToken(newTok)
			t.mu.Lock()
			t.tok = newTok
			tok = newTok
			t.mu.Unlock()
		}
	}

	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	return http.DefaultTransport.RoundTrip(req2)
}

// Client is an authenticated Microsoft Graph API client.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new Graph API client using the provided token and config.
func NewClient(_ context.Context, tok *Token, cfg *oauthConfig) *Client {
	return &Client{
		httpClient: &http.Client{
			Transport: &tokenTransport{tok: tok, cfg: cfg},
		},
	}
}

// CalendarEvent represents a Microsoft Graph calendar event.
type CalendarEvent struct {
	ID          string `json:"id"`
	Subject     string `json:"subject"`
	BodyPreview string `json:"bodyPreview"`
	IsAllDay    bool   `json:"isAllDay"`
	IsCancelled bool   `json:"isCancelled"`
	Sensitivity string `json:"sensitivity"` // "normal", "personal", "private", "confidential"
	ShowAs      string `json:"showAs"`      // "free", "tentative", "busy", "oof", "workingElsewhere", "unknown"
	Start       struct {
		DateTime string `json:"dateTime"`
		TimeZone string `json:"timeZone"`
	} `json:"start"`
	End struct {
		DateTime string `json:"dateTime"`
		TimeZone string `json:"timeZone"`
	} `json:"end"`
	Location struct {
		DisplayName string `json:"displayName"`
	} `json:"location"`
}

// calendarViewResponse is the Graph API paged response for calendar events.
type calendarViewResponse struct {
	Value    []CalendarEvent `json:"value"`
	NextLink string          `json:"@odata.nextLink"`
}

// GetCalendarView fetches calendar events in [from, to) using the calendarView endpoint.
// timezone is an IANA timezone name (e.g. "Europe/Berlin"); pass "" for UTC.
func (c *Client) GetCalendarView(ctx context.Context, from, to time.Time, timezone string) ([]CalendarEvent, error) {
	startISO := from.UTC().Format(time.RFC3339)
	endISO := to.UTC().Format(time.RFC3339)

	endpoint := fmt.Sprintf("%s/me/calendarView?startDateTime=%s&endDateTime=%s&$top=100",
		graphBaseURL,
		url.QueryEscape(startISO),
		url.QueryEscape(endISO),
	)

	var all []CalendarEvent
	for endpoint != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		if timezone != "" {
			req.Header.Set("Prefer", fmt.Sprintf(`outlook.timezone="%s"`, timezone))
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("graph API request failed: %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response body: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("graph API error %d: %s", resp.StatusCode, string(body))
		}

		var page calendarViewResponse
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("decoding graph response: %w", err)
		}

		all = append(all, page.Value...)
		endpoint = page.NextLink
	}
	return all, nil
}
