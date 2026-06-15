package airbnb

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// island.go reads the JSON island Airbnb embeds in a listing page. The page
// carries a single <script id="data-deferred-state-0" type="application/json">
// block holding the whole client data tree. The useful object is reached by
// walking the Niobe client data to the stay PDP sections, an array of typed
// section objects, plus a metadata blob carrying the flat numeric facts
// (coordinates, capacity, the six category ratings, the review count, the
// superhost flag). The shapes here are deliberately loose, decoding only the
// fields Room needs, because the island carries far more than this tool reports.

// extractIsland returns the JSON bytes of the data-deferred-state-0 island, or
// false when the page has none (a wall stub, or an unexpected layout).
func extractIsland(body []byte) ([]byte, bool) {
	marker := []byte(`id="data-deferred-state-0"`)
	i := bytes.Index(body, marker)
	if i < 0 {
		return nil, false
	}
	start := bytes.IndexByte(body[i:], '>')
	if start < 0 {
		return nil, false
	}
	start += i + 1
	end := bytes.Index(body[start:], []byte("</script>"))
	if end < 0 {
		return nil, false
	}
	return bytes.TrimSpace(body[start : start+end]), true
}

// island is the top of the embedded tree. The Niobe client data is an array of
// [operationName, payload] pairs; Airbnb has shipped it under both
// niobeMinimalClientData and niobeClientData, so both are read.
type island struct {
	Minimal [][]json.RawMessage `json:"niobeMinimalClientData"`
	Full    [][]json.RawMessage `json:"niobeClientData"`
}

// pdpPayload is the PDP entry's payload: the stay product detail page sections.
type pdpPayload struct {
	Data struct {
		Presentation struct {
			StayProductDetailPage struct {
				Sections struct {
					Sections []pdpSection    `json:"sections"`
					Metadata json.RawMessage `json:"metadata"`
				} `json:"sections"`
			} `json:"stayProductDetailPage"`
		} `json:"presentation"`
	} `json:"data"`
}

type pdpSection struct {
	SectionID string      `json:"sectionId"`
	Section   sectionBody `json:"section"`
}

// sectionBody carries the union of fields the section types this tool reads. A
// given section fills only the few that match its __typename.
type sectionBody struct {
	Typename        string `json:"__typename"`
	Title           string `json:"title"`
	Subtitle        string `json:"subtitle"`
	HTMLDescription *struct {
		HTMLText string `json:"htmlText"`
	} `json:"htmlDescription"`
	DetailItems []struct {
		Title string `json:"title"`
	} `json:"detailItems"`
	SeeAllAmenitiesGroups []struct {
		Amenities []struct {
			Title     string `json:"title"`
			Available bool   `json:"available"`
		} `json:"amenities"`
	} `json:"seeAllAmenitiesGroups"`
	MediaItems []struct {
		BaseURL string `json:"baseUrl"`
	} `json:"mediaItems"`
	HouseRules []struct {
		Title string `json:"title"`
	} `json:"houseRules"`
	// PdpHighlightsSection: icon highlights such as "Self check-in".
	Highlights []struct {
		Title    string `json:"title"`
		Subtitle string `json:"subtitle"`
	} `json:"highlights"`
	// SleepingArrangementSection: one entry per room, title "Bedroom 1",
	// subtitle "1 queen bed".
	ArrangementDetails []struct {
		Title    string `json:"title"`
		Subtitle string `json:"subtitle"`
	} `json:"arrangementDetails"`
	IsSuperhost bool   `json:"isSuperhost"`
	HostName    string `json:"hostName"`
	HostID      string `json:"hostId"`
}

// eventData is the flat numeric blob the island logs alongside the sections.
type eventData struct {
	LoggingContext struct {
		EventDataLogging struct {
			ListingLat               float64 `json:"listingLat"`
			ListingLng               float64 `json:"listingLng"`
			PersonCapacity           int     `json:"personCapacity"`
			AccuracyRating           float64 `json:"accuracyRating"`
			CheckinRating            float64 `json:"checkinRating"`
			CleanlinessRating        float64 `json:"cleanlinessRating"`
			CommunicationRating      float64 `json:"communicationRating"`
			LocationRating           float64 `json:"locationRating"`
			ValueRating              float64 `json:"valueRating"`
			GuestSatisfactionOverall float64 `json:"guestSatisfactionOverall"`
			VisibleReviewCount       int     `json:"visibleReviewCount"`
			ReviewCount              int     `json:"reviewCount"`
			RoomType                 string  `json:"roomType"`
			PropertyType             string  `json:"propertyType"`
			IsSuperhost              bool    `json:"isSuperhost"`
			HostID                   string  `json:"hostId"`
		} `json:"eventDataLogging"`
	} `json:"loggingContext"`
}

// parsePDPIsland maps a listing page island to a Room, or nil when the island
// carries no PDP payload. id seeds the record when the payload omits it.
func parsePDPIsland(nd []byte, id string) *Room {
	var root island
	if json.Unmarshal(nd, &root) != nil {
		return nil
	}
	entries := root.Minimal
	if len(entries) == 0 {
		entries = root.Full
	}
	for _, pair := range entries {
		if len(pair) < 2 {
			continue
		}
		var p pdpPayload
		if json.Unmarshal(pair[1], &p) != nil {
			continue
		}
		secs := p.Data.Presentation.StayProductDetailPage.Sections
		if len(secs.Sections) == 0 && len(secs.Metadata) == 0 {
			continue
		}
		return roomFromSections(secs.Sections, secs.Metadata, id)
	}
	return nil
}

