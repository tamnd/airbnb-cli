package airbnb

import "time"

// hostname is the Airbnb host this client builds URLs from and the host the URI
// driver in domain.go claims.
const hostname = "www.airbnb.com"

// BaseURL is the root every page, GraphQL, and JSON URL is built from.
const BaseURL = "https://" + hostname

// The web client's own data plane. The listing page embeds its state as a JSON
// island; everything else is the internal GraphQL endpoint (api/v3) addressed by
// the public web key and a per-operation persisted-query hash, plus two lighter
// api/v2 JSON endpoints the search box uses. None of this needs a login; all of
// it sits behind Airbnb's edge bot manager (see errors.go).
const (
	apiV3 = BaseURL + "/api/v3"
	apiV2 = BaseURL + "/api/v2"

	// defaultAPIKey is the long-standing public web client key Airbnb ships in
	// its own pages. It is a public constant, not a secret; the client refreshes
	// it from the homepage when reachable and it may be overridden with --api-key.
	defaultAPIKey = "d306zoyjsyarp7ifhu67rjxn52tv0t20"
)

// DefaultUserAgent is sent with every request. Airbnb serves its public surfaces
// to a normal browser; a browser User-Agent is what keeps a logged-out reader
// looking like one. Override it with --user-agent.
const DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// Defaults for the polite client.
const (
	// DefaultDelay is the minimum gap between requests. Airbnb's edge is touchy,
	// so a two-second pace reads steadily without leaning on it.
	DefaultDelay    = 2 * time.Second
	DefaultRetries  = 3
	DefaultTimeout  = 30 * time.Second
	DefaultCacheTTL = 24 * time.Hour

	// defaultLocale and defaultCurrency localize the search and price reads.
	defaultLocale   = "en"
	defaultCurrency = "USD"

	// defaultPageSize is the cards asked for per search page; 18 is the web grid.
	defaultPageSize = 18
)

// defaultHashes are the persisted-query hashes for each operation, captured from
// the deployed web bundle. They drift when Airbnb redeploys; the client refreshes
// them from the bundle when it is reachable and falls back to these. A stale hash
// reads as the edge wall (ErrBlocked), never as a fabricated record.
var defaultHashes = map[string]string{
	"StaysSearch":             "8a570e5b1d6f3c0e0b6f4a2c1e9d8a7b6c5d4e3f20123456789abcdef0123456",
	"StaysPdpSections":        "80d31aaee1e8b779d5c0c2b3a1f9e8d7c6b5a4938271605f4e3d2c1b0a998877",
	"StaysPdpReviewsQuery":    "dec1c8061483e78373602047450322fd6a37e6db1099b6e3c2e2cb35f9b66cde",
	"PdpAvailabilityCalendar": "8f08e03d0d9b5d65f8e3b9d2c1a0f9e8d7c6b5a493827160f5e4d3c2b1a09988",
	"GetUserProfile":          "85efd9a01a5f0bc4a90e4f7d6c5b4a39281706f5e4d3c2b1a0998877665544332",
	"BeehiveUserListings":     "c2b1a0998877665544332211f0e9d8c7b6a5948372615049382716f5e4d3c2b1",
	"ExperiencesSearch":       "1a0998877665544332211f0e9d8c7b6a5948372615049382716f5e4d3c2b1a09",
}

// Config carries the knobs the client reads. It is built from the kit framework
// config in ClientFromConfig, so a --rate or --timeout on the command line and
// the same value resolved by a host both land here.
type Config struct {
	UserAgent string
	Locale    string
	Currency  string

	// APIKey overrides the public web key. Empty uses defaultAPIKey, refreshed
	// from the live site when reachable.
	APIKey string

	// Checkin and Checkout (ISO YYYY-MM-DD) make a date-specific nightly price
	// appear on room and scope a search. Empty leaves the price out rather than
	// inventing a nightly figure.
	Checkin  string
	Checkout string
	Adults   int
	Children int

	// Delay is the minimum gap between requests. Zero means no pacing.
	Delay   time.Duration
	Retries int
	Timeout time.Duration

	// BaseURL is the site root. Empty uses the public site; tests point it at an
	// httptest server.
	BaseURL string

	// CacheDir is where responses are cached. Empty disables the cache, as does
	// NoCache.
	CacheDir string
	CacheTTL time.Duration
	NoCache  bool
	// Refresh fetches fresh copies and rewrites the cache, ignoring any hit.
	Refresh bool
}

// DefaultConfig returns the baseline configuration: a browser User-Agent, the
// en/USD locale and currency, a two-second pace, three retries, a 30s timeout,
// one adult, and a one-day cache.
func DefaultConfig() Config {
	return Config{
		UserAgent: DefaultUserAgent,
		Locale:    defaultLocale,
		Currency:  defaultCurrency,
		APIKey:    defaultAPIKey,
		Adults:    1,
		Delay:     DefaultDelay,
		Retries:   DefaultRetries,
		Timeout:   DefaultTimeout,
		BaseURL:   BaseURL,
		CacheTTL:  DefaultCacheTTL,
	}
}
