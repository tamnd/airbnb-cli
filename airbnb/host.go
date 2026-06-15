package airbnb

import "context"

// host.go reads a host/user public profile via GetUserProfile and the host's
// public listings via BeehiveUserListings. Both are walled from datacenter IPs
// by Airbnb's edge bot manager, so they are best-effort: a wall returns
// ErrBlocked (exit 4). The listings query caps at about ten per call, which the
// docs name, so a host's public Listings count can read higher than the number
// of listing rows returned.

type userProfileResp struct {
	Presentation struct {
		UserProfileContainer struct {
			UserProfile apiUser `json:"userProfile"`
		} `json:"userProfileContainer"`
	} `json:"presentation"`
}

type apiUser struct {
	ID               string   `json:"id"`
	SmartName        string   `json:"smartName"`
	IsSuperhost      bool     `json:"isSuperhost"`
	CreatedAt        string   `json:"createdAt"`
	About            string   `json:"about"`
	Languages        []string `json:"languages"`
	ListingsCount    int      `json:"listingsCount"`
	ReviewsCount     int      `json:"reviewsCount"`
	ResponseRate     string   `json:"responseRate"`
	ResponseTime     string   `json:"responseTime"`
	IdentityVerified bool     `json:"identityVerified"`
	PictureURL       string   `json:"pictureUrl"`
}

// GetHost returns a host/user profile by id (or /users/show/ URL).
func (c *Client) GetHost(ctx context.Context, ref string) (*Host, error) {
	id := hostID(ref)
	if id == "" {
		return nil, ErrNotFound
	}
	vars := map[string]any{
		"userId":                          id,
		"isPassportStamp":                 false,
		"fetchCombinedSportsAndInterests": false,
	}
	var resp userProfileResp
	if err := c.gql(ctx, "GetUserProfile", vars, &resp); err != nil {
		return nil, err
	}
	u := resp.Presentation.UserProfileContainer.UserProfile
	uid := u.ID
	if uid == "" {
		uid = id
	}
	return &Host{
		ID:           uid,
		Name:         squish(u.SmartName),
		Superhost:    u.IsSuperhost,
		Since:        u.CreatedAt,
		About:        squish(u.About),
		ResponseRate: u.ResponseRate,
		ResponseTime: squish(u.ResponseTime),
		Languages:    u.Languages,
		Listings:     u.ListingsCount,
		Reviews:      u.ReviewsCount,
		Verified:     u.IdentityVerified,
		Image:        u.PictureURL,
		URL:          BaseURL + "/users/show/" + uid,
		ListingsRef:  uid,
	}, nil
}

type beehiveResp struct {
	Beehive struct {
		GetListOfListings struct {
			Listings []apiBeehiveListing `json:"listings"`
		} `json:"getListOfListings"`
	} `json:"beehive"`
}

type apiBeehiveListing struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	RoomType    string  `json:"roomType"`
	AvgRating   float64 `json:"avgRating"`
	ReviewCount int     `json:"reviewCount"`
	PictureURL  string  `json:"pictureUrl"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
}

// HostListings returns a host's public listings, up to limit (capped by the
// operation at about ten).
func (c *Client) HostListings(ctx context.Context, ref string, limit int) ([]*Listing, error) {
	id := hostID(ref)
	if id == "" {
		return nil, ErrNotFound
	}
	vars := map[string]any{
		"userId": id,
	}
	var resp beehiveResp
	if err := c.gql(ctx, "BeehiveUserListings", vars, &resp); err != nil {
		return nil, err
	}
	var out []*Listing
	seen := map[string]bool{}
	for _, bl := range resp.Beehive.GetListOfListings.Listings {
		rid := normalizeRoomID(bl.ID)
		if rid == "" || seen[rid] {
			continue
		}
		seen[rid] = true
		out = append(out, &Listing{
			ID:       rid,
			Name:     squish(bl.Name),
			RoomType: bl.RoomType,
			Rating:   bl.AvgRating,
			Reviews:  bl.ReviewCount,
			Lat:      bl.Lat,
			Lng:      bl.Lng,
			Image:    bl.PictureURL,
			URL:      BaseURL + "/rooms/" + rid,
			Room:     rid,
			Host:     id,
		})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}
