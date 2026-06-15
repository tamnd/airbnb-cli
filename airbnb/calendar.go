package airbnb

import "context"

// calendar.go reads a listing's availability and nightly price via the
// PdpAvailabilityCalendar GraphQL operation. It is walled from datacenter IPs by
// Airbnb's edge bot manager, so this is best-effort: a wall returns ErrBlocked
// (exit 4). Each day links back to its listing, and its id joins the room id to
// the date so the store keeps one row per listing-day.

const calendarMonths = 12

type calendarResp struct {
	Merlin struct {
		PdpAvailabilityCalendar struct {
			CalendarMonths []struct {
				Days []apiDay `json:"days"`
			} `json:"calendarMonths"`
		} `json:"pdpAvailabilityCalendar"`
	} `json:"merlin"`
}

type apiDay struct {
	CalendarDate string `json:"calendarDate"`
	Available    bool   `json:"available"`
	Bookable     bool   `json:"bookable"`
	MinNights    int    `json:"minNights"`
	MaxNights    int    `json:"maxNights"`
	Price        *struct {
		LocalPriceFormatted string `json:"localPriceFormatted"`
		LocalCurrency       string `json:"localCurrency"`
	} `json:"price"`
}

// Calendar returns a listing's availability days, up to limit (0 = all fetched).
func (c *Client) Calendar(ctx context.Context, ref string, limit int) ([]*Day, error) {
	id := roomID(ref)
	if id == "" {
		return nil, ErrNotFound
	}
	vars := map[string]any{
		"request": map[string]any{
			"count":     calendarMonths,
			"listingId": id,
			"month":     1,
			"year":      2000, // server clamps to the current month forward
		},
	}
	var resp calendarResp
	if err := c.gql(ctx, "PdpAvailabilityCalendar", vars, &resp); err != nil {
		return nil, err
	}
	var out []*Day
	for _, m := range resp.Merlin.PdpAvailabilityCalendar.CalendarMonths {
		for _, d := range m.Days {
			if d.CalendarDate == "" {
				continue
			}
			out = append(out, d.toDay(id, c.Currency))
			if limit > 0 && len(out) >= limit {
				return out, nil
			}
		}
	}
	return out, nil
}

func (d apiDay) toDay(roomID, fallbackCurrency string) *Day {
	day := &Day{
		ID:        roomID + ":" + d.CalendarDate,
		Date:      d.CalendarDate,
		Available: d.Available || d.Bookable,
		MinNights: d.MinNights,
		MaxNights: d.MaxNights,
		Room:      roomID,
	}
	if d.Price != nil {
		day.Price = priceFromDisplay(d.Price.LocalPriceFormatted)
		day.Currency = d.Price.LocalCurrency
	}
	if day.Currency == "" && day.Price > 0 {
		day.Currency = fallbackCurrency
	}
	return day
}
