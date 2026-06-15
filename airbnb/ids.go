package airbnb

import (
	"encoding/base64"
	"strings"
)

// ids.go resolves a reference to a (kind, id) pair, builds canonical URLs, and
// encodes and decodes the GraphQL global ids, all offline. It backs `airbnb ref
// id` and `airbnb ref url`, and the Resolver the ant host calls to turn an
// airbnb:// URI into the right command.

// reserved path segments are not ids.
var reserved = map[string]bool{
	"rooms": true, "users": true, "experiences": true, "s": true, "h": true,
	"api": true, "help": true, "account": true, "plus": true, "show": true,
}

// Classify reads a reference (a URL, a path, a bare id, or a pasted GraphQL
// global id) and reports what it points at. Kind is one of room, host,
// experience, or unknown.
func Classify(ref string) Ref {
	in := strings.TrimSpace(ref)
	r := Ref{Input: in, Kind: "unknown"}
	if in == "" {
		return r
	}

	// A pasted GraphQL global id (base64 of "<Type>:<digits>").
	if kind, id, ok := decodeGlobalID(in); ok {
		r.Kind, r.ID = kind, id
		r.URL = URLFor(kind, id)
		return r
	}

	// A bare numeric id is the most common reference and points at a room.
	if isDigits(in) {
		r.Kind, r.ID = "room", in
		r.URL = URLFor(r.Kind, r.ID)
		return r
	}

	segs := splitSegs(refPath(in))
	if len(segs) == 0 {
		return r
	}

	switch segs[0] {
	case "rooms":
		// /rooms/<id> or /rooms/plus/<id>: the id is the last all-digit segment.
		if id := lastDigits(segs[1:]); id != "" {
			r.Kind, r.ID = "room", id
		}
	case "users":
		// /users/show/<id>: the user id is the last all-digit segment.
		if id := lastDigits(segs[1:]); id != "" {
			r.Kind, r.ID = "host", id
		}
	case "experiences":
		if id := lastDigits(segs[1:]); id != "" {
			r.Kind, r.ID = "experience", id
		}
	}
	if r.Kind != "unknown" {
		r.URL = URLFor(r.Kind, r.ID)
	}
	return r
}

// URLFor builds the canonical Airbnb URL for a (kind, id) pair.
func URLFor(kind, id string) string {
	switch kind {
	case "room":
		return BaseURL + "/rooms/" + id
	case "host":
		return BaseURL + "/users/show/" + id
	case "experience":
		return BaseURL + "/experiences/" + id
	default:
		return ""
	}
}

// encodeGlobalID builds the GraphQL global id for an entity: base64 of
// "<typeName>:<id>". It is what the api/v3 variables carry for a room or a user.
func encodeGlobalID(typeName, id string) string {
	return base64.StdEncoding.EncodeToString([]byte(typeName + ":" + id))
}

// decodeGlobalID base64-decodes a global id and recovers its (kind, numeric id).
// It accepts the StayListing/DemandStayListing/User type names the web client
// uses and maps each to a CLI kind. ok is false when the input is not a global
// id this CLI recognizes.
func decodeGlobalID(s string) (kind, id string, ok bool) {
	// A global id is base64 and decodes to "<Type>:<digits>". Reject plain inputs
	// fast: they are not valid standard base64 of that shape.
	dec, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", "", false
	}
	typeName, num, found := strings.Cut(string(dec), ":")
	if !found || !isDigits(num) {
		return "", "", false
	}
	switch typeName {
	case "StayListing", "DemandStayListing", "Listing":
		return "room", num, true
	case "User", "Host":
		return "host", num, true
	case "Experience", "ExperienceHostPage":
		return "experience", num, true
	default:
		return "", "", false
	}
}

// roomID reduces a reference to its bare room id.
func roomID(ref string) string {
	if r := Classify(ref); r.Kind == "room" {
		return r.ID
	}
	return strings.Trim(ref, "/")
}

// hostID reduces a reference to its bare user id.
func hostID(ref string) string {
	if r := Classify(ref); r.Kind == "host" {
		return r.ID
	}
	return strings.Trim(ref, "/")
}

// refPath reduces a reference to a site path: a full URL loses scheme and host,
// a bare path is returned trimmed.
func refPath(ref string) string {
	if i := strings.Index(ref, "://"); i >= 0 {
		rest := ref[i+3:]
		if s := strings.IndexByte(rest, '/'); s >= 0 {
			rest = rest[s:]
		} else {
			return "/"
		}
		ref = rest
	}
	if q := strings.IndexByte(ref, '?'); q >= 0 {
		ref = ref[:q]
	}
	if !strings.HasPrefix(ref, "/") {
		ref = "/" + ref
	}
	return ref
}

func splitSegs(path string) []string {
	var out []string
	for _, s := range strings.Split(path, "/") {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// lastDigits returns the last all-digit segment that is not a reserved word.
func lastDigits(segs []string) string {
	id := ""
	for _, s := range segs {
		if reserved[s] {
			continue
		}
		if isDigits(s) {
			id = s
		}
	}
	return id
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
