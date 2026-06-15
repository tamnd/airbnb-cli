package airbnb

import "context"

// room.go reads one listing by id. The /rooms/<id> page embeds its whole state
// as the data-deferred-state-0 island, which this client GETs and parses for the
// static detail. The page is walled from datacenter IPs by Airbnb's edge bot
// manager, so this is best-effort: when the wall is up the GET returns ErrBlocked
// (exit 4). When check-in and check-out are set, the date-specific nightly price
// is read from the StaysPdpSections GraphQL operation and folded in; without
// dates the price is left empty rather than invented.

// GetRoom returns one listing by id (or /rooms/ URL).
func (c *Client) GetRoom(ctx context.Context, ref string) (*Room, error) {
	id := roomID(ref)
	if id == "" {
		return nil, ErrNotFound
	}
	body, err := c.get(ctx, c.BaseURL+"/rooms/"+id)
	if err != nil {
		return nil, err
	}
	nd, ok := extractIsland(body)
	if !ok {
		// No island on a 200 body is the wall serving a stub.
		return nil, ErrBlocked
	}
	r := parsePDPIsland(nd, id)
	if r == nil {
		return nil, ErrBlocked
	}
	if c.Checkin != "" && c.Checkout != "" {
		if price, cur := c.roomPrice(ctx, id); price > 0 {
			r.Price = price
			if cur != "" {
				r.Currency = cur
			}
		}
	}
	if r.Currency == "" {
		r.Currency = c.Currency
	}
	return r, nil
}

// pdpSectionsResp is the StaysPdpSections shape this reads for the priced line.
type pdpSectionsResp struct {
	Presentation struct {
		StayProductDetailPage struct {
			Sections struct {
				Sections []struct {
					Section struct {
						Typename               string `json:"__typename"`
						StructuredDisplayPrice *struct {
							PrimaryLine *struct {
								Price string `json:"price"`
							} `json:"primaryLine"`
						} `json:"structuredDisplayPrice"`
					} `json:"section"`
				} `json:"sections"`
			} `json:"sections"`
		} `json:"stayProductDetailPage"`
	} `json:"presentation"`
}

// roomPrice fetches the date-specific nightly price for a listing. A wall or a
// missing price yields 0, so the caller leaves the field empty.
func (c *Client) roomPrice(ctx context.Context, id string) (float64, string) {
	vars := map[string]any{
		"id": encodeGlobalID("StayListing", id),
		"pdpSectionsRequest": map[string]any{
			"adults":   c.Adults,
			"children": c.Children,
			"checkIn":  c.Checkin,
			"checkOut": c.Checkout,
			"layouts":  []string{"SIDEBAR", "SINGLE_COLUMN"},
		},
	}
	var resp pdpSectionsResp
	if err := c.gql(ctx, "StaysPdpSections", vars, &resp); err != nil {
		return 0, ""
	}
	for _, s := range resp.Presentation.StayProductDetailPage.Sections.Sections {
		sp := s.Section.StructuredDisplayPrice
		if sp != nil && sp.PrimaryLine != nil && sp.PrimaryLine.Price != "" {
			return priceFromDisplay(sp.PrimaryLine.Price), c.Currency
		}
	}
	return 0, ""
}
