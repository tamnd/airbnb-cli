---
title: "CLI"
description: "Every command and subcommand, with the flags that matter."
weight: 10
---

```
airbnb <command> [arguments] [flags]
```

Run `airbnb <command> --help` for the full flag list on any command.

## Commands

| Command | What it does |
|---|---|
| `search <place>` | Stay search by place (best-effort, may hit the edge wall) |
| `room <id>` | Show one listing by id (best-effort, may hit the edge wall) |
| `reviews <id>` | List a listing's reviews (best-effort) |
| `calendar <id>` | A listing's availability and nightly price (best-effort) |
| `experiences <place>` | Search experiences in a place (best-effort) |
| `suggest <prefix>` | Location autocomplete suggestions (best-effort) |
| `host show <id>` | Show a host's public profile (best-effort) |
| `host listings <id>` | List a host's public listings (best-effort) |
| `ref id <ref>` | Classify a reference into its (kind, id), offline |
| `ref url <kind> <id>` | Build the canonical URL for a (kind, id), offline |
| `serve [--addr]` | Serve the operations over HTTP as NDJSON |
| `mcp` | Run as an MCP server over stdio |
| `version` | Print the version and exit |

A listing is addressed by its numeric room id, like `12345`, or a `/rooms/` URL.
A host is addressed by its user id, like `555`, or a `/users/show/` URL. A
reference can also be an `/experiences/` path, a full Airbnb URL, or a pasted
GraphQL global id. Every live surface is walled from datacenter IPs; see
[what anonymous access reaches](/getting-started/introduction/#what-anonymous-access-reaches).

A nightly price only exists for specific dates, so `room` and `search` leave the
price empty unless you give `--checkin` and `--checkout` (with `--adults` and
`--children`).

## Global flags

These are shared by every operation, so they work the same on every command.

| Flag | Meaning |
|---|---|
| `-o, --output` | Output format: `auto`, `table`, `json`, `jsonl`, `csv`, `tsv`, `url`, `raw` |
| `--fields` | Comma-separated columns to keep |
| `--template` | Go text/template applied per record |
| `--no-header` | Omit the header row in `table` and `csv` |
| `-n, --limit` | Stop after N records (0 means no limit) |
| `--rate` | Minimum delay between requests |
| `--retries` | Retry attempts on rate limit or 5xx |
| `--timeout` | Per-request timeout |
| `--data-dir` | Override the data directory |
| `--no-cache` | Bypass on-disk caches |
| `--db` | Tee every record into a store (e.g. `out.db`, `postgres://...`) |
| `-v, --verbose` | Increase verbosity (repeatable) |
| `-q, --quiet` | Suppress progress output |
| `--color` | `auto`, `always`, or `never` |
| `--cache-ttl` | How long a cached response stays fresh |
| `--refresh` | Fetch fresh copies and rewrite the cache, ignoring any hit |

## Airbnb flags

These tune how `airbnb` reads the site and what a price quote covers.

| Flag | Meaning |
|---|---|
| `--user-agent` | Override the User-Agent sent with each request |
| `--locale` | Locale for localized strings (default `en`) |
| `--currency` | Currency for prices (default `USD`) |
| `--api-key` | Override the public web key (not a credential; see below) |
| `--checkin` | Stay check-in date `YYYY-MM-DD` (makes a nightly price appear) |
| `--checkout` | Stay check-out date `YYYY-MM-DD` |
| `--adults` | Number of adults for the price quote (default 1) |
| `--children` | Number of children for the price quote |

`--api-key` overrides only the public web key the site already ships. It is not a
secret and there is no signed or affiliate API behind it. See
[configuration](/reference/configuration/#the-public-web-key).

See [output formats](/reference/output/) for what `-o`, `--fields`, and
`--template` produce, and [configuration](/reference/configuration/) for
environment variables and defaults.