// roomFromSections builds a Room from the typed sections and the metadata blob.
func roomFromSections(sections []pdpSection, metadata json.RawMessage, id string) *Room {
	r := &Room{ID: id, URL: BaseURL + "/rooms/" + id}

	for _, s := range sections {
		b := s.Section
		switch b.Typename {
		case "PdpTitleSection":
			r.Name = squish(b.Title)
		case "PdpDescriptionSection":
			if b.HTMLDescription != nil {
				r.Description = stripHTML(b.HTMLDescription.HTMLText)
			}
		case "PdpOverviewV2Section", "PdpOverviewSection", "SharingConfigOverviewSection":
			applyOverview(r, b.DetailItems)
		case "PdpHighlightsSection":
			for _, h := range b.Highlights {
				if h.Title != "" {
					r.Highlights = append(r.Highlights, squish(h.Title))
				}
			}
		case "SleepingArrangementSection":
			for _, a := range b.ArrangementDetails {
				if a.Title == "" {
					continue
				}
				line := squish(a.Title)
				if a.Subtitle != "" {
					line += ": " + squish(a.Subtitle)
				}
				r.Sleeping = append(r.Sleeping, line)
			}
		case "LocationSection", "PdpLocationSection":
			if r.Location == "" {
				r.Location = squish(b.Subtitle)
			}
		case "AmenitiesSection", "PhotoTourModalAmenitiesSection":
			for _, g := range b.SeeAllAmenitiesGroups {
				for _, a := range g.Amenities {
					if a.Title != "" {
						r.Amenities = append(r.Amenities, a.Title)
					}
				}
			}
		case "PhotoTourModalSection", "HeroSection":
			for _, m := range b.MediaItems {
				if m.BaseURL != "" {
					r.Images = append(r.Images, m.BaseURL)
				}
			}
		case "HostProfileSection", "MeetYourHostSection":
			if b.HostName != "" {
				r.HostName = squish(b.HostName)
			}
			if b.HostID != "" {
				r.HostID = b.HostID
			}
			if b.IsSuperhost {
				r.Superhost = true
			}
		case "PoliciesSection", "PdpHouseRulesModalSection":
			for _, hr := range b.HouseRules {
				if hr.Title != "" {
					r.HouseRules = append(r.HouseRules, squish(hr.Title))
				}
			}
		}
	}
	if len(r.Images) > 0 {
		r.Image = r.Images[0]
	}
	applyCheckTimes(r)

	// The collection edges carry the room id so a crawl reaching a room expands
	// to its reviews and its calendar.
	r.ReviewsRef = id
	r.CalendarRef = id

	applyEventData(r, metadata)
	return r
}

// applyCheckTimes lifts the check-in and checkout windows out of the house rules,
// where Airbnb states them as lines like "Check-in after 3:00 PM" and "Checkout
// before 11:00 AM". The wording is stable enough to read without a dedicated
// field, and a listing that omits the lines just leaves the fields empty.
func applyCheckTimes(r *Room) {
	for _, hr := range r.HouseRules {
		low := strings.ToLower(hr)
		switch {
		case r.CheckIn == "" && strings.HasPrefix(low, "check-in"):
			r.CheckIn = hr
		case r.CheckOut == "" && (strings.HasPrefix(low, "checkout") || strings.HasPrefix(low, "check-out")):
			r.CheckOut = hr
		}
	}
}

// applyOverview reads the bedrooms/beds/baths/guests line, whose items read like
// "4 guests", "2 bedrooms", "2 beds", "1.5 baths".
func applyOverview(r *Room, items []struct {
	Title string `json:"title"`
}) {
	for _, it := range items {
		t := strings.ToLower(it.Title)
		switch {
		case strings.Contains(t, "guest"):
			r.Capacity = firstInt(t)
		case strings.Contains(t, "bedroom"):
			r.Bedrooms = firstInt(t)
		case strings.Contains(t, "bed"):
			r.Beds = firstInt(t)
		case strings.Contains(t, "bath"):
			r.Bathrooms = squish(it.Title)
		}
	}
}

// applyEventData fills the flat numeric facts from the metadata blob.
func applyEventData(r *Room, metadata json.RawMessage) {
	if len(metadata) == 0 {
		return
	}
	var ev eventData
	if json.Unmarshal(metadata, &ev) != nil {
		return
	}
	e := ev.LoggingContext.EventDataLogging
	if r.Lat == 0 {
		r.Lat = e.ListingLat
	}
	if r.Lng == 0 {
		r.Lng = e.ListingLng
	}
	if r.Capacity == 0 {
		r.Capacity = e.PersonCapacity
	}
	r.Accuracy = e.AccuracyRating
	r.CheckinRating = e.CheckinRating
	r.Cleanliness = e.CleanlinessRating
	r.Communication = e.CommunicationRating
	r.LocationRating = e.LocationRating
	r.Value = e.ValueRating
	r.Rating = e.GuestSatisfactionOverall
	if e.VisibleReviewCount > 0 {
		r.Reviews = e.VisibleReviewCount
	} else {
		r.Reviews = e.ReviewCount
	}
	if r.RoomType == "" {
		r.RoomType = e.RoomType
	}
	if r.PropertyType == "" {
		r.PropertyType = e.PropertyType
	}
	if e.IsSuperhost {
		r.Superhost = true
	}
	if r.HostID == "" {
		r.HostID = e.HostID
	}
}

// firstInt returns the first whole number in s, or 0.
func firstInt(s string) int {
	var digits strings.Builder
	for _, c := range s {
		if c >= '0' && c <= '9' {
			digits.WriteRune(c)
		} else if digits.Len() > 0 {
			break
		}
	}
	n, _ := strconv.Atoi(digits.String())
	return n
}
