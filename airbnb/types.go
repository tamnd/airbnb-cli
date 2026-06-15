package airbnb

// This file holds the exported records the commands emit. Their json tags name
// the fields a reader sees, kit:"id" marks the key the record store upserts on,
// kit:"body" marks the long-text field `airbnb cat` and the Markdown export
// print, and table:",truncate" keeps wide free text from blowing up a terminal
// table. Each record carries only fields a logged-out reader can actually fill:
// no trips, no messages, no wishlists, no host management, no payout data,
// because none of that is reachable without a signed-in account. There is no
// Rank column either; emit order is the rank. Several records are emitted by more
// than one surface, and omitempty carries the gaps. The per-surface files hold
// the parsing these map from.
//
// The kit:"link" edges connect the records into one graph a host walks for
// breadth-first crawls, and they are what lets a crawl reconstruct the public
// site from a single seed. A resolver edge (room, host, the reviewer) names a
// bare field and points at one record; a collection edge carries the parent id
// under a <name>_ref field and points at a list authority. Following all of them
// closes the loop:
//
//	place(suggest) --search_ref------> search
//	place(suggest) --experiences_ref-> experiences
//	search --------> listing --room--> room
//	                 listing --host--> host
//	room ----host_id-----> host
//	room ----reviews_ref-> reviews ---> review --room--> room
//	room ----calendar_ref-> calendar --> day ----room--> room
//	review --author_id---> host (the reviewer's profile)
//	host ----listings_ref-> listings --> listing ...
//
// so suggest -> search -> every listing -> its room -> its host, reviews, and
// calendar -> each reviewer's profile -> each host's other listings, plus suggest
// -> experiences for the same place, with no node left without an outward edge.

// Listing is a summary card in a grid, emitted by search and host listings. Its
// id is the room id, so Room carries the same value as the graph edge to the full
// listing: a host walks a search result straight through to airbnb://room/<id>.
type Listing struct {
	ID       string   `json:"id" kit:"id"` // room id
	Name     string   `json:"name,omitempty" table:",truncate"`
	Title    string   `json:"title,omitempty" table:",truncate"` // the card's title line
	RoomType string   `json:"room_type,omitempty"`               // Entire home, Private room, ...
	Price    float64  `json:"price,omitempty"`                   // nightly, for the searched dates
	Original float64  `json:"original,omitempty"`                // pre-discount nightly, when the card strikes one through
	Currency string   `json:"currency,omitempty"`
	Total    float64  `json:"total,omitempty"` // total for the stay, when dates are given
	Rating   float64  `json:"rating,omitempty"`
	Reviews  int      `json:"reviews,omitempty"`
	Lat      float64  `json:"lat,omitempty"`
	Lng      float64  `json:"lng,omitempty"`
	Badges   []string `json:"badges,omitempty" table:"-"`        // Guest favorite, Superhost, ...
	Images   []string `json:"images,omitempty" table:"-"`        // the card's photo carousel
	Image    string   `json:"image,omitempty" table:",truncate"` // the first photo
	URL      string   `json:"url"`
	Room     string   `json:"room,omitempty" table:"-" kit:"link,kind=airbnb/room"` // edge to the full listing (= id)
	Host     string   `json:"host,omitempty" table:"-" kit:"link,kind=airbnb/host"` // edge to the host, when the card carries it
}

