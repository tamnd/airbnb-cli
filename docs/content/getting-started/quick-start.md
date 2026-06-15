---
title: "Quick start"
description: "Run your first airbnb commands and shape their output."
weight: 30
---

Once `airbnb` is on your `PATH`, complete a place name. `suggest` reads the
location autocomplete endpoint (best-effort behind the edge):

```bash
airbnb suggest paris -n 6 --fields name
```

```
╭──────────────────────────╮
│ NAME                     │
├──────────────────────────┤
│ Paris, France            │
│ Paris, TX                │
│ Paris, Ontario, Canada   │
│ Paris, KY                │
│ Paris, Tennessee         │
│ Parisian, ...            │
╰──────────────────────────╯
```

Ask for JSON when you want to pipe it:

```bash
airbnb suggest paris -o json
```

```json
[
  {
    "query": "paris",
    "name": "Paris, France",
    "place_id": "ChIJD7fiBh9u5kcRYJSMaMOCCwQ",
    "lat": 48.8566,
    "lng": 2.3522
  }
]
```

## The best-effort surfaces

Every live surface sits behind Airbnb's edge bot manager and may exit 4 from a
datacenter. See
[what anonymous access reaches](/getting-started/introduction/#what-anonymous-access-reaches).

```bash
airbnb search "Lake Tahoe"           # stay search by place (best-effort)
airbnb room 12345                    # one listing by id (best-effort)
airbnb reviews 12345                 # a listing's reviews (best-effort)
airbnb calendar 12345                # availability and nightly price (best-effort)
airbnb host show 555                 # a host profile (best-effort)
airbnb host listings 555             # the host's public listings (best-effort)
airbnb experiences "Lake Tahoe"      # experience search (best-effort)
```

There is no API to fall back to, so a walled read exits 4 and names the only
remedy: run from a residential or mobile network.

## Prices need dates

A nightly price only exists for specific dates, so `room` and `search` leave the
price empty unless you give `--checkin` and `--checkout`:

```bash
airbnb room 12345 --checkin 2025-07-01 --checkout 2025-07-05 --adults 2
```

## Shape the output

The same flags work on every command:

```bash
airbnb search "Lake Tahoe" --fields name,price,rating
airbnb room 12345 --template '{{.Name}} {{.Price}} {{.Currency}}'
airbnb suggest paris -o jsonl | jq .name
```

`-o` takes `table`, `json`, `jsonl`, `csv`, `tsv`, `url`, or `raw`. Left to
`auto`, it prints a table to a terminal and JSONL into a pipe, so the same
command reads well by hand and parses cleanly downstream. See
[output formats](/reference/output/) for the full contract.

## Resolve a reference offline

The `ref` commands classify and build Airbnb references with no network call:

```bash
airbnb ref id "https://www.airbnb.com/rooms/Cozy-Cabin/12345"
airbnb ref url experience 777
```

A reference can be a bare numeric id, a `/rooms/`, `/users/show/`, or
`/experiences/` path, a full Airbnb URL, or a pasted GraphQL global id (the
base64 of `<Type>:<digits>`).

## Serve it instead

The same operations are available over HTTP and to agents over MCP:

```bash
airbnb serve --addr :7777 &
curl -s 'localhost:7777/v1/suggest/paris'   # NDJSON, one record per line
airbnb mcp                                   # MCP over stdio
```

## What to read next

The [guides](/guides/) cover the common jobs, and the
[CLI reference](/reference/cli/) is the full command tree and flag list.
