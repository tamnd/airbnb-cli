package airbnb

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// These tests run the client against httptest fakes. Airbnb fronts every real
// surface with an edge bot manager that a datacenter IP cannot pass, so live
// verification is impossible from here; the fakes serve the exact island and
// GraphQL shapes the per-surface decoders read, the same way walmart-cli tests
// the offline path. The wall behaviour (403, 429, a challenge body) is exercised
// against bare-status and stub servers.

// newClient points a client at a fake server with no pacing, no retries, and no
// cache, so each surface hits the network once and the wall surfaces directly.
func testClient(ts *httptest.Server) *Client {
	cfg := DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Delay = 0
	cfg.Retries = 0
	cfg.NoCache = true
	return NewClient(cfg)
}

// serve replies with body for every request.
func serve(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
}

// status replies with a bare status code for every request.
func status(code int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
	}))
}

// gqlServe wraps data in a GraphQL envelope and returns it for every request,
// asserting the public api key rides along the way the web client sends it.
func gqlServe(t *testing.T, data string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Airbnb-Api-Key") == "" {
			t.Error("gql request carried no X-Airbnb-Api-Key")
		}
		_, _ = w.Write([]byte(`{"data":` + data + `}`))
	}))
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("recovered"))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Delay = 0
	cfg.Retries = 5
	cfg.NoCache = true
	c := NewClient(cfg)

	start := time.Now()
	body, err := c.get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "recovered" {
		t.Errorf("body = %q after retries", body)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

// roomPage mirrors a real /rooms/ page: the whole client state in the
// data-deferred-state-0 island, with the typed sections this tool reads and the
// flat numeric facts in the metadata blob.
const roomPage = `<html><head>
<script id="data-deferred-state-0" type="application/json">
{"niobeMinimalClientData":[["StaysPdpSections",{"data":{"presentation":{"stayProductDetailPage":{"sections":{
  "sections":[
    {"sectionId":"TITLE","section":{"__typename":"PdpTitleSection","title":"Cozy Cabin"}},
    {"sectionId":"DESC","section":{"__typename":"PdpDescriptionSection","htmlDescription":{"htmlText":"<p>A lovely cabin in the woods.</p>"}}},
    {"sectionId":"OVERVIEW","section":{"__typename":"PdpOverviewV2Section","detailItems":[{"title":"4 guests"},{"title":"2 bedrooms"},{"title":"3 beds"},{"title":"1.5 baths"}]}},
    {"sectionId":"AMENITIES","section":{"__typename":"AmenitiesSection","seeAllAmenitiesGroups":[{"amenities":[{"title":"Wifi","available":true},{"title":"Kitchen","available":true}]}]}},
    {"sectionId":"PHOTOS","section":{"__typename":"PhotoTourModalSection","mediaItems":[{"baseUrl":"https://img/1.jpg"},{"baseUrl":"https://img/2.jpg"}]}},
    {"sectionId":"HOST","section":{"__typename":"HostProfileSection","hostName":"Jordan","hostId":"555","isSuperhost":true}},
    {"sectionId":"POLICIES","section":{"__typename":"PoliciesSection","houseRules":[{"title":"No smoking"},{"title":"No parties"}]}}
  ],
  "metadata":{"loggingContext":{"eventDataLogging":{"listingLat":39.1,"listingLng":-120.1,"personCapacity":4,"accuracyRating":4.9,"checkinRating":5.0,"cleanlinessRating":4.8,"communicationRating":5.0,"locationRating":4.7,"valueRating":4.6,"guestSatisfactionOverall":4.85,"visibleReviewCount":120,"roomType":"Entire home","propertyType":"Cabin","isSuperhost":true,"hostId":"555"}}}
}}}}}]]}
</script></head><body>room</body></html>`

func TestGetRoom(t *testing.T) {
	ts := serve(roomPage)
	defer ts.Close()

	r, err := testClient(ts).GetRoom(context.Background(), "999")
	if err != nil {
		t.Fatal(err)
	}
	if r.ID != "999" || r.Name != "Cozy Cabin" {
		t.Errorf("room = %+v", r)
	}
	if r.Description != "A lovely cabin in the woods." {
		t.Errorf("description = %q", r.Description)
	}
	if r.Capacity != 4 || r.Bedrooms != 2 || r.Beds != 3 || r.Bathrooms != "1.5 baths" {
		t.Errorf("overview = cap %d br %d beds %d bath %q", r.Capacity, r.Bedrooms, r.Beds, r.Bathrooms)
	}
	if len(r.Amenities) != 2 || r.Amenities[0] != "Wifi" {
		t.Errorf("amenities = %v", r.Amenities)
	}
	if len(r.Images) != 2 || r.Image != "https://img/1.jpg" {
		t.Errorf("images = %v (first %q)", r.Images, r.Image)
	}
	if r.HostName != "Jordan" || r.HostID != "555" || !r.Superhost {
		t.Errorf("host = %q / %q / %v", r.HostName, r.HostID, r.Superhost)
	}
	if len(r.HouseRules) != 2 || r.HouseRules[0] != "No smoking" {
		t.Errorf("house rules = %v", r.HouseRules)
	}
	if r.Rating != 4.85 || r.Reviews != 120 {
		t.Errorf("rating/reviews = %v / %d", r.Rating, r.Reviews)
	}
	if r.Accuracy != 4.9 || r.Cleanliness != 4.8 || r.Value != 4.6 {
		t.Errorf("category ratings = %+v", r)
	}
	if r.RoomType != "Entire home" || r.PropertyType != "Cabin" {
		t.Errorf("room/property type = %q / %q", r.RoomType, r.PropertyType)
	}
	if r.Lat != 39.1 || r.Lng != -120.1 {
		t.Errorf("coords = %v, %v", r.Lat, r.Lng)
	}
	// Without dates the nightly price is left out, not invented, and the currency
	// falls back to the configured default.
	if r.Price != 0 {
		t.Errorf("price without dates = %v, want 0", r.Price)
	}
	if r.Currency != "USD" {
		t.Errorf("currency = %q", r.Currency)
	}
	if r.URL != BaseURL+"/rooms/999" {
		t.Errorf("url = %q", r.URL)
	}
}

func TestGetRoomNoIslandIsBlocked(t *testing.T) {
	ts := serve(`<html><body>nothing useful here</body></html>`)
	defer ts.Close()

	_, err := testClient(ts).GetRoom(context.Background(), "999")
	if !errors.Is(err, ErrBlocked) {
		t.Errorf("a page without the island should be ErrBlocked, got %v", err)
	}
}

const searchData = `{"presentation":{"staysSearch":{"results":{
  "searchResults":[
    {"__typename":"StaySearchResult","listing":{"id":"111","name":"Cozy Cabin","title":"Entire cabin in Tahoe","coordinate":{"latitude":39.1,"longitude":-120.1},"avgRatingA11yLabel":"4.95 out of 5 average rating, 120 reviews","roomTypeCategory":"entire_home","contextualPictures":[{"picture":"https://a/1.jpg"}],"formattedBadges":[{"text":"Guest favorite"}],"hostId":"555"},
     "pricingQuote":{"structuredStayDisplayPrice":{"primaryLine":{"price":"$120 night"},"secondaryLine":{"price":"$640 total"}}}},
    {"__typename":"StaySearchResult","listing":{"id":""}}
  ],
  "paginationInfo":{"nextPageCursor":""}
}}}}`

func TestSearch(t *testing.T) {
	ts := gqlServe(t, searchData)
	defer ts.Close()

	got, err := testClient(ts).Search(context.Background(), "Lake Tahoe", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 listing (the id-less card dropped), got %d", len(got))
	}
	l := got[0]
	if l.ID != "111" || l.Name != "Cozy Cabin" || l.Title != "Entire cabin in Tahoe" {
		t.Errorf("listing = %+v", l)
	}
	if l.RoomType != "Entire home" {
		t.Errorf("room type label = %q", l.RoomType)
	}
	if l.Rating != 4.95 || l.Reviews != 120 {
		t.Errorf("rating/reviews from label = %v / %d", l.Rating, l.Reviews)
	}
	if l.Price != 120 || l.Total != 640 {
		t.Errorf("price/total = %v / %v", l.Price, l.Total)
	}
	if l.Image != "https://a/1.jpg" {
		t.Errorf("image = %q", l.Image)
	}
	if len(l.Badges) != 1 || l.Badges[0] != "Guest favorite" {
		t.Errorf("badges = %v", l.Badges)
	}
	// Each card links straight through to its full listing and to its host.
	if l.Room != "111" || l.Host != "555" {
		t.Errorf("edges room/host = %q / %q", l.Room, l.Host)
	}
	if l.URL != BaseURL+"/rooms/111" {
		t.Errorf("url = %q", l.URL)
	}
}

const reviewsData = `{"presentation":{"stayProductDetailPage":{"reviews":{"reviews":[
  {"id":"r1","comments":"Great place","language":"en","localizedDate":"April 2025","rating":5,"reviewer":{"firstName":"Alice","location":"Seattle, WA"},"response":"Thanks!"}
]}}}}`

func TestReviews(t *testing.T) {
	ts := gqlServe(t, reviewsData)
	defer ts.Close()

	got, err := testClient(ts).Reviews(context.Background(), "999", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 review, got %d", len(got))
	}
	rv := got[0]
	if rv.ID != "r1" || rv.Author != "Alice" || rv.Location != "Seattle, WA" {
		t.Errorf("review = %+v", rv)
	}
	if rv.Date != "April 2025" || rv.Rating != 5 || rv.Language != "en" {
		t.Errorf("date/rating/lang = %q / %d / %q", rv.Date, rv.Rating, rv.Language)
	}
	if rv.Text != "Great place" || rv.Response != "Thanks!" {
		t.Errorf("text/response = %q / %q", rv.Text, rv.Response)
	}
	// Every review links back to its listing for BFS.
	if rv.Room != "999" {
		t.Errorf("room edge = %q", rv.Room)
	}
}

const calendarData = `{"merlin":{"pdpAvailabilityCalendar":{"calendarMonths":[
  {"days":[
    {"calendarDate":"2025-07-01","available":true,"minNights":2,"maxNights":30,"price":{"localPriceFormatted":"$150","localCurrency":"USD"}},
    {"calendarDate":"2025-07-02","available":false}
  ]}
]}}}`

func TestCalendar(t *testing.T) {
	ts := gqlServe(t, calendarData)
	defer ts.Close()

	got, err := testClient(ts).Calendar(context.Background(), "999", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 days, got %d", len(got))
	}
	d := got[0]
	if d.ID != "999:2025-07-01" || d.Date != "2025-07-01" {
		t.Errorf("day id/date = %q / %q", d.ID, d.Date)
	}
	if !d.Available || d.MinNights != 2 || d.MaxNights != 30 {
		t.Errorf("availability = %+v", d)
	}
	if d.Price != 150 || d.Currency != "USD" {
		t.Errorf("price = %v %q", d.Price, d.Currency)
	}
	if d.Room != "999" {
		t.Errorf("room edge = %q", d.Room)
	}
	if got[1].Available {
		t.Error("second day should be unavailable")
	}
}

const hostData = `{"presentation":{"userProfileContainer":{"userProfile":{
  "id":"555","smartName":"Jordan","isSuperhost":true,"createdAt":"2015","about":"I love hosting.","languages":["English","Spanish"],"listingsCount":3,"reviewsCount":210,"responseRate":"100%","identityVerified":true,"pictureUrl":"https://h/p.jpg"
}}}}`

func TestGetHost(t *testing.T) {
	ts := gqlServe(t, hostData)
	defer ts.Close()

	h, err := testClient(ts).GetHost(context.Background(), "555")
	if err != nil {
		t.Fatal(err)
	}
	if h.ID != "555" || h.Name != "Jordan" || !h.Superhost {
		t.Errorf("host = %+v", h)
	}
	if h.Since != "2015" || h.About != "I love hosting." || h.ResponseRate != "100%" {
		t.Errorf("since/about/rate = %q / %q / %q", h.Since, h.About, h.ResponseRate)
	}
	if len(h.Languages) != 2 || h.Listings != 3 || h.Reviews != 210 || !h.Verified {
		t.Errorf("languages/listings/reviews/verified = %v / %d / %d / %v", h.Languages, h.Listings, h.Reviews, h.Verified)
	}
	if h.URL != BaseURL+"/users/show/555" {
		t.Errorf("url = %q", h.URL)
	}
}

const hostListingsData = `{"beehive":{"getListOfListings":{"listings":[
  {"id":"111","name":"Cozy Cabin","roomType":"Entire home","avgRating":4.9,"reviewCount":120,"pictureUrl":"https://a/1.jpg","lat":39.1,"lng":-120.1}
]}}}`

func TestHostListings(t *testing.T) {
	ts := gqlServe(t, hostListingsData)
	defer ts.Close()

	got, err := testClient(ts).HostListings(context.Background(), "555", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 listing, got %d", len(got))
	}
	l := got[0]
	if l.ID != "111" || l.Name != "Cozy Cabin" || l.RoomType != "Entire home" {
		t.Errorf("listing = %+v", l)
	}
	if l.Rating != 4.9 || l.Reviews != 120 {
		t.Errorf("rating/reviews = %v / %d", l.Rating, l.Reviews)
	}
	// The listing links to its full room and back to the queried host.
	if l.Room != "111" || l.Host != "555" {
		t.Errorf("edges room/host = %q / %q", l.Room, l.Host)
	}
}

const experiencesData = `{"presentation":{"experiencesSearch":{"results":{"searchResults":[
  {"id":"777","title":"Kayak tour","hostName":"Sam","displayPrice":"$80","starRating":4.8,"reviewCount":45,"locationName":"Lake Tahoe","lat":39.0,"lng":-120.0,"pictureUrl":"https://e/1.jpg"}
],"paginationInfo":{"nextPageCursor":""}}}}}`

func TestExperiences(t *testing.T) {
	ts := gqlServe(t, experiencesData)
	defer ts.Close()

	got, err := testClient(ts).Experiences(context.Background(), "Lake Tahoe", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 experience, got %d", len(got))
	}
	e := got[0]
	if e.ID != "777" || e.Title != "Kayak tour" || e.Host != "Sam" {
		t.Errorf("experience = %+v", e)
	}
	if e.Price != 80 || e.Rating != 4.8 || e.Reviews != 45 {
		t.Errorf("price/rating/reviews = %v / %v / %d", e.Price, e.Rating, e.Reviews)
	}
	if e.Location != "Lake Tahoe" || e.URL != BaseURL+"/experiences/777" {
		t.Errorf("location/url = %q / %q", e.Location, e.URL)
	}
}

const suggestData = `{"autocomplete_terms":[
  {"display_name":"Paris, France","location":{"location_name":"Paris, France","google_place_id":"ChIJ","lat":48.8,"lng":2.3}},
  {"display_name":"Paris, TX","location":{"location_name":"Paris, TX","google_place_id":"ChIJ2","lat":33.6,"lng":-95.5}}
]}`

func TestSuggest(t *testing.T) {
	ts := serve(suggestData)
	defer ts.Close()

	got, err := testClient(ts).Suggest(context.Background(), "Paris", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 suggestions, got %d", len(got))
	}
	if got[0].Name != "Paris, France" || got[0].Query != "Paris" || got[0].PlaceID != "ChIJ" {
		t.Errorf("suggestion 0 = %+v", got[0])
	}
	if got[0].Lat != 48.8 || got[0].Lng != 2.3 {
		t.Errorf("coords = %v, %v", got[0].Lat, got[0].Lng)
	}
	if got[1].Name != "Paris, TX" {
		t.Errorf("suggestion 1 = %+v", got[1])
	}
}

func TestSearchWallForbidden(t *testing.T) {
	ts := status(http.StatusForbidden)
	defer ts.Close()

	_, err := testClient(ts).Search(context.Background(), "Lake Tahoe", 10)
	if !errors.Is(err, ErrBlocked) {
		t.Errorf("403 should be ErrBlocked, got %v", err)
	}
}

func TestChallengeBodyIsBlocked(t *testing.T) {
	// DataDome serves the challenge with a 200 status as often as a 403, so the
	// body is what gives it away.
	ts := serve(`<html><body>please verify at captcha-delivery.com</body></html>`)
	defer ts.Close()

	_, err := testClient(ts).GetRoom(context.Background(), "999")
	if !errors.Is(err, ErrBlocked) {
		t.Errorf("challenge body should be ErrBlocked, got %v", err)
	}
}

func TestRoomNotFound(t *testing.T) {
	ts := status(http.StatusNotFound)
	defer ts.Close()

	_, err := testClient(ts).GetRoom(context.Background(), "999")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("404 should be ErrNotFound, got %v", err)
	}
}

func TestSearchRateLimited(t *testing.T) {
	ts := status(http.StatusTooManyRequests)
	defer ts.Close()

	_, err := testClient(ts).Search(context.Background(), "Lake Tahoe", 10)
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("429 should be ErrRateLimited, got %v", err)
	}
}

// TestGQLErrorEnvelopeIsBlocked proves a 200 carrying a GraphQL errors envelope
// (a stale persisted-query hash or a rejected key surfacing through the edge)
// reads as the wall, never as a fabricated record.
func TestGQLErrorEnvelopeIsBlocked(t *testing.T) {
	ts := serve(`{"errors":[{"message":"invalid api key"}]}`)
	defer ts.Close()

	_, err := testClient(ts).Search(context.Background(), "Lake Tahoe", 10)
	if !errors.Is(err, ErrBlocked) {
		t.Errorf("an unclassified gql error should be ErrBlocked, got %v", err)
	}
}