// Room is the full detail for one listing, emitted by room. It is parsed from the
// PDP JSON island, with its date-specific price filled from StaysPdpSections when
// check-in and check-out are given. The reviews_ref and calendar_ref edges carry
// the room id so a crawl reaching a room expands to its reviews and its calendar.
type Room struct {
	ID             string   `json:"id" kit:"id"` // room id
	Name           string   `json:"name,omitempty" table:",truncate"`
	Description    string   `json:"description,omitempty" table:",truncate" kit:"body"`
	Highlights     []string `json:"highlights,omitempty" table:"-"` // the icon highlights, e.g. "Self check-in"
	PropertyType   string   `json:"property_type,omitempty"`        // House, Apartment, ...
	RoomType       string   `json:"room_type,omitempty"`            // Entire home, Private room, ...
	Capacity       int      `json:"capacity,omitempty"`             // person capacity
	Bedrooms       int      `json:"bedrooms,omitempty"`
	Beds           int      `json:"beds,omitempty"`
	Bathrooms      string   `json:"bathrooms,omitempty"`          // "1.5 baths"; text, since half-baths are common
	Sleeping       []string `json:"sleeping,omitempty" table:"-"` // sleeping arrangements, e.g. "Bedroom 1: 1 queen bed"
	Amenities      []string `json:"amenities,omitempty" table:"-"`
	Price          float64  `json:"price,omitempty"` // nightly, when dates are given
	Currency       string   `json:"currency,omitempty"`
	Rating         float64  `json:"rating,omitempty"`   // overall guest-satisfaction rating
	Accuracy       float64  `json:"accuracy,omitempty"` // the six category ratings
	CheckinRating  float64  `json:"checkin_rating,omitempty"`
	Cleanliness    float64  `json:"cleanliness,omitempty"`
	Communication  float64  `json:"communication,omitempty"`
	LocationRating float64  `json:"location_rating,omitempty"`
	Value          float64  `json:"value,omitempty"`
	Reviews        int      `json:"reviews,omitempty"`  // visible review count
	Location       string   `json:"location,omitempty"` // the listing's area, e.g. "Tahoe Vista, California"
	Lat            float64  `json:"lat,omitempty"`
	Lng            float64  `json:"lng,omitempty"`
	CheckIn        string   `json:"check_in,omitempty"`  // the check-in window, when the rules state one
	CheckOut       string   `json:"check_out,omitempty"` // the checkout time, when stated
	HouseRules     []string `json:"house_rules,omitempty" table:"-"`
	Superhost      bool     `json:"superhost,omitempty"`
	HostID         string   `json:"host_id,omitempty" table:"-" kit:"link,kind=airbnb/host"` // edge to the host
	HostName       string   `json:"host_name,omitempty"`
	HostImage      string   `json:"host_image,omitempty" table:"-"`    // the host's public profile photo
	Images         []string `json:"images,omitempty" table:"-"`        // the photo gallery
	Image          string   `json:"image,omitempty" table:",truncate"` // the first photo
	URL            string   `json:"url"`
	ReviewsRef     string   `json:"reviews_ref,omitempty" table:"-" kit:"link,kind=airbnb/reviews"`   // edge to this listing's reviews (= id)
	CalendarRef    string   `json:"calendar_ref,omitempty" table:"-" kit:"link,kind=airbnb/calendar"` // edge to this listing's calendar (= id)
}

// Review is one review of a listing, emitted by reviews. AuthorID, when the
// payload carries it, is the reviewer's user id, so a crawl can walk a review to
// the reviewer's public profile.
type Review struct {
	ID          string `json:"id" kit:"id"`
	Author      string `json:"author,omitempty"`
	AuthorID    string `json:"author_id,omitempty" table:"-" kit:"link,kind=airbnb/host"` // edge to the reviewer's profile
	AuthorImage string `json:"author_image,omitempty" table:"-"`
	Location    string `json:"location,omitempty"` // the reviewer's stated home, when shown
	Date        string `json:"date,omitempty"`     // the localized review date
	Trip        string `json:"trip,omitempty"`     // the stay descriptor, e.g. "Stayed a few nights"
	Rating      int    `json:"rating,omitempty"`   // the reviewer's star rating, when present
	Text        string `json:"text,omitempty" table:",truncate" kit:"body"`
	Response    string `json:"response,omitempty" table:",truncate"` // the host's public response, when present
	Language    string `json:"language,omitempty"`
	Room        string `json:"room,omitempty" table:"-" kit:"link,kind=airbnb/room"` // edge back to the listing
}

