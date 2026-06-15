// Package airbnb is the library behind the airbnb command line: an HTTP client
// for Airbnb's public web pages and its internal data plane, and the typed
// records every command emits.
//
// Airbnb renders a listing page server-side and embeds the whole client state as
// a JSON island (data-deferred-state-0), so this client GETs the page and reads
// that island; every other surface (search, reviews, calendar, host,
// experiences) is the logged-out web client's own GraphQL endpoint (api/v3),
// addressed by the public web key the site ships and a per-operation
// persisted-query hash, plus two lighter api/v2 JSON endpoints the search box
// uses. Airbnb fronts its whole estate with an edge bot manager (Cloudflare and
// DataDome) that classifies a request on IP reputation and TLS fingerprint
// before the application sees it, so from a datacenter IP nearly everything comes
// back as a bodyless 403; from a residential or mobile connection the same
// requests return the data. There is no official API to fall back to, so a
// walled read returns ErrBlocked (exit 4) with a message that names the remedy.
// This client does not forge a TLS fingerprint and does not rent or rotate IPs:
// it reads what a logged-out browser reads, the way it reads it. Each surface
// lives in its own file (search.go, room.go, reviews.go, calendar.go, host.go,
// experiences.go, suggest.go) with its parsing and record mapping; this file
// holds the shared web client and api.go holds the GraphQL client.
package airbnb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Client talks to Airbnb's public web pages and its internal data plane. It
// paces requests, retries the transient failures, detects the edge bot wall, and
// caches response bodies on disk keyed by the request.
type Client struct {
	HTTP      *http.Client
	BaseURL   string
	UserAgent string
	Locale    string
	Currency  string
	Adults    int
	Children  int
	Checkin   string
	Checkout  string
	Delay     time.Duration
	Retries   int

	apiKey string            // the public web key; defaultAPIKey unless overridden
	hashes map[string]string // persisted-query hashes by operation

	cache   *cache
	refresh bool

	mu   sync.Mutex
	last time.Time
}

// NewClient builds a client from cfg.
func NewClient(cfg Config) *Client {
	c := &Client{
		HTTP:      &http.Client{Timeout: cfg.Timeout},
		BaseURL:   cfg.BaseURL,
		UserAgent: cfg.UserAgent,
		Locale:    cfg.Locale,
		Currency:  cfg.Currency,
		Adults:    cfg.Adults,
		Children:  cfg.Children,
		Checkin:   cfg.Checkin,
		Checkout:  cfg.Checkout,
		Delay:     cfg.Delay,
		Retries:   cfg.Retries,
		apiKey:    cfg.APIKey,
		refresh:   cfg.Refresh,
	}
	if c.BaseURL == "" {
		c.BaseURL = BaseURL
	}
	if c.UserAgent == "" {
		c.UserAgent = DefaultUserAgent
	}
	if c.Locale == "" {
		c.Locale = defaultLocale
	}
	if c.Currency == "" {
		c.Currency = defaultCurrency
	}
	if c.apiKey == "" {
		c.apiKey = defaultAPIKey
	}
	// Each client gets its own copy of the default hashes so a live refresh on
	// one does not race another.
	c.hashes = make(map[string]string, len(defaultHashes))
	for k, v := range defaultHashes {
		c.hashes[k] = v
	}
	// --refresh keeps the cache (so it is rewritten) but skips reads. --no-cache
	// drops it entirely.
	if !cfg.NoCache {
		c.cache = newCache(cfg.CacheDir, cfg.CacheTTL)
	}
	return c
}

// get fetches url and returns the response body: paced, retried, cached, and
// wall-checked.
func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	if !c.refresh {
		if b, ok := c.cache.get(url); ok {
			return b, nil
		}
	}
	var lastErr error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, http.MethodGet, url, nil, nil)
		if err == nil {
			c.cache.put(url, body)
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, lastErr
}

// do performs one request and returns the body. retry reports whether the
// failure is worth another attempt. header, when non-nil, is applied to the
// request; reqBody, when non-nil, is the POST payload.
func (c *Client) do(ctx context.Context, method, url string, header http.Header, reqBody []byte) (body []byte, retry bool, err error) {
	c.pace()
	var rdr io.Reader
	if reqBody != nil {
		rdr = bytes.NewReader(reqBody)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	for k, vs := range header {
		for _, v := range vs {
			req.Header.Set(k, v)
		}
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		// A connection reset mid-handshake is how the edge sometimes drops a
		// datacenter request; treat a transport error as retryable.
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusOK:
		// fall through to read and check the body
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, true, ErrRateLimited
	case resp.StatusCode == http.StatusForbidden,
		resp.StatusCode == http.StatusUnauthorized:
		// 403 is the Cloudflare/DataDome edge block; 401 a rejected key.
		return nil, false, ErrBlocked
	case resp.StatusCode == http.StatusNotFound, resp.StatusCode == http.StatusGone:
		return nil, false, ErrNotFound
	case resp.StatusCode >= 500:
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	default:
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	if isChallenge(b) {
		return nil, false, ErrBlocked
	}
	return b, false, nil
}

// isChallenge reports whether a 200 body is in fact the edge bot wall, which
// DataDome and Cloudflare sometimes serve with a 200 status, or an empty body on
// a page that should be large. The markers are the DataDome captcha host and the
// Cloudflare challenge platform.
func isChallenge(b []byte) bool {
	if len(bytes.TrimSpace(b)) == 0 {
		return true
	}
	return bytes.Contains(b, []byte("captcha-delivery.com")) ||
		bytes.Contains(b, []byte("/cdn-cgi/challenge-platform")) ||
		bytes.Contains(b, []byte("cf-chl-"))
}

// pace blocks until at least Delay has passed since the previous request.
func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Delay <= 0 {
		c.last = time.Now()
		return
	}
	if wait := c.Delay - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// ClearCache removes the on-disk cache.
func (c *Client) ClearCache() error { return c.cache.clear() }
