package airbnb

import "errors"

// The library reports its outcomes as a few sentinel errors. domain.go's mapErr
// translates each into the kit error kind that carries the matching exit code,
// so the standalone binary and a host agree on what a wall, a throttle, and a
// miss mean.
var (
	// ErrNotFound is a missing entity: an unknown room, user, or experience.
	// Airbnb serves these as a 404 or a not-found GraphQL envelope. Exit code 6.
	ErrNotFound = errors.New("not found")

	// ErrRateLimited is a sustained HTTP 429 after the client's own retries. Slow
	// down with --rate. Exit code 5.
	ErrRateLimited = errors.New("rate limited")

	// ErrBlocked is Airbnb's edge bot wall: Cloudflare or DataDome classifies the
	// request on IP reputation and TLS fingerprint and answers with a bodyless
	// 403, a challenge body, or a connection reset, before the application sees
	// it. A GraphQL reply whose errors name an auth or rate problem (a stale
	// persisted-query hash or a rejected key) reads here too. The data is public
	// and needs no account, so the message names the real remedy: a residential
	// or mobile connection. This tool does not forge a TLS fingerprint or rent an
	// IP to get past the edge. Exit code 4.
	ErrBlocked = errors.New("blocked by Airbnb's edge bot wall (the data is public and needs no " +
		"account; retry from a residential or mobile connection)")
)