// Day is one day on a listing's availability calendar, emitted by calendar. The
// id is the room id joined to the date, so the store keeps one row per
// listing-day and a re-fetch upserts cleanly. Available is the day's calendar
// state; Bookable reports whether a stay may actually start or span it.
type Day struct {
	ID        string  `json:"id" kit:"id"` // "<roomid>:<date>"
	Date      string  `json:"date"`        // YYYY-MM-DD
	Available bool    `json:"available"`
	Bookable  bool    `json:"bookable,omitempty"`
	MinNights int     `json:"min_nights,omitempty"`
	MaxNights int     `json:"max_nights,omitempty"`
	Price     float64 `json:"price,omitempty"` // nightly price for this date, when published
	Currency  string  `json:"currency,omitempty"`
	Room      string  `json:"room,omitempty" table:"-" kit:"link,kind=airbnb/room"` // edge back to the listing
}

// Host is a host/user public profile, emitted by host show. The listings_ref edge
// carries the user id so a crawl reaching a host expands to the host's listings.
type Host struct {
	ID           string   `json:"id" kit:"id"` // user id
	Name         string   `json:"name,omitempty"`
	Superhost    bool     `json:"superhost,omitempty"`
	Since        string   `json:"since,omitempty"`    // the "joined in <year>" text
	Location     string   `json:"location,omitempty"` // the host's stated home, e.g. "Lives in Paris"
	About        string   `json:"about,omitempty" table:",truncate" kit:"body"`
	ResponseRate string   `json:"response_rate,omitempty"`
	ResponseTime string   `json:"response_time,omitempty"` // "within an hour", ...
	Languages    []string `json:"languages,omitempty" table:"-"`
	Listings     int      `json:"listings,omitempty"` // public managed-listings count
	Reviews      int      `json:"reviews,omitempty"`  // total reviews across the host's listings
	Verified     bool     `json:"verified,omitempty"` // identity verified
	Image        string   `json:"image,omitempty" table:",truncate"`
	URL          string   `json:"url"`
	ListingsRef  string   `json:"listings_ref,omitempty" table:"-" kit:"link,kind=airbnb/listings"` // edge to the host's listings (= id)
}

// Experience is an experience search result, emitted by experiences.
type Experience struct {
	ID       string  `json:"id" kit:"id"`
	Title    string  `json:"title,omitempty" table:",truncate"`
	Host     string  `json:"host,omitempty"`
	Price    float64 `json:"price,omitempty"` // per-guest price
	Currency string  `json:"currency,omitempty"`
	Rating   float64 `json:"rating,omitempty"`
	Reviews  int     `json:"reviews,omitempty"`
	Location string  `json:"location,omitempty"`
	Lat      float64 `json:"lat,omitempty"`
	Lng      float64 `json:"lng,omitempty"`
	Image    string  `json:"image,omitempty" table:",truncate"`
	URL      string  `json:"url"`
}

// Place is one location autocomplete suggestion, emitted by suggest. PlaceID is
// the identifier search resolves a free-text place into. SearchRef carries the
// place name as the edge into a stay search and ExperiencesRef as the edge into
// an experience search, so a crawl can start from a typed prefix and fan out into
// both the stays and the experiences a place returns.
type Place struct {
	Query          string  `json:"query"`              // the prefix that was queried
	Name           string  `json:"name" kit:"id"`      // the suggested place name
	PlaceID        string  `json:"place_id,omitempty"` // the Google place id the search uses
	Lat            float64 `json:"lat,omitempty"`
	Lng            float64 `json:"lng,omitempty"`
	SearchRef      string  `json:"search_ref,omitempty" table:"-" kit:"link,kind=airbnb/search"`           // edge into a stay search (= name)
	ExperiencesRef string  `json:"experiences_ref,omitempty" table:"-" kit:"link,kind=airbnb/experiences"` // edge into an experience search (= name)
}

// Ref is the result of `airbnb ref id`: the canonical (kind, id) a reference
// resolves to, plus the live URL, all without touching the network.
type Ref struct {
	Input string `json:"input"`
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	URL   string `json:"url"`
}
