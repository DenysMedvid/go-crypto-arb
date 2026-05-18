package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go-crypto-arb/internal/marketdata"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 8 * time.Second},
	}
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) Snapshot(ctx context.Context) (marketdata.Snapshot, time.Duration, error) {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/snapshot", nil)
	if err != nil {
		return marketdata.Snapshot{}, 0, err
	}
	req.Header.Set("X-API-Key", c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return marketdata.Snapshot{}, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return marketdata.Snapshot{}, time.Since(start), fmt.Errorf("backend returned %s", resp.Status)
	}
	var snapshot marketdata.Snapshot
	if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		return marketdata.Snapshot{}, time.Since(start), err
	}
	return snapshot, time.Since(start), nil
}
