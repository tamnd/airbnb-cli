package airbnb

import (
	"errors"
	"testing"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// These tests are offline: they exercise the URI driver's pure string functions
// (Classify, URLFor, the global-id codec), the error mapping, and the host wiring
// (mint, body, resolve, the graph edges). The client's HTTP behaviour is covered
// in airbnb_test.go.

func TestClassify(t *testing.T) {
	cases := []struct {
		in   string
		kind string
		id   string
		url  string
	}{
		{"12345", "room", "12345", "https://www.airbnb.com/rooms/12345"},
		{"https://www.airbnb.com/rooms/12345", "room", "12345", "https://www.airbnb.com/rooms/12345"},
		{"/rooms/plus/12345", "room", "12345", "https://www.airbnb.com/rooms/12345"},
		{"https://www.airbnb.com/rooms/Cozy-Cabin/12345?check_in=2025-07-01", "room", "12345", "https://www.airbnb.com/rooms/12345"},
		{"https://www.airbnb.com/users/show/555", "host", "555", "https://www.airbnb.com/users/show/555"},
		{"/users/show/555", "host", "555", "https://www.airbnb.com/users/show/555"},
		{"https://www.airbnb.com/experiences/777", "experience", "777", "https://www.airbnb.com/experiences/777"},
		{"", "unknown", "", ""},
		{"https://www.airbnb.com/help/article/123", "unknown", "", ""},
	}
	for _, c := range cases {
		got := Classify(c.in)
		if got.Kind != c.kind || got.ID != c.id || got.URL != c.url {
			t.Errorf("Classify(%q) = {%q %q %q}, want {%q %q %q}",
				c.in, got.Kind, got.ID, got.URL, c.kind, c.id, c.url)
		}
	}
}

// TestClassifyGlobalID proves a pasted GraphQL global id resolves to its kind and
// numeric id without touching the network.
func TestClassifyGlobalID(t *testing.T) {
	cases := []struct{ typeName, id, kind string }{
		{"StayListing", "12345", "room"},
		{"DemandStayListing", "12345", "room"},
		{"User", "555", "host"},
		{"Experience", "777", "experience"},
	}
	for _, c := range cases {
		gid := encodeGlobalID(c.typeName, c.id)
		got := Classify(gid)
		if got.Kind != c.kind || got.ID != c.id {
			t.Errorf("Classify(%s global id) = {%q %q}, want {%q %q}",
				c.typeName, got.Kind, got.ID, c.kind, c.id)
		}
	}
}

func TestDecodeGlobalIDRejectsPlain(t *testing.T) {
	for _, in := range []string{"12345", "/rooms/12345", "not base64!!"} {
		if _, _, ok := decodeGlobalID(in); ok {
			t.Errorf("decodeGlobalID(%q) should not be ok", in)
		}
	}
}

// TestURLForRoundTrip checks that re-classifying a built URL recovers the same
// (kind, id).
func TestURLForRoundTrip(t *testing.T) {
	cases := []struct{ kind, id string }{
		{"room", "12345"},
		{"host", "555"},
		{"experience", "777"},
	}
	for _, c := range cases {
		u := URLFor(c.kind, c.id)
		if u == "" {
			t.Errorf("URLFor(%q,%q) empty", c.kind, c.id)
			continue
		}
		got := Classify(u)
		if got.Kind != c.kind || got.ID != c.id {
			t.Errorf("round-trip %q: Classify(%q) = {%q %q}, want {%q %q}",
				c.kind, u, got.Kind, got.ID, c.kind, c.id)
		}
	}
}

func TestURLForUnknown(t *testing.T) {
	if u := URLFor("nonsense", "x"); u != "" {
		t.Errorf("URLFor of an unknown kind should be empty, got %q", u)
	}
}

func TestMapErr(t *testing.T) {
	cases := []struct {
		err  error
		kind errs.Kind
	}{
		{ErrNotFound, errs.KindNotFound},
		{ErrRateLimited, errs.KindRateLimited},
		{ErrBlocked, errs.KindNeedAuth},
	}
	for _, c := range cases {
		got := mapErr(c.err)
		if errs.KindOf(got) != c.kind {
			t.Errorf("mapErr(%v) kind = %v, want %v", c.err, errs.KindOf(got), c.kind)
		}
	}
	if mapErr(nil) != nil {
		t.Error("mapErr(nil) should be nil")
	}
	plain := errors.New("boom")
	if got := mapErr(plain); got != plain {
		t.Errorf("mapErr passes an unmapped error through unchanged, got %v", got)
	}
}

func TestDomainClassify(t *testing.T) {
	d := Domain{}
	kind, id, err := d.Classify("https://www.airbnb.com/rooms/Cozy-Cabin/12345")
	if err != nil {
		t.Fatal(err)
	}
	if kind != "room" || id != "12345" {
		t.Errorf("Domain.Classify = {%q %q}", kind, id)
	}
	if _, _, err := d.Classify("https://www.airbnb.com/help/article/123"); err == nil {
		t.Error("Domain.Classify of an unknown reference should error")
	}
}

func TestDomainLocate(t *testing.T) {
	d := Domain{}
	u, err := d.Locate("host", "555")
	if err != nil {
		t.Fatal(err)
	}
	if u != "https://www.airbnb.com/users/show/555" {
		t.Errorf("Locate = %q", u)
	}
	if _, err := d.Locate("nonsense", "x"); err == nil {
		t.Error("Locate of an unknown kind should error")
	}
}

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "airbnb" {
		t.Errorf("scheme = %q", info.Scheme)
	}
	if info.Identity.Binary != "airbnb" {
		t.Errorf("identity binary = %q", info.Identity.Binary)
	}
	want := map[string]bool{"www.airbnb.com": true, "airbnb.com": true}
	for _, h := range info.Hosts {
		delete(want, h)
	}
	if len(want) != 0 {
		t.Errorf("missing hosts: %v", want)
	}
}

