---
title: "airbnb"
description: "airbnb reads public Airbnb data (best-effort listings, stay search, reviews, calendars, hosts, experiences, and location autocomplete) into structured records over a CLI, an HTTP server, and an MCP tool set."
heroTitle: "airbnb, from the command line"
heroLead: "Resolve any Airbnb reference offline, complete a place into its search id, and best-effort open a listing, run a stay search, read a listing's reviews and availability, show a host profile and the host's listings, or search experiences. One pure-Go binary, no API key, output that pipes into the rest of your tools, and a resource-URI driver other programs can address."
heroPrimaryURL: "/getting-started/quick-start/"
heroPrimaryText: "Get started"
---

`airbnb` reads the public Airbnb pages a logged-out browser sees, lifts the data
out of the `data-deferred-state-0` JSON island a listing page embeds and the
internal GraphQL endpoint its own web client calls, and gets out of your way.

```bash
airbnb suggest paris              # location autocomplete (best-effort)
airbnb search "Lake Tahoe"        # stay search by place (best-effort)
airbnb room 12345                 # one listing by id (best-effort)
airbnb reviews 12345              # a listing's reviews (best-effort)
airbnb serve --addr :7777         # the same operations over HTTP
```

There is no API key, no login, and nothing to run alongside it. Output adapts to
where it goes: an aligned table on your terminal, JSONL the moment you pipe it
somewhere.

## Honest about what is reachable

Airbnb fronts its whole site with an edge bot manager that classifies a request
before the application sees it, on IP reputation and TLS fingerprint, and
hard-walls datacenter IPs. `airbnb` is explicit about the line. The reference
resolver is offline and always works. Every live surface (listings, search,
reviews, calendars, hosts, experiences, autocomplete) sits behind the edge, so
those reads are best-effort. There is no official API to fall back to, so a
walled read exits 4 and names the only remedy: run from a residential or mobile
network. See
[what anonymous access reaches](/getting-started/introduction/#what-anonymous-access-reaches).

## Two ways to use it

- **As a command** for reading Airbnb by hand or in a script. Start with the
  [quick start](/getting-started/quick-start/).
- **As a resource-URI driver** so a host like
  [ant](https://github.com/tamnd/ant) can address Airbnb as `airbnb://` URIs
  and follow links across sites. See [resource URIs](/guides/resource-uris/).

Both are the same code: one operation, declared once, is a CLI command, an HTTP
route, an MCP tool, and a URI dereference.

## Where to go next

- New here? Read the [introduction](/getting-started/introduction/), then the
  [quick start](/getting-started/quick-start/).
- Installing? See [installation](/getting-started/installation/).
- Doing a specific job? The [guides](/guides/) are task-first.
- Need every flag? The [CLI reference](/reference/cli/) is the full surface.
