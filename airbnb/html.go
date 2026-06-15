package airbnb

import (
	"regexp"
	"strconv"
	"strings"
)

// html.go holds the small shared helpers the surfaces lean on. Airbnb's primary
// mode is JSON, not HTML cards (the listing page embeds its whole state as a JSON
// island, and every other surface is a JSON API), so most fields arrive already
// typed; these helpers clean the free text and read the few values that arrive
// as display strings (prices, the rating-and-count label).

var (
	priceRE  = regexp.MustCompile(`([0-9][0-9,]*\.?[0-9]*)`)
	tagRE    = regexp.MustCompile(`<[^>]+>`)
	ratingRE = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)`)
)

// priceFromDisplay turns a displayed price ("$120", "$1,360 total") into a
// number, or 0 when there is none. The first number wins and separators are
// dropped.
func priceFromDisplay(s string) float64 {
	m := priceRE.FindString(s)
	if m == "" {
		return 0
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(m, ",", ""), 64)
	if err != nil {
		return 0
	}
	return v
}

// stripHTML reduces an HTML fragment to its text, collapsing whitespace. Airbnb
// ships listing descriptions as small HTML blobs.
func stripHTML(s string) string {
	return squish(tagRE.ReplaceAllString(s, " "))
}

// squish collapses runs of whitespace into single spaces and trims the ends, so
// a value lifted from indented markup reads cleanly.
func squish(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// ratingFromLabel reads the "4.9 out of 5 average rating, 120 reviews" label
// Airbnb pairs with a card, returning the rating and the review count. Either is
// zero when the label does not carry it.
func ratingFromLabel(s string) (rating float64, reviews int) {
	if s == "" {
		return 0, 0
	}
	nums := ratingRE.FindAllString(s, -1)
	if len(nums) == 0 {
		return 0, 0
	}
	rating, _ = strconv.ParseFloat(nums[0], 64)
	// The review count is the last whole number in the label.
	for i := len(nums) - 1; i >= 0; i-- {
		if !strings.Contains(nums[i], ".") {
			reviews, _ = strconv.Atoi(nums[i])
			break
		}
	}
	return rating, reviews
}