// TestHostWiring mounts the driver in a kit Host (the runtime ant drives) and
// checks the round trip: a room record mints to its URI, its body is readable,
// and a bare id resolves back to the same URI. The init in domain.go registers
// the domain, so kit.Open finds it.
func TestHostWiring(t *testing.T) {
	h, err := kit.Open()
	if err != nil {
		t.Fatal(err)
	}

	r := &Room{ID: "12345", URL: BaseURL + "/rooms/12345", Name: "Cozy Cabin", Description: "A lovely cabin."}
	u, err := h.Mint(r)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	if want := "airbnb://room/12345"; u.String() != want {
		t.Errorf("Mint = %q, want %q", u.String(), want)
	}

	if body, ok := h.Body(r); !ok || body == "" {
		t.Errorf("Body = (%q, %v), want non-empty", body, ok)
	}

	got, err := h.ResolveOn("airbnb", "12345")
	if err != nil || got.String() != "airbnb://room/12345" {
		t.Errorf("ResolveOn = (%q, %v), want airbnb://room/12345", got.String(), err)
	}
}

// TestHostLinks proves the graph edges a host walks for BFS, the edges `ant
// export --follow` traverses to reconstruct the public site from one seed:
//
//	place --> search; search listing --> room, host; room --> host, reviews,
//	calendar; review --> room, the reviewer's profile; day --> room; host -->
//	the host's listings.
//
// Following all of them leaves no node without an outward edge, so a crawl
// started anywhere reaches the rest of the reachable graph.
func TestHostLinks(t *testing.T) {
	h, err := kit.Open()
	if err != nil {
		t.Fatal(err)
	}

	links := func(rec any) []string {
		var out []string
		for _, u := range h.Links(rec) {
			out = append(out, u.String())
		}
		return out
	}
	has := func(t *testing.T, got []string, want string) {
		t.Helper()
		for _, g := range got {
			if g == want {
				return
			}
		}
		t.Errorf("missing edge %q in %v", want, got)
	}

	// The entry edges: a suggestion fans out into both a stay search and an
	// experience search for the place.
	pl := links(&Place{Name: "Lake Tahoe", SearchRef: "Lake Tahoe", ExperiencesRef: "Lake Tahoe"})
	has(t, pl, "airbnb://search/Lake%20Tahoe")
	has(t, pl, "airbnb://experiences/Lake%20Tahoe")

	// A search card walks straight through to its full room and its host.
	l := &Listing{ID: "111", Room: "111", Host: "555"}
	ll := links(l)
	has(t, ll, "airbnb://room/111")
	has(t, ll, "airbnb://host/555")

	// A room reaches its host, its reviews, and its calendar.
	rm := links(&Room{ID: "111", HostID: "555", ReviewsRef: "111", CalendarRef: "111"})
	has(t, rm, "airbnb://host/555")
	has(t, rm, "airbnb://reviews/111")
	has(t, rm, "airbnb://calendar/111")

	// A review reaches its listing and the reviewer's profile.
	rv := links(&Review{ID: "r1", Room: "111", AuthorID: "888"})
	has(t, rv, "airbnb://room/111")
	has(t, rv, "airbnb://host/888")

	has(t, links(&Day{ID: "111:2025-07-01", Room: "111"}), "airbnb://room/111")

	// A host reaches the host's own listings, so a crawl never dead-ends there.
	has(t, links(&Host{ID: "555", ListingsRef: "555"}), "airbnb://listings/555")
}
