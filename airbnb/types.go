package airbnb

// This file holds the exported records the commands emit. Their json tags name
// the fields a reader sees, kit:"id" marks the key the record store upserts on,
// kit:"body" marks the long-text field `airbnb cat` and the Markdown export
// print, and table:",truncate" keeps wide free text from blowing up a terminal
// table. Each record carries only fields a logged-out reader can actually fill:
// no trips, no messages, no wishlists, no host management, no payout data,
// because none of that is reachable without a signed-in account. There is no
// Rank column either; emit order is the rank. Several records are emitted by more
// than one surface, and omitempty carries the gaps. The kit:"link" edges connect
// the records into a graph a host walks for breadth-first crawls (place -> search
// -> listings -> host -> the host's other listings; listing -> reviews and
// calendar). The per-surface files hold the parsing these map from.

// Listing is a summary card in a grid, emitted by search and host listings. Its
// id is the room id, so Room carries the same value as the graph edge to the full
// listing: a host walks a search result straight through to airbnb://room/<id>.
type Listing struct {
	ID       string   `json:"id" kit:"id"` // room id
	Name     string   `json:"name,omitempty" table:",truncate"`
	Title    string   `json:"title,omitempty" table:",truncate"` // the card's title line
	RoomType string   `json:"room_type,omitempty"`               // Entire home, Private room, ...
	Price    float64  `json:"price,omitempty"`                   // nightly, for the searched dates
	Currency string   `json:"currency,omitempty"`
	Total    float64  `json:"total,omitempty"` // total for the stay, when dates are given
	Rating   float64  `json:"rating,omitempty"`
	Reviews  int      `json:"reviews,omitempty"`
	Lat      float64  `json:"lat,omitempty"`
	Lng      float64  `json:"lng,omitempty"`
	Badges   []string `json:"badges,omitempty" table:"-"` // Guest favorite, Superhost, ...
	Image    string   `json:"image,omitempty" table:",truncate"`
	URL      string   `json:"url"`
	Room     string   `json:"room,omitempty" table:"-" kit:"link,kind=airbnb/room"` // edge to the full listing (= id)
	Host     string   `json:"host,omitempty" table:"-" kit:"link,kind=airbnb/host"` // edge to the host, when the card carries it
}

// Room is the full detail for one listing, emitted by room. It is parsed from the
// PDP JSON island, with its date-specific price filled from StaysPdpSections when
// check-in and check-out are given.
type Room struct {
	ID             string   `json:"id" kit:"id"` // room id
	Name           string   `json:"name,omitempty" table:",truncate"`
	Description    string   `json:"description,omitempty" table:",truncate" kit:"body"`
	PropertyType   string   `json:"property_type,omitempty"` // House, Apartment, ...
	RoomType       string   `json:"room_type,omitempty"`     // Entire home, Private room, ...
	Capacity       int      `json:"capacity,omitempty"`      // person capacity
	Bedrooms       int      `json:"bedrooms,omitempty"`
	Beds           int      `json:"beds,omitempty"`
	Bathrooms      string   `json:"bathrooms,omitempty"` // "1.5 baths"; text, since half-baths are common
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
	Reviews        int      `json:"reviews,omitempty"` // visible review count
	Lat            float64  `json:"lat,omitempty"`
	Lng            float64  `json:"lng,omitempty"`
	Superhost      bool     `json:"superhost,omitempty"`
	HostID         string   `json:"host_id,omitempty" table:"-" kit:"link,kind=airbnb/host"` // edge to the host
	HostName       string   `json:"host_name,omitempty"`
	Images         []string `json:"images,omitempty" table:"-"`        // the photo gallery
	Image          string   `json:"image,omitempty" table:",truncate"` // the first photo
	HouseRules     []string `json:"house_rules,omitempty" table:"-"`
	URL            string   `json:"url"`
}

// Review is one review of a listing, emitted by reviews.
type Review struct {
	ID       string `json:"id" kit:"id"`
	Author   string `json:"author,omitempty"`
	Location string `json:"location,omitempty"` // the reviewer's stated home, when shown
	Date     string `json:"date,omitempty"`     // the localized review date
	Rating   int    `json:"rating,omitempty"`   // the reviewer's star rating, when present
	Text     string `json:"text,omitempty" table:",truncate" kit:"body"`
	Response string `json:"response,omitempty" table:",truncate"` // the host's public response, when present
	Language string `json:"language,omitempty"`
	Room     string `json:"room,omitempty" table:"-" kit:"link,kind=airbnb/room"` // edge back to the listing
}

// Day is one day on a listing's availability calendar, emitted by calendar. The
// id is the room id joined to the date, so the store keeps one row per
// listing-day and a re-fetch upserts cleanly.
type Day struct {
	ID        string  `json:"id" kit:"id"` // "<roomid>:<date>"
	Date      string  `json:"date"`        // YYYY-MM-DD
	Available bool    `json:"available"`
	MinNights int     `json:"min_nights,omitempty"`
	MaxNights int     `json:"max_nights,omitempty"`
	Price     float64 `json:"price,omitempty"` // nightly price for this date, when published
	Currency  string  `json:"currency,omitempty"`
	Room      string  `json:"room,omitempty" table:"-" kit:"link,kind=airbnb/room"` // edge back to the listing
}

// Host is a host/user public profile, emitted by host show.
type Host struct {
	ID           string   `json:"id" kit:"id"` // user id
	Name         string   `json:"name,omitempty"`
	Superhost    bool     `json:"superhost,omitempty"`
	Since        string   `json:"since,omitempty"` // the "joined in <year>" text
	About        string   `json:"about,omitempty" table:",truncate" kit:"body"`
	ResponseRate string   `json:"response_rate,omitempty"`
	Languages    []string `json:"languages,omitempty" table:"-"`
	Listings     int      `json:"listings,omitempty"` // public managed-listings count
	Reviews      int      `json:"reviews,omitempty"`  // total reviews across the host's listings
	Verified     bool     `json:"verified,omitempty"` // identity verified
	Image        string   `json:"image,omitempty" table:",truncate"`
	URL          string   `json:"url"`
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
// the identifier search resolves a free-text place into, so a host can chain
// suggest into search.
type Place struct {
	Query   string  `json:"query"`              // the prefix that was queried
	Name    string  `json:"name" kit:"id"`      // the suggested place name
	PlaceID string  `json:"place_id,omitempty"` // the Google place id the search uses
	Lat     float64 `json:"lat,omitempty"`
	Lng     float64 `json:"lng,omitempty"`
}

// Ref is the result of `airbnb ref id`: the canonical (kind, id) a reference
// resolves to, plus the live URL, all without touching the network.
type Ref struct {
	Input string `json:"input"`
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	URL   string `json:"url"`
}
