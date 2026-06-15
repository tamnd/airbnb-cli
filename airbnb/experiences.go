package airbnb

import "context"

// experiences.go reads experience search for a place via the ExperiencesSearch
// GraphQL operation. It is walled from datacenter IPs by Airbnb's edge bot
// manager, so this is best-effort: a wall returns ErrBlocked (exit 4). There is
// no single-experience resolver, since no stable anonymous detail operation was
// confirmed; the experience kind is still classifiable for ref id / ref url.

type experiencesResp struct {
	Presentation struct {
		ExperiencesSearch struct {
			Results struct {
				SearchResults  []apiExperience `json:"searchResults"`
				PaginationInfo struct {
					NextPageCursor string `json:"nextPageCursor"`
				} `json:"paginationInfo"`
			} `json:"results"`
		} `json:"experiencesSearch"`
	} `json:"presentation"`
}

type apiExperience struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	HostName     string  `json:"hostName"`
	DisplayPrice string  `json:"displayPrice"`
	StarRating   float64 `json:"starRating"`
	ReviewCount  int     `json:"reviewCount"`
	LocationName string  `json:"locationName"`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	PictureURL   string  `json:"pictureUrl"`
}

// Experiences returns experience cards for a place, paging up to limit.
func (c *Client) Experiences(ctx context.Context, place string, limit int) ([]*Experience, error) {
	var out []*Experience
	cursor := ""
	seen := map[string]bool{}
	for {
		vars := map[string]any{
			"experiencesSearchRequest": map[string]any{
				"query":  place,
				"cursor": cursor,
			},
		}
		var resp experiencesResp
		if err := c.gql(ctx, "ExperiencesSearch", vars, &resp); err != nil {
			if len(out) > 0 {
				return out, nil
			}
			return nil, err
		}
		res := resp.Presentation.ExperiencesSearch.Results
		for _, e := range res.SearchResults {
			id := normalizeExperienceID(e.ID)
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			out = append(out, &Experience{
				ID:       id,
				Title:    squish(e.Title),
				Host:     squish(e.HostName),
				Price:    priceFromDisplay(e.DisplayPrice),
				Currency: c.Currency,
				Rating:   e.StarRating,
				Reviews:  e.ReviewCount,
				Location: squish(e.LocationName),
				Lat:      e.Lat,
				Lng:      e.Lng,
				Image:    e.PictureURL,
				URL:      BaseURL + "/experiences/" + id,
			})
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

func normalizeExperienceID(s string) string {
	if s == "" {
		return ""
	}
	if kind, id, ok := decodeGlobalID(s); ok && kind == "experience" {
		return id
	}
	if isDigits(s) {
		return s
	}
	return ""
}
