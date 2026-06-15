package airbnb

import "context"

// search.go reads stay search for a place via the StaysSearch GraphQL operation,
// the web client's own search data plane. It is walled from datacenter IPs by
// Airbnb's edge bot manager, so this is best-effort: a wall returns ErrBlocked
// (exit 4). Each card links straight through to its full listing (and to its
// host when the card names one), so a host can crawl a search result onward.

// staysSearchResp is the StaysSearch response shape.
type staysSearchResp struct {
	Presentation struct {
		StaysSearch struct {
			Results struct {
				SearchResults  []staySearchResult `json:"searchResults"`
				PaginationInfo struct {
					NextPageCursor string `json:"nextPageCursor"`
				} `json:"paginationInfo"`
			} `json:"results"`
		} `json:"staysSearch"`
	} `json:"presentation"`
}

type staySearchResult struct {
	Typename string `json:"__typename"`
	Listing  struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Title      string `json:"title"`
		Coordinate struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"coordinate"`
		AvgRatingLabel     string `json:"avgRatingA11yLabel"`
		RoomTypeCategory   string `json:"roomTypeCategory"`
		ContextualPictures []struct {
			Picture string `json:"picture"`
		} `json:"contextualPictures"`
		FormattedBadges []struct {
			Text string `json:"text"`
		} `json:"formattedBadges"`
		HostID string `json:"hostId"`
	} `json:"listing"`
	PricingQuote struct {
		StructuredStayDisplayPrice struct {
			PrimaryLine struct {
				Price string `json:"price"`
			} `json:"primaryLine"`
			SecondaryLine struct {
				Price string `json:"price"`
			} `json:"secondaryLine"`
		} `json:"structuredStayDisplayPrice"`
	} `json:"pricingQuote"`
}

// Search returns listings for a place, paging the cursor up to limit.
func (c *Client) Search(ctx context.Context, place string, limit int) ([]*Listing, error) {
	var out []*Listing
	cursor := ""
	seen := map[string]bool{}
	for {
		vars := map[string]any{
			"staysSearchRequest": map[string]any{
				"query":        place,
				"cursor":       cursor,
				"checkin":      c.Checkin,
				"checkout":     c.Checkout,
				"adults":       c.Adults,
				"children":     c.Children,
				"itemsPerGrid": defaultPageSize,
				"source":       "structured_search_input_header",
				"searchType":   "filter_change",
			},
		}
		var resp staysSearchResp
		if err := c.gql(ctx, "StaysSearch", vars, &resp); err != nil {
			if len(out) > 0 {
				return out, nil
			}
			return nil, err
		}
		res := resp.Presentation.StaysSearch.Results
		for _, sr := range res.SearchResults {
			l := sr.toListing()
			if l == nil || seen[l.ID] {
				continue
			}
			seen[l.ID] = true
			out = append(out, l)
			if limit > 0 && len(out) >= limit {
				return out, nil
			}
		}
		next := res.PaginationInfo.NextPageCursor
		if next == "" || next == cursor || len(res.SearchResults) == 0 {
			break
		}
		cursor = next
	}
	return out, nil
}

// toListing maps one search card, or nil when it carries no usable room id.
func (sr staySearchResult) toListing() *Listing {
	id := normalizeRoomID(sr.Listing.ID)
	if id == "" {
		return nil
	}
	l := &Listing{
		ID:       id,
		Name:     squish(sr.Listing.Name),
		Title:    squish(sr.Listing.Title),
		RoomType: roomTypeLabel(sr.Listing.RoomTypeCategory),
		Lat:      sr.Listing.Coordinate.Latitude,
		Lng:      sr.Listing.Coordinate.Longitude,
		URL:      BaseURL + "/rooms/" + id,
		Room:     id,
	}
	l.Rating, l.Reviews = ratingFromLabel(sr.Listing.AvgRatingLabel)
	if p := sr.PricingQuote.StructuredStayDisplayPrice.PrimaryLine.Price; p != "" {
		l.Price = priceFromDisplay(p)
	}
	if t := sr.PricingQuote.StructuredStayDisplayPrice.SecondaryLine.Price; t != "" {
		l.Total = priceFromDisplay(t)
	}
	if len(sr.Listing.ContextualPictures) > 0 {
		l.Image = sr.Listing.ContextualPictures[0].Picture
	}
	for _, b := range sr.Listing.FormattedBadges {
		if b.Text != "" {
			l.Badges = append(l.Badges, b.Text)
		}
	}
	if h := normalizeHostID(sr.Listing.HostID); h != "" {
		l.Host = h
	}
	return l
}

// normalizeRoomID turns a search id (a base64 global id or a bare number) into a
// bare room id, or "" when it is neither.
func normalizeRoomID(s string) string {
	if s == "" {
		return ""
	}
	if kind, id, ok := decodeGlobalID(s); ok && kind == "room" {
		return id
	}
	if isDigits(s) {
		return s
	}
	return ""
}

// normalizeHostID does the same for a host id carried on a card.
func normalizeHostID(s string) string {
	if s == "" {
		return ""
	}
	if kind, id, ok := decodeGlobalID(s); ok && kind == "host" {
		return id
	}
	if isDigits(s) {
		return s
	}
	return ""
}

// roomTypeLabel turns a roomTypeCategory token ("entire_home") into a readable
// label ("Entire home").
func roomTypeLabel(s string) string {
	switch s {
	case "entire_home":
		return "Entire home"
	case "private_room":
		return "Private room"
	case "shared_room":
		return "Shared room"
	case "hotel_room":
		return "Hotel room"
	case "":
		return ""
	default:
		return s
	}
}
