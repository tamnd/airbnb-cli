package airbnb

import (
	"context"
	"strconv"

	"github.com/tamnd/any-cli/kit/errs"
)

// ops.go holds the handler for every operation declared in domain.go. kit
// reflects each input struct into CLI flags, HTTP query params, and MCP tool
// arguments: kit:"arg" is a positional, kit:"flag,inherit" binds the shared
// --limit, and kit:"inject" receives the client newClient builds. The reference
// ops (id, url) take no client; they run offline.

// --- top-level reads ---

type placeIn struct {
	Place  string  `kit:"arg" help:"a city, region, or place name"`
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func search(ctx context.Context, in placeIn, emit func(*Listing) error) error {
	items, err := in.Client.Search(ctx, in.Place, limitOr(in.Limit, defaultPageSize))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(items, emit)
}

func experiences(ctx context.Context, in placeIn, emit func(*Experience) error) error {
	items, err := in.Client.Experiences(ctx, in.Place, limitOr(in.Limit, defaultPageSize))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(items, emit)
}

type roomRef struct {
	ID     string  `kit:"arg" help:"room id or /rooms/ URL"`
	Client *Client `kit:"inject"`
}

func getRoom(ctx context.Context, in roomRef, emit func(*Room) error) error {
	r, err := in.Client.GetRoom(ctx, in.ID)
	if err != nil {
		return mapErr(err)
	}
	return emit(r)
}

type roomListIn struct {
	ID     string  `kit:"arg" help:"room id or /rooms/ URL"`
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func reviews(ctx context.Context, in roomListIn, emit func(*Review) error) error {
	rs, err := in.Client.Reviews(ctx, in.ID, limitOr(in.Limit, reviewsPage))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(rs, emit)
}

func calendar(ctx context.Context, in roomListIn, emit func(*Day) error) error {
	days, err := in.Client.Calendar(ctx, in.ID, limitOr(in.Limit, 0))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(days, emit)
}

type prefixIn struct {
	Prefix string  `kit:"arg" help:"the typed prefix"`
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func suggest(ctx context.Context, in prefixIn, emit func(*Place) error) error {
	ps, err := in.Client.Suggest(ctx, in.Prefix, limitOr(in.Limit, 0))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(ps, emit)
}

// --- host ---

type hostRef struct {
	ID     string  `kit:"arg" help:"user id or /users/show/ URL"`
	Client *Client `kit:"inject"`
}

type hostListIn struct {
	ID     string  `kit:"arg" help:"user id or /users/show/ URL"`
	Limit  int     `kit:"flag,inherit"`
	Client *Client `kit:"inject"`
}

func getHost(ctx context.Context, in hostRef, emit func(*Host) error) error {
	h, err := in.Client.GetHost(ctx, in.ID)
	if err != nil {
		return mapErr(err)
	}
	return emit(h)
}

func hostListings(ctx context.Context, in hostListIn, emit func(*Listing) error) error {
	items, err := in.Client.HostListings(ctx, in.ID, limitOr(in.Limit, 0))
	if err != nil {
		return mapErr(err)
	}
	return emitAll(items, emit)
}

// --- reference tools (offline) ---

type refIn struct {
	Ref string `kit:"arg" help:"any Airbnb URL, path, id, or global id"`
}

func classifyRef(_ context.Context, in refIn, emit func(*Ref) error) error {
	r := Classify(in.Ref)
	if r.Kind == "unknown" {
		return errs.Usage("unrecognized airbnb reference: %q", in.Ref)
	}
	return emit(&r)
}

type urlIn struct {
	Kind string `kit:"arg" help:"room, host, or experience"`
	ID   string `kit:"arg" help:"the id for that kind"`
}

func buildURL(_ context.Context, in urlIn, emit func(*Ref) error) error {
	u := URLFor(in.Kind, in.ID)
	if u == "" {
		return errs.Usage("airbnb has no resource type %q", in.Kind)
	}
	return emit(&Ref{Input: in.Kind + "/" + in.ID, Kind: in.Kind, ID: in.ID, URL: u})
}

// emitAll streams a slice of records through emit.
func emitAll[T any](items []*T, emit func(*T) error) error {
	for _, it := range items {
		if err := emit(it); err != nil {
			return err
		}
	}
	return nil
}

// atoiOr parses s as an int, returning def on failure.
func atoiOr(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}
