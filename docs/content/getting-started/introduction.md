---
title: "Introduction"
description: "What airbnb is, how it is put together, and which surfaces it reads."
weight: 10
---

`airbnb` reads public Airbnb data the way a logged-out browser does: location
autocomplete, a single listing, a stay search by place, a listing's reviews, its
availability and nightly prices, a host profile and the host's listings, and
experience search. It is a single binary. It speaks to Airbnb over plain HTTPS,
lifts the data out of the `data-deferred-state-0` JSON island a listing page
embeds and the internal GraphQL endpoint its own web client calls, and gets out
of your way. There is no API key, no login, and nothing to run alongside it.

## How it is built

- A **library package** (`airbnb`) holds the web and GraphQL client, the listing
  page island parser, and the typed data models. It paces requests, sends a
  browser User-Agent because that is what a logged-out reader looks like, caches
  on disk, and retries the transient failures any public site throws under load.
- A **domain** (`airbnb/domain.go`) declares each operation once on the
  [any-cli/kit](https://github.com/tamnd/any-cli) framework. That single
  declaration becomes a CLI command, an HTTP route, an MCP tool, and a
  resource-URI dereference. It is the one place you add to the tool.
- A thin **`cmd/airbnb`** hands the assembled app to `kit.Run`, which builds the
  command tree and the serve and mcp surfaces.

## One operation, four surfaces

Because an operation is surface-neutral, the same `room` you run on the command
line is also a route and a tool:

```bash
airbnb room 12345                 # the command
airbnb serve --addr :7777         # GET /v1/room/12345
airbnb mcp                        # the room tool, over stdio
ant get airbnb://room/12345       # the URI dereference (via a host)
```

## What anonymous access reaches

Airbnb fronts its whole site with an edge bot manager (Cloudflare and DataDome)
that classifies a request before the application sees it, on IP reputation and
TLS fingerprint, and hard-walls datacenter IPs. `airbnb` sorts the surfaces into
what works regardless and what is gated, and never pretends the line is
elsewhere.

Works from any network, offline:

- `ref id`, `ref url` (pure string resolution, no request)

Walled from datacenter IPs, best-effort (everything live):

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
Every live read is best-effort behind the edge; there is no live surface that is
reliable from any network.

Records carry only fields a logged-out reader can fill. There is no trip, no
message thread, no wishlist, no host dashboard, and no payout data, because none
of that exists without an account. A listing shows the title, the description,
the icon highlights, the capacity and room breakdown, the sleeping arrangements,
the amenities, the house rules with the check-in and checkout times, the area,
the guest rating and its six category scores, the full photo gallery, and the
host a visitor sees with their public photo; a search card carries its photo
carousel and any struck-through original price; a review carries the stay
descriptor, the reviewer's photo, and a link to the reviewer's own profile; a
host carries the response time and where they live. A field a page does not show
is left empty rather than guessed.

A nightly price only exists for specific dates, so `room` and `search` leave the
price empty unless you give `--checkin` and `--checkout` (with `--adults` and
`--children`).

## Scope

`airbnb` is a read-only client over data Airbnb already serves publicly. It reads
that data and shapes it for you. That narrow scope keeps it a single small binary
with no database, no daemon, and no setup.

`airbnb` is an independent tool and is not affiliated with Airbnb.

Next: [install it](/getting-started/installation/), then take the
[quick start](/getting-started/quick-start/).
