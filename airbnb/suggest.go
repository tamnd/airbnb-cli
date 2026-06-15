package airbnb

import (
	"context"
	"net/url"
)

// suggest.go reads the api/v2 autocompletes-personalized endpoint, the lightest
// surface and the one most likely to answer, though it too is edge-gated. It
// returns location suggestions, each with the Google place id the search uses,
// so a host can chain suggest into search.

type autocompleteResp struct {
	AutocompleteTerms []struct {
		DisplayName string `json:"display_name"`
		Location    struct {
			LocationName  string  `json:"location_name"`
			GooglePlaceID string  `json:"google_place_id"`
			Lat           float64 `json:"lat"`
			Lng           float64 `json:"lng"`
		} `json:"location"`
	} `json:"autocomplete_terms"`
}

// Suggest returns location autocomplete terms for a typed prefix.
func (c *Client) Suggest(ctx context.Context, prefix string, limit int) ([]*Place, error) {
	q := url.Values{}
	q.Set("user_input", prefix)
	q.Set("num_results", "10")
	q.Set("options", "should_show_pdp_url_only")
	var resp autocompleteResp
	if err := c.getJSON(ctx, "autocompletes-personalized", q, &resp); err != nil {
		return nil, err
	}
	var out []*Place
	seen := map[string]bool{}
	for _, t := range resp.AutocompleteTerms {
		name := t.Location.LocationName
		if name == "" {
			name = t.DisplayName
		}
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, &Place{
			Query:   prefix,
			Name:    squish(name),
			PlaceID: t.Location.GooglePlaceID,
			Lat:     t.Location.Lat,
			Lng:     t.Location.Lng,
		})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}
