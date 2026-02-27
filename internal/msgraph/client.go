package msgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"
)

const graphBaseURL = "https://graph.microsoft.com/v1.0"

// Client is an authenticated Microsoft Graph API client.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new Graph API client using the provided token and config.
func NewClient(ctx context.Context, tok *oauth2.Token, cfg *oauth2.Config) *Client {
	ts := cfg.TokenSource(ctx, tok)
	return &Client{
		httpClient: oauth2.NewClient(ctx, &savingTokenSource{ts: ts}),
	}
}

// savingTokenSource wraps a TokenSource and persists refreshed tokens.
type savingTokenSource struct {
	ts oauth2.TokenSource
}

func (s *savingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := s.ts.Token()
	if err != nil {
		return nil, err
	}
	// Best-effort save; ignore errors.
	_ = saveToken(tok)
	return tok, nil
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
