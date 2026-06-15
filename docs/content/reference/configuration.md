---
title: "Configuration"
description: "Environment variables, the public web key, defaults, and the data directory."
weight: 20
---

`airbnb` needs almost no configuration: it runs anonymously against public data
out of the box. The settings below let you tune politeness, set the locale and
currency, and choose where data lands.

## Defaults

| Setting | Default | Flag |
|---|---|---|
| Requests | paced and retried on 429/5xx | `--rate`, `--retries` |
| Per-request timeout | 30s | `--timeout` |
| Locale | `en` | `--locale` |
| Currency | `USD` | `--currency` |
| Adults for a price quote | 1 | `--adults` |
| On-disk cache | under the data directory | `--no-cache` to bypass |

## The public web key

Every live surface is the logged-out web client's own data plane. Airbnb's pages
address its internal GraphQL endpoint with a public web key the site ships in its
own bundle, the same constant any visitor's browser sends. `airbnb` uses that key
automatically and refreshes it from the live site when reachable, so there is
nothing to set.

`--api-key` overrides that public key if you have a reason to. It is not a
credential: there is no signed API, no affiliate or partner backend, no consumer
id, and no private key. There is no API to fall back to when a request is walled,
so the only remedy for the edge wall is to run from a residential or mobile
network. See
[what anonymous access reaches](/getting-started/introduction/#what-anonymous-access-reaches).

## The data directory

Caches and any record store live under one data directory, chosen in this order:

1. `--data-dir`
2. `AIRBNB_DATA_DIR`
3. `$XDG_DATA_HOME/airbnb`
4. `~/.local/share/airbnb`

## Environment variables

Every flag has an environment fallback, prefixed `AIRBNB_` in upper case with
dashes as underscores. For example:

```bash
export AIRBNB_RATE=1s          # same as --rate 1s
export AIRBNB_CURRENCY=EUR     # same as --currency EUR
export AIRBNB_DATA_DIR=~/data/airbnb
```

Flags win over environment variables, which win over the built-in defaults.

## Sending records to a store

`--db` tees every emitted record into a store as a side effect of reading, so a
session fills a local database without a separate import step:

```bash
airbnb host listings 555 --db out.db        # SQLite file
airbnb host listings 555 --db 'postgres://...'
```
