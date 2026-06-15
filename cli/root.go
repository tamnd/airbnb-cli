// Package cli assembles the airbnb command tree from the airbnb
// domain on top of the any-cli/kit framework.
package cli

import (
	"strconv"

	"github.com/tamnd/airbnb-cli/airbnb"
	"github.com/tamnd/any-cli/kit"
)

// Build metadata, set via -ldflags at release time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// builder holds the domain-global flags while the app is assembled, then folds
// them onto the resolved config in finalize, using the exact keys
// ClientFromConfig reads.
type builder struct {
	userAgent string
	locale    string
	currency  string
	apiKey    string
	checkin   string
	checkout  string
	adults    int
	children  int
	cacheTTL  string
	refresh   bool
}

// NewApp assembles the kit application from the airbnb domain. The domain's
// Register installs the client factory and every operation, so the binary and a
// host (ant, which blank-imports the package) share one source of truth. This
// package adds the domain-global flags and the version command; kit.Run turns the
// App into the CLI, plus the serve and mcp surfaces and the typed-error-to-exit-
// code mapping.
//
// To add a command, declare it in airbnb/domain.go with kit.Handle and it
// appears here automatically. Reach for app.AddCommand only for a verb that does
// not fit the emit-records shape, the way version does below.
func NewApp() *kit.App {
	b := &builder{}
	id := airbnb.Identity()
	id.Version = Version

	app := kit.New(id, kit.WithDefaults(airbnb.Defaults))
	app.GlobalFlags(b.globals)
	app.Finalize(b.finalize)

	airbnb.Domain{}.Register(app)
	app.AddCommand(newVersionCmd())
	return app
}

func (b *builder) globals(f *kit.FlagSet) {
	f.StringVar(&b.userAgent, "user-agent", airbnb.DefaultUserAgent, "User-Agent sent with each request")
	f.StringVar(&b.locale, "locale", "", "locale for localized strings (default en)")
	f.StringVar(&b.currency, "currency", "", "currency for prices (default USD)")
	f.StringVar(&b.apiKey, "api-key", "", "override the public web API key")
	f.StringVar(&b.checkin, "checkin", "", "stay check-in date YYYY-MM-DD (makes a nightly price appear)")
	f.StringVar(&b.checkout, "checkout", "", "stay check-out date YYYY-MM-DD")
	f.IntVar(&b.adults, "adults", 0, "number of adults for the price quote (default 1)")
	f.IntVar(&b.children, "children", 0, "number of children for the price quote")
	f.StringVar(&b.cacheTTL, "cache-ttl", airbnb.DefaultCacheTTL.String(), "how long a cached response stays fresh")
	f.BoolVar(&b.refresh, "refresh", false, "fetch fresh copies and rewrite the cache, ignoring any hit")
}

func (b *builder) finalize(c *kit.Config) {
	if c.Extra == nil {
		c.Extra = map[string]string{}
	}
	set := func(k, v string) {
		if v != "" {
			c.Extra[k] = v
		}
	}
	set("user-agent", b.userAgent)
	set("locale", b.locale)
	set("currency", b.currency)
	set("api-key", b.apiKey)
	set("checkin", b.checkin)
	set("checkout", b.checkout)
	set("cache-ttl", b.cacheTTL)
	if b.adults > 0 {
		c.Extra["adults"] = strconv.Itoa(b.adults)
	}
	if b.children > 0 {
		c.Extra["children"] = strconv.Itoa(b.children)
	}
	if b.refresh {
		c.Extra["refresh"] = "true"
	}
}
