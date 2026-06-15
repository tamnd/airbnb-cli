package airbnb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// api.go is the client for Airbnb's internal data plane: the api/v3 GraphQL
// endpoint and the lighter api/v2 JSON endpoints the logged-out web client uses.
// The web app does not send raw queries; it sends persisted queries, addressed
// by operation name and a SHA-256 hash that is tied to the deployed frontend
// bundle. This client sends exactly those, with the public web API key the site
// itself ships. It authors no new GraphQL documents and registers no new
// persisted queries. A stale hash or a rejected key comes back as a GraphQL
// errors envelope, which maps to ErrBlocked (exit 4), never to a fabricated
// record.

// gqlEnvelope is the GraphQL response shape: a data tree and an optional errors
// list. The per-surface decoders unmarshal data into their own typed shapes.
type gqlEnvelope struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message    string `json:"message"`
		Extensions struct {
			Classification string `json:"classification"`
		} `json:"extensions"`
	} `json:"errors"`
}

// gql posts a persisted query and unmarshals data into out. vars is the
// operation's variables object. A transport wall propagates as ErrBlocked; a
// GraphQL errors envelope whose message names an auth or rate problem maps to
// ErrBlocked, and a not-found message to ErrNotFound.
func (c *Client) gql(ctx context.Context, op string, vars any, out any) error {
	hash := c.hashes[op]
	varsJSON, err := json.Marshal(vars)
	if err != nil {
		return err
	}
	// Cache by operation and a stable hash of the variables, so re-running the
	// same query stays off the network within the TTL.
	ck := "gql:" + op + ":" + shortHash(varsJSON)
	body, ok := c.cacheGet(ck)
	if !ok {
		body, err = c.gqlFetch(ctx, op, hash, varsJSON)
		if err != nil {
			return err
		}
		c.cache.put(ck, body)
	}

	var env gqlEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("decode %s: %w", op, err)
	}
	if len(env.Errors) > 0 {
		return classifyGQLError(env.Errors[0].Message)
	}
	if len(env.Data) == 0 {
		return ErrNotFound
	}
	return json.Unmarshal(env.Data, out)
}

// gqlFetch performs the POST, paced, retried, and wall-checked.
func (c *Client) gqlFetch(ctx context.Context, op, hash string, varsJSON []byte) ([]byte, error) {
	ext := fmt.Sprintf(`{"persistedQuery":{"version":1,"sha256Hash":%q}}`, hash)
	payload := []byte(fmt.Sprintf(`{"operationName":%q,"variables":%s,"extensions":%s}`,
		op, varsJSON, ext))

	q := url.Values{}
	q.Set("operationName", op)
	q.Set("locale", c.Locale)
	q.Set("currency", c.Currency)
	u := apiV3OnBase(c.BaseURL) + "/" + op + "/" + hash + "?" + q.Encode()

	header := http.Header{}
	header.Set("X-Airbnb-Api-Key", c.apiKey)
	header.Set("Content-Type", "application/json")

	var lastErr error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
		}
		body, retry, err := c.do(ctx, http.MethodPost, u, header, payload)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, lastErr
}

// getJSON fetches an api/v2 JSON endpoint, attaching the public key. The key
// rides as a query parameter, the way the search box sends it.
func (c *Client) getJSON(ctx context.Context, path string, q url.Values, out any) error {
	if q == nil {
		q = url.Values{}
	}
	q.Set("key", c.apiKey)
	q.Set("locale", c.Locale)
	q.Set("currency", c.Currency)
	u := apiV2OnBase(c.BaseURL) + "/" + strings.TrimPrefix(path, "/") + "?" + q.Encode()
	body, err := c.get(ctx, u)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

// cacheGet reads the response cache unless --refresh is set.
func (c *Client) cacheGet(key string) ([]byte, bool) {
	if c.refresh {
		return nil, false
	}
	return c.cache.get(key)
}

// classifyGQLError maps a GraphQL error message to a sentinel. An auth, key, or
// rate signal is the edge wall surfacing through a 200; a not-found signal is a
// genuine miss.
func classifyGQLError(msg string) error {
	low := strings.ToLower(msg)
	switch {
	case strings.Contains(low, "not found"), strings.Contains(low, "no such"):
		return ErrNotFound
	case strings.Contains(low, "rate"), strings.Contains(low, "too many"):
		return ErrRateLimited
	default:
		// An auth, key, persisted-query, or unspecified failure is the wall.
		return ErrBlocked
	}
}

// shortHash returns a short stable hex digest of b, for cache keys.
func shortHash(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8])
}

// apiV3OnBase and apiV2OnBase build the data-plane roots from a base URL, so a
// test can point the whole client (pages and API) at one httptest server.
func apiV3OnBase(base string) string { return strings.TrimRight(base, "/") + "/api/v3" }
func apiV2OnBase(base string) string { return strings.TrimRight(base, "/") + "/api/v2" }
