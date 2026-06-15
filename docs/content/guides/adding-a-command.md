---
title: "Add a command"
description: "Model a real airbnb record and expose it as a command, a route, and a tool at once."
weight: 10
---

An airbnb operation is declared once and shows up everywhere: as a CLI
subcommand, as an HTTP route under `serve`, as an MCP tool, and as an
`airbnb://` URI a host can dereference. You add one by touching three files, and
every surface updates itself. The `room` command is the worked example below.

## 1. Model the record

In `airbnb/types.go`, a struct describes the thing you fetch. The `kit` and
`table` struct tags decide how a host addresses it and how it prints:

```go
type Room struct {
    ID          string   `json:"id" kit:"id"`                          // the room id, the URI id
    Name        string   `json:"name,omitempty"`
    Description  string   `json:"description,omitempty" kit:"body"`     // what cat and Markdown print
    PropertyType string  `json:"property_type,omitempty"`
    Price       float64  `json:"price,omitempty"`                      // nightly, when dates are given
    Currency    string   `json:"currency,omitempty"`
    Rating      float64  `json:"rating,omitempty"`
    HostID      string   `json:"host_id,omitempty" kit:"link,kind=airbnb/host"` // an edge to the host
    URL         string   `json:"url"`
}
```

- `kit:"id"` marks the field that becomes the URI id.
- `kit:"body"` marks the prose that `cat` and the Markdown export render.
- `kit:"link,kind=<scheme>/<type>"` marks an outbound edge. `Room.host_id` points
  at the host, which is what lets a host program walk the graph.
- `json:",omitempty"` keeps a record honest: a field Airbnb did not serve is
  absent rather than zero.

## 2. Fetch it

In `airbnb/room.go`, a client method returns the record. The client is a single
web and GraphQL client; there is no API backend to fall back to, so an error
flows straight up:

```go
func (c *Client) GetRoom(ctx context.Context, ref string) (*Room, error) {
    id := roomID(ref) // accept a bare id or a /rooms/ URL
    body, err := c.get(ctx, c.BaseURL+"/rooms/"+id)
    if err != nil {
        return nil, err // ErrBlocked, ErrRateLimited, ErrNotFound flow up unchanged
    }
    // read the data-deferred-state-0 JSON island, parse it into a Room ...
    return r, nil
}
```

The static detail comes from the page island. A date-specific nightly price is
filled from the GraphQL sections only when `--checkin` and `--checkout` are
given.

## 3. Declare the operation

In `airbnb/ops.go`, add an input struct and a handler. The struct tags tell
`kit` what is a positional argument and where the client is injected:

```go
type roomRef struct {
    ID     string  `kit:"arg" help:"room id or /rooms/ URL"`
    Client *Client `kit:"inject"`
}

func getRoom(ctx context.Context, in roomRef, emit func(*Room) error) error {
    r, err := in.Client.GetRoom(ctx, in.ID)
    if err != nil {
        return mapErr(err)
    }
    return emit(r)
}
```

Then register it in `Register` in `airbnb/domain.go`:

```go
kit.Handle(app, kit.OpMeta{
    Name: "room", Group: "read", Single: true,
    Summary: "Show one listing by id",
    URIType: "room", Resolver: true,
    Args: []kit.Arg{{Name: "id", Help: "room id or /rooms/ URL"}},
}, getRoom)
```

That is the whole change. `kit.Handle` reflects the input for flags and the
output for the record shape, so the operation immediately becomes:

```bash
airbnb room 12345                       # the command
curl 'localhost:7777/v1/room/12345'     # the route, under serve
ant get airbnb://room/12345             # the URI dereference, via a host
```

## Resolver ops and list ops

Two flags shape how a host treats an operation:

- **`Single: true`** with **`Resolver: true`** marks the canonical one-record
  fetch for a `URIType`. It answers `ant get`. `room` and `host show` are the
  resolvers.
- **`List: true`** marks a member-lister for a parent resource. It answers
  `ant ls`. A list op emits records that are themselves addressable, so every
  member is a URI a host can follow. `search`, `reviews`, `calendar`,
  `experiences`, and `host listings` do this, each tagged with its own
  collection authority so they never shadow one another.

## Map errors to exit codes

Return through `mapErr` so every surface reports the same outcome with the same
exit code: the edge wall reads as need-auth (exit 4), a throttle as rate-limited
(exit 5), a missing listing as not-found (exit 6):

```go
case errors.Is(err, ErrNotFound):
    return errs.NotFound("%s", err.Error())
case errors.Is(err, ErrRateLimited):
    return errs.RateLimited("%s", err.Error())
case errors.Is(err, ErrBlocked):
    return errs.NeedAuth("%s", err.Error())
```

See [output formats](/reference/output/) for how records render, and
[resource URIs](/guides/resource-uris/) for how a host addresses them.
