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
free text. A resolver edge names a bare field and points at one record; a
collection edge carries the parent id under a `<name>_ref` field and points at a
list authority. The edges are:

| From      | Field          | Edge to                  |
| --------- | -------------- | ------------------------ |
| `Place`   | `search_ref`   | `airbnb://search/<name>` |
| `Listing` | `room`         | `airbnb://room/<id>`     |
| `Listing` | `host`         | `airbnb://host/<id>`     |
| `Room`    | `host_id`      | `airbnb://host/<id>`     |
| `Room`    | `reviews_ref`  | `airbnb://reviews/<id>`  |
| `Room`    | `calendar_ref` | `airbnb://calendar/<id>` |
| `Review`  | `room`         | `airbnb://room/<id>`     |
| `Review`  | `author_id`    | `airbnb://host/<id>`     |
| `Day`     | `room`         | `airbnb://room/<id>`     |
| `Host`    | `listings_ref` | `airbnb://listings/<id>` |

The edges close into one connected graph. A suggestion fans out into a stay
search for the place; a search card walks straight through to its full room and
its host; a room reaches its host, its reviews, and its calendar; a review
reaches its listing and the reviewer's own profile; a host reaches the host's
other listings. No node is left without an outward edge, so a crawl started
anywhere reaches the rest of the reachable site. Starting from any node,
`--follow` walks these edges:

```bash
ant export airbnb://search/Lake%20Tahoe --follow 2 --to ./data  # each listing's room, then its host, reviews, and calendar
ant get airbnb://room/12345
ant cat airbnb://room/12345        # the description body
ant url airbnb://room/12345
```

Each record is written under its minted URI with its edges intact, so the saved
set reconstructs the slice of the site that was reached: the search results, the
full listing behind each card, its reviews and calendar, the hosts behind those
listings, and the profile of each reviewer.

These edge fields stay out of the table and CSV views (they would be noise in a
terminal) but are always present in the JSON and JSONL a host reads.

## Why this is the same code

The driver and the binary share one definition per operation. A resolver op
answers both `airbnb room` on the command line and `ant get airbnb://room/...`
through a host, from the same handler and the same client. There is no second
implementation to keep in step.
