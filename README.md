# airbnb

A command line for [Airbnb](https://www.airbnb.com). One binary that resolves any
Airbnb reference offline, completes a place into its search id, and best-effort
opens a listing, runs a stay search, reads a listing's reviews and availability,
shows a host profile and the host's listings, or searches experiences. No API
key, no login, nothing to run alongside it.

```
airbnb ref id "https://www.airbnb.com/rooms/Cozy-Cabin/12345" -o json
```

```json
[
  {
    "input": "https://www.airbnb.com/rooms/Cozy-Cabin/12345",
    "kind": "room",
    "id": "12345",
    "url": "https://www.airbnb.com/rooms/12345"
  }
]
```

On a terminal the table header and JSON values are colorized; piped to a file or
another program the output drops to plain text so it parses cleanly. Use
`--color always` to keep color through a pipe, or `--color never` to drop it.

Full documentation: [airbnb-cli.tamnd.com](https://airbnb-cli.tamnd.com).

## Why

Reading Airbnb programmatically has no official path at all: there is no public
API, and the partner API is invitation-only. `airbnb` reads the same public pages
a logged-out visitor sees, lifts the data out of the `data-deferred-state-0` JSON
island Airbnb embeds in a listing page and the internal GraphQL endpoint its own
web client calls, and shapes each surface into a clean record with real output
formats and pipelines that compose.

Airbnb fronts its whole site with an edge bot manager, and this tool is honest
about what that leaves reachable. The reference resolver is offline and always
works. Every live surface sits behind the edge, which classifies a request on IP
reputation and TLS fingerprint and hard-walls datacenter IPs; those reads are
best-effort. There is no API to fall back to, so a walled read exits 4 and names
the remedy. See [what anonymous access reaches](#what-anonymous-access-reaches).

## Install

```sh
go install github.com/tamnd/airbnb-cli/cmd/airbnb@latest
```

Or grab a prebuilt binary from the [releases page](https://github.com/tamnd/airbnb-cli/releases).
The binary is pure Go with no runtime dependencies. You can also run the
container image:

```sh
docker run --rm ghcr.io/tamnd/airbnb:latest --help
```

Build from source:

```sh
git clone https://github.com/tamnd/airbnb-cli
cd airbnb-cli
make build      # produces ./bin/airbnb
```

## Quick start

```sh
airbnb suggest paris                  # location autocomplete (best-effort)
airbnb search "Lake Tahoe"            # stay search by place (best-effort)
airbnb room 12345                     # one listing by id (best-effort)
airbnb reviews 12345                  # a listing's reviews (best-effort)
airbnb calendar 12345                 # availability and nightly price (best-effort)
airbnb host show 555                  # a host profile (best-effort)
airbnb host listings 555              # the host's public listings (best-effort)
airbnb experiences "Lake Tahoe"       # experience search (best-effort)
```

Most commands accept a bare id, a `/rooms/`, `/users/show/`, or `/experiences/`
path, a full Airbnb URL, or a pasted GraphQL global id wherever they take a
reference. The `ref` commands resolve those offline, with no network call:

```sh
airbnb ref id "/users/show/555" -o json
airbnb ref url experience 777
```

A nightly price only exists for specific dates, so `room` and `search` leave the
price empty unless you give `--checkin` and `--checkout`:

```sh
airbnb room 12345 --checkin 2025-07-01 --checkout 2025-07-05 --adults 2
```

## How it works

A listing page renders server-side and ships its whole client state as a
`data-deferred-state-0` JSON island. `airbnb` GETs the page and reads that island
for the static detail. Every other surface (search, reviews, calendar, host,
experiences) is the logged-out web client's own GraphQL endpoint (`api/v3`),
addressed by the public web key the site ships and a per-operation persisted-query
hash; autocomplete is a lighter `api/v2` JSON endpoint. It paces and caches
requests and retries the transient failures, and it sends a browser user-agent
because that is what a logged-out reader looks like. No API key, no token.

Prices are read in whatever currency Airbnb serves, so each record carries an
explicit `currency` field alongside the number. Set `--currency` and `--locale` to
ask for another.

## What anonymous access reaches

Airbnb fronts its site with an edge bot manager (Cloudflare and DataDome) that
classifies a request before the application sees it, on IP reputation and TLS
fingerprint. This tool sorts the surfaces into what works regardless and what is
gated, and never pretends the line is elsewhere.

Works from any network, offline:

- `ref id`, `ref url` (pure string resolution, no request)

Walled from datacenter IPs, best-effort:

- `room` (the `/rooms/` page island)
- `search` (the `StaysSearch` GraphQL operation)
- `reviews` (the `StaysPdpReviewsQuery` operation)
- `calendar` (the `PdpAvailabilityCalendar` operation)
- `host show`, `host listings` (the `GetUserProfile` and `BeehiveUserListings` operations)
- `experiences` (the `ExperiencesSearch` operation)
- `suggest` (the `api/v2` autocomplete)

From a home or mobile network the best-effort surfaces usually answer; from a
datacenter they hit the edge wall and exit 4. There is no official API to fall
back to, so the only remedy the tool names is to run from a residential or mobile
network. It does not forge a TLS fingerprint and does not rent or rotate IPs to
get past the edge: it reads what a logged-out browser reads, the way it reads it.

Records carry only fields a logged-out reader can fill. There is no trip, no
message thread, no wishlist, no host dashboard, and no payout data, because none
of that exists without an account. A listing shows the title, the description, the
icon highlights, the capacity and room breakdown, the sleeping arrangements, the
amenities, the house rules with the check-in and checkout times, the area, the
guest rating and its six category scores, the full photo gallery, and the host a
visitor sees; a search card carries its photo carousel and any struck-through
original price; a review carries the stay descriptor, the reviewer's photo, and a
link to the reviewer's own profile; a host carries the response time. A field a
page does not show is left empty rather than guessed.

When something is genuinely missing the exit code says which, so a script can tell
the cases apart:

| Exit | Meaning |
| --- | --- |
| 0 | ok |
| 2 | usage error |
| 3 | no results (the resource is genuinely empty) |
| 4 | need auth, or the edge bot wall |
| 5 | rate limited (raise `--rate`) |
| 6 | not found (unknown id, removed listing, bad reference) |
| 8 | network error |

## Commands

| Command | What it does |
| --- | --- |
| `search <place>` | Stay search by place (best-effort, see above) |
| `room <id>` | One listing by id (best-effort, see above) |
| `reviews <id>` | A listing's reviews (best-effort) |
| `calendar <id>` | A listing's availability and nightly price (best-effort) |
| `experiences <place>` | Experience search by place (best-effort) |
| `suggest <prefix>` | Location autocomplete suggestions |
| `host show <id>` | A host's public profile (best-effort) |
| `host listings <id>` | A host's public listings (best-effort) |
| `ref id <ref>` | Classify a reference into its (kind, id), offline |
| `ref url <kind> <id>` | Build the canonical URL for a (kind, id), offline |
| `serve` | Serve the same operations over HTTP as NDJSON |
| `mcp` | Serve the same operations to an agent over MCP |
| `version` | Print version, commit, and build date |

A listing is addressed by its numeric room id, like `12345`, or a `/rooms/` URL; a
host by its user id, like `555`, or a `/users/show/` URL. Run
`airbnb <command> --help` for the full flag list on any command.

## Output

Every command shares one output contract. The default adapts to where output
goes, a table on a terminal and JSONL in a pipe, so the same command reads well by
hand and parses cleanly downstream.

```sh
airbnb search "Lake Tahoe" -n 4 --fields name,price,rating
```

Pick the format with `-o table|markdown|json|jsonl|csv|tsv|url|raw`, choose
columns with `--fields a,b,c`, render a custom line with `--template`, drop the
header with `--no-header`, and cap results with `-n/--limit`. The `url` format
prints just the canonical URL of each record, which is handy for piping into
another tool.

## Recipes

Resolve a pile of pasted links to their (kind, id) offline:

```sh
airbnb ref id "https://www.airbnb.com/rooms/12345" -o json | jq '{kind, id}'
```

A stay search as JSON, piped to jq (best-effort):

```sh
airbnb search "Lake Tahoe" --checkin 2025-07-01 --checkout 2025-07-05 -o json \
  | jq '{name, price, currency, rating}'
```

The canonical URLs of a search, one per line:

```sh
airbnb search "Lake Tahoe" -n 50 -o url
```

Tee a host's listings into a local SQLite store, keyed by each room id:

```sh
airbnb host listings 555 --db airbnb.db
```

## Serve it

The same operations are available over HTTP and as an MCP tool set for agents,
with no extra code:

```sh
airbnb serve --addr :7777    # GET /v1/... returns NDJSON
airbnb mcp                   # speak MCP over stdio
```

## Use it as a resource-URI driver

`airbnb` registers an `airbnb` domain the way a program registers a database
driver with `database/sql`. A host enables it with one blank import:

```go
import _ "github.com/tamnd/airbnb-cli/airbnb"
```

Then [ant](https://github.com/tamnd/ant) (or any program that links the package)
dereferences `airbnb://` URIs without knowing anything about Airbnb:

```sh
ant get airbnb://room/<id>        # fetch a listing
ant cat airbnb://room/<id>        # just the description body
ant ls  airbnb://reviews/<id>     # a listing's reviews, each addressable
ant get airbnb://host/<id>        # fetch a host profile
ant url airbnb://room/<id>        # the live https URL
```

Records carry explicit edges that close into one connected graph, so a host can
breadth-first crawl it and write it to disk: a suggestion fans out into a search
for the place, a search listing links to its full room and its host, a room links
to its host, its reviews, and its calendar, a review links back to its listing and
on to the reviewer's own profile, and a host links to the host's other listings.
No node is left without an outward edge, so a crawl started anywhere reaches the
rest of the reachable site. `ant export <uri> --follow N` walks those edges. See
the [resource-URI guide](https://airbnb-cli.tamnd.com/guides/resource-uris/) for
the full edge map.

## Development

```
cmd/airbnb/   thin main: hands cli.NewApp to kit.Run
cli/          assembles the kit App from the airbnb domain
airbnb/       the library: web client, GraphQL client, the island parser, data
              models, and domain.go (the driver)
docs/         tago documentation site
```

```sh
make build      # ./bin/airbnb
make test       # go test ./...
make vet        # go vet ./...
```

Every read command is declared once as a kit operation in `airbnb/domain.go`. That
single declaration becomes the CLI subcommand, the HTTP route, and the MCP tool,
so the three surfaces never drift.

## Releasing

Push a version tag and GitHub Actions runs GoReleaser, which builds the archives,
Linux packages, the multi-arch GHCR image, checksums, SBOMs, and a cosign
signature:

```sh
git tag v0.1.0
git push --tags
```

The Homebrew and Scoop steps self-disable until their tokens exist, so the first
release works with no extra secrets.

## License

`airbnb` is an independent tool and is not affiliated with Airbnb. Apache-2.0,
see [LICENSE](LICENSE).
