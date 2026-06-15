package airbnb

import (
	"context"
	"errors"
	"time"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes airbnb as a kit Domain: a driver that a multi-domain host
// (ant) enables with a single blank import,
//
//	import _ "github.com/tamnd/airbnb-cli/airbnb"
//
// exactly as a database/sql program enables a driver with `import _
// "github.com/lib/pq"`. The init below registers it; the host then dereferences
// airbnb:// URIs by routing to the operations Register installs. The same Domain
// also builds the standalone airbnb binary (see cli.NewApp), so the binary and a
// host share one source of truth.
func init() { kit.Register(Domain{}) }

// Domain is the airbnb driver. It carries no state; the per-run client is built
// by the factory Register hands kit.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against, and
// the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme:   "airbnb",
		Hosts:    []string{hostname, "airbnb.com"},
		Identity: Identity(),
	}
}

// Identity is the fixed description of the airbnb CLI, shared by the domain and
// the standalone composition root so help and version read the same everywhere.
func Identity() kit.Identity {
	return kit.Identity{
		Binary: "airbnb",
		Short:  "Read public Airbnb stays, listings, reviews, calendars, hosts, and experiences into structured records",
		Long: `airbnb reads public Airbnb data the way a logged-out browser does:
stay search by place, a single listing, its reviews, its availability
and nightly prices, a host profile and the host's listings, experience
search, and location autocomplete. Airbnb fronts its whole site with an
edge bot manager (Cloudflare and DataDome) that classifies a request on
IP reputation and TLS fingerprint, so from a datacenter IP nearly every
surface returns a wall (exit 4); the data is public and needs no
account, and from a residential or mobile connection these surfaces
answer. There is no official API to fall back to, and this tool does not
forge a TLS fingerprint or rent an IP to get past the edge. It returns
records as a table, JSON, JSONL, CSV, TSV, or URLs, and serves the same
operations over HTTP and MCP.

airbnb is an independent tool and is not affiliated with Airbnb.`,
		Site: BaseURL,
		Repo: "https://github.com/tamnd/airbnb-cli",
	}
}

// Register installs the client factory and every operation onto app. A resolver
// op (Single) names its own record type and answers `ant get`; a List op
// enumerates a parent resource's members and answers `ant ls`.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)
	app.CommandGroup("read", "Read public Airbnb data")
	app.CommandGroup("host", "Read a host profile and the host's listings")
	app.CommandGroup("ref", "Resolve references to ids and URLs (offline)")

	// Top-level reads. Each list op names its own collection authority, distinct
	// from the room and host resolvers, so `ant ls airbnb://search/<place>`,
	// airbnb://reviews/<id>, airbnb://calendar/<id>, and
	// airbnb://experiences/<place> each reach the right op rather than shadowing
	// one another; every member they emit links back to its listing.
	kit.Handle(app, kit.OpMeta{
		Name: "search", Group: "read", List: true,
		Summary: "Search stays in a place",
		URIType: "search",
		Args:    []kit.Arg{{Name: "place", Help: "a city, region, or place name"}},
	}, search)

	kit.Handle(app, kit.OpMeta{
		Name: "room", Group: "read", Single: true,
		Summary: "Show one listing by id",
		URIType: "room", Resolver: true,
		Args: []kit.Arg{{Name: "id", Help: "room id or /rooms/ URL"}},
	}, getRoom)

	kit.Handle(app, kit.OpMeta{
		Name: "reviews", Group: "read", List: true,
		Summary: "List a listing's reviews",
		URIType: "reviews",
		Args:    []kit.Arg{{Name: "id", Help: "room id or /rooms/ URL"}},
	}, reviews)

	kit.Handle(app, kit.OpMeta{
		Name: "calendar", Group: "read", List: true,
		Summary: "A listing's availability and nightly price",
		URIType: "calendar",
		Args:    []kit.Arg{{Name: "id", Help: "room id or /rooms/ URL"}},
	}, calendar)

	kit.Handle(app, kit.OpMeta{
		Name: "experiences", Group: "read", List: true,
		Summary: "Search experiences in a place",
		URIType: "experiences",
		Args:    []kit.Arg{{Name: "place", Help: "a city, region, or place name"}},
	}, experiences)

	kit.Handle(app, kit.OpMeta{
		Name: "suggest", Group: "read", List: true,
		Summary: "Location autocomplete suggestions",
		Args:    []kit.Arg{{Name: "prefix", Help: "the typed prefix"}},
	}, suggest)

	// Host: a profile, the host's listings.
	kit.Handle(app, kit.OpMeta{
		Name: "show", Parent: "host", Single: true,
		Summary: "Show a host's profile",
		URIType: "host", Resolver: true,
		Args: []kit.Arg{{Name: "id", Help: "user id or /users/show/ URL"}},
	}, getHost)

	kit.Handle(app, kit.OpMeta{
		Name: "listings", Parent: "host", List: true,
		Summary: "List a host's public listings",
		URIType: "listings", // ls airbnb://listings/<id>, distinct from the host resolver
		Args:    []kit.Arg{{Name: "id", Help: "user id or /users/show/ URL"}},
	}, hostListings)

	// Reference tools (offline).
	kit.Handle(app, kit.OpMeta{
		Name: "id", Parent: "ref", Single: true,
		Summary: "Classify a reference into its (kind, id)",
		Args:    []kit.Arg{{Name: "ref", Help: "any Airbnb URL, path, id, or global id"}},
	}, classifyRef)

	kit.Handle(app, kit.OpMeta{
		Name: "url", Parent: "ref", Single: true,
		Summary: "Build the canonical URL for a (kind, id)",
		Args: []kit.Arg{
			{Name: "kind", Help: "room, host, or experience"},
			{Name: "id", Help: "the id for that kind"},
		},
	}, buildURL)
}

