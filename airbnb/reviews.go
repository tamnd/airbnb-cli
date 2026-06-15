package airbnb

import "context"

// reviews.go reads a listing's reviews via the StaysPdpReviewsQuery GraphQL
// operation, paging its offset. It is walled from datacenter IPs by Airbnb's
// edge bot manager, so this is best-effort: a wall returns ErrBlocked (exit 4).
// Each review links back to its listing, so a crawl can walk reviews to the room.

const reviewsPage = 50

type reviewsResp struct {
	Presentation struct {
		StayProductDetailPage struct {
			Reviews struct {
				Reviews []apiReview `json:"reviews"`
			} `json:"reviews"`
		} `json:"stayProductDetailPage"`
	} `json:"presentation"`
}

type apiReview struct {
	ID            string `json:"id"`
	Comments      string `json:"comments"`
	Language      string `json:"language"`
	LocalizedDate string `json:"localizedDate"`
	Rating        int    `json:"rating"`
	Reviewer      struct {
		FirstName string `json:"firstName"`
		Location  string `json:"location"`
	} `json:"reviewer"`
	Response string `json:"response"`
}

// Reviews returns a listing's reviews, paging up to limit.
func (c *Client) Reviews(ctx context.Context, ref string, limit int) ([]*Review, error) {
	id := roomID(ref)
	if id == "" {
		return nil, ErrNotFound
	}
	var out []*Review
	seen := map[string]bool{}
	for offset := 0; ; offset += reviewsPage {
		vars := map[string]any{
			"id": encodeGlobalID("StayListing", id),
			"pdpReviewsRequest": map[string]any{
				"fieldSelector":     "for_p3_translation_only",
				"limit":             reviewsPage,
				"offset":            offset,
				"sortingPreference": "MOST_RECENT",
			},
		}
		var resp reviewsResp
		if err := c.gql(ctx, "StaysPdpReviewsQuery", vars, &resp); err != nil {
			if len(out) > 0 {
				return out, nil
			}
			return nil, err
		}
		page := resp.Presentation.StayProductDetailPage.Reviews.Reviews
		if len(page) == 0 {
			break
		}
		for _, rv := range page {
			if rv.ID == "" || seen[rv.ID] {
				continue
			}
			seen[rv.ID] = true
			out = append(out, rv.toReview(id))
			if limit > 0 && len(out) >= limit {
				return out, nil
			}
		}
		if len(page) < reviewsPage {
			break
		}
	}
	return out, nil
}

func (rv apiReview) toReview(roomID string) *Review {
	return &Review{
		ID:       rv.ID,
		Author:   squish(rv.Reviewer.FirstName),
		Location: squish(rv.Reviewer.Location),
		Date:     rv.LocalizedDate,
		Rating:   rv.Rating,
		Text:     squish(rv.Comments),
		Response: squish(rv.Response),
		Language: rv.Language,
		Room:     roomID,
	}
}
