---
title: "Resource URIs"
description: "Use airbnb as a database/sql-style driver so a host program can address Airbnb as airbnb:// URIs."
weight: 20
---

`airbnb` is a command line, but the `airbnb` Go package is also a small driver
that makes Airbnb addressable as a resource URI. A host program registers it the
way a program registers a database driver with `database/sql`, then dereferences
`airbnb://` URIs without knowing anything about how Airbnb is fetched.

The host that does this today is [ant](https://github.com/tamnd/ant), a single
binary that puts one URI namespace over a family of site tools. The examples
below use `ant`; any program that links the package gets the same behavior.

## Mounting the driver

A host enables the driver with one blank import, exactly like
`import _ "github.com/lib/pq"`:

```go
import _ "github.com/tamnd/airbnb-cli/airbnb"
```

The package's `init` registers a domain with the scheme `airbnb` for the hosts
`www.airbnb.com` and `airbnb.com`. The standalone `airbnb` binary does not
change.

## Addressing records

A URI is `scheme://authority/id`. The resolver types are:

| URI                    | What it is               |
| ---------------------- | ------------------------ |
| `airbnb://room/<id>`   | one listing, keyed by its room id |
| `airbnb://host/<id>`   | a host's public profile  |

```bash
ant get airbnb://room/12345        # the listing record
ant cat airbnb://room/12345        # just the description body
ant get airbnb://host/555          # the host profile
ant url airbnb://room/12345        # the live https URL
ant resolve https://www.airbnb.com/rooms/12345  # a pasted link, back to its URI
```

`room` and `host` are best-effort: from a datacenter they may hit Airbnb's edge
wall and report need-auth, the same as the matching commands. There is no API to
fall back to. See
[what anonymous access reaches](/getting-started/introduction/#what-anonymous-access-reaches).

## Collections

`ls` lists the members of a collection. Each list operation has its own
authority, so they never shadow one another:

| URI                          | What it lists                  |
| ---------------------------- | ------------------------------ |
| `airbnb://search/<place>`    | stays in a place               |
| `airbnb://reviews/<id>`      | a listing's reviews            |
| `airbnb://calendar/<id>`     | a listing's availability days  |
| `airbnb://experiences/<place>` | experiences in a place       |
| `airbnb://listings/<id>`     | a host's public listings       |

```bash
ant ls airbnb://search/Lake%20Tahoe    # the stays in a place
ant ls airbnb://reviews/12345          # the listing's reviews
ant ls airbnb://listings/555           # the host's listings
```

## Walking the graph

Every record carries explicit edges to the records it points at, so a host can
breadth-first crawl the site and write it to disk without scraping URLs out of
free text. The edges are:

| From       | Field     | Edge to                  |
| ---------- | --------- | ------------------------ |
| `Listing`  | `room`    | `airbnb://room/<id>`     |
| `Listing`  | `host`    | `airbnb://host/<id>`     |
| `Room`     | `host_id` | `airbnb://host/<id>`     |
| `Review`   | `room`    | `airbnb://room/<id>`     |
| `Day`      | `room`    | `airbnb://room/<id>`     |

A search listing links straight through to its full room and to its host; a room
links to its host; a review and a calendar day link back to their room. Starting
from any node, `--follow` walks these edges:

```bash
ant export airbnb://search/Lake%20Tahoe --follow 1 --to ./data  # a search, each listing, and its host
ant get airbnb://room/12345
ant cat airbnb://room/12345        # the description body
ant url airbnb://room/12345
```

Each record is written under its minted URI with its edges intact, so the saved
set reconstructs the slice of the site that was reached: the search results, the
full listing behind each card, and the hosts behind those listings.

These edge fields stay out of the table and CSV views (they would be noise in a
terminal) but are always present in the JSON and JSONL a host reads.

## Why this is the same code

The driver and the binary share one definition per operation. A resolver op
answers both `airbnb room` on the command line and `ant get airbnb://room/...`
through a host, from the same handler and the same client. There is no second
implementation to keep in step.