// newClient builds the client from the host-resolved config, so a host and the
// standalone binary pace and identify themselves the same way.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	return ClientFromConfig(cfg), nil
}

// ClientFromConfig maps the framework config onto an airbnb.Config and returns a
// client. There are no credentials to read: the public web key is a constant,
// overridable only with --api-key, and Airbnb has no opt-in signed backend.
func ClientFromConfig(cfg kit.Config) *Client {
	ac := DefaultConfig()
	if cfg.Rate > 0 {
		ac.Delay = cfg.Rate
	}
	if cfg.Retries >= 0 {
		ac.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		ac.Timeout = cfg.Timeout
	}
	if ua := cfg.Extra["user-agent"]; ua != "" {
		ac.UserAgent = ua
	} else if cfg.UserAgent != "" {
		ac.UserAgent = cfg.UserAgent
	}
	if v := cfg.Extra["locale"]; v != "" {
		ac.Locale = v
	}
	if v := cfg.Extra["currency"]; v != "" {
		ac.Currency = v
	}
	if v := cfg.Extra["api-key"]; v != "" {
		ac.APIKey = v
	}
	if v := cfg.Extra["checkin"]; v != "" {
		ac.Checkin = v
	}
	if v := cfg.Extra["checkout"]; v != "" {
		ac.Checkout = v
	}
	if v := cfg.Extra["adults"]; v != "" {
		if n := atoiOr(v, 0); n > 0 {
			ac.Adults = n
		}
	}
	if v := cfg.Extra["children"]; v != "" {
		ac.Children = atoiOr(v, 0)
	}
	ac.CacheDir = cfg.CacheDir
	ac.NoCache = cfg.NoCache
	if ttl := cfg.Extra["cache-ttl"]; ttl != "" {
		if d, err := time.ParseDuration(ttl); err == nil {
			ac.CacheTTL = d
		}
	}
	ac.Refresh = cfg.Extra["refresh"] == "true"
	return NewClient(ac)
}

// Defaults seeds the framework baseline with airbnb's own values, so an unset
// --rate or --timeout uses the airbnb default rather than the generic kit one.
func Defaults(c *kit.Config) {
	def := DefaultConfig()
	c.Rate = def.Delay
	c.Retries = def.Retries
	c.Timeout = def.Timeout
	c.UserAgent = def.UserAgent
}

// Classify turns any accepted input into the canonical (type, id), so `ant
// resolve` and `ant url` touch no network.
func (Domain) Classify(input string) (uriType, id string, err error) {
	r := Classify(input)
	if r.Kind == "unknown" {
		return "", "", errs.Usage("unrecognized airbnb reference: %q", input)
	}
	return r.Kind, r.ID, nil
}

// Locate is the inverse: the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	u := URLFor(uriType, id)
	if u == "" {
		return "", errs.Usage("airbnb has no resource type %q", uriType)
	}
	return u, nil
}

// mapErr translates a library error into a kit error so the exit code matches the
// rest of the fleet: a missing entity reads as "not found" (exit 6), a throttle
// as "rate limited" (exit 5), and the edge wall as "need auth" (exit 4).
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrNotFound):
		return errs.NotFound("%s", err.Error())
	case errors.Is(err, ErrRateLimited):
		return errs.RateLimited("%s", err.Error())
	case errors.Is(err, ErrBlocked):
		return errs.NeedAuth("%s", err.Error())
	default:
		return err
	}
}

// limitOr returns the operator's --limit when set, else the command's own
// default fetch count.
func limitOr(limit, def int) int {
	if limit > 0 {
		return limit
	}
	return def
}
