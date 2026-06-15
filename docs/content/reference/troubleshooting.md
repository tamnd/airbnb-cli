---
title: "Troubleshooting"
description: "The handful of things that trip people up, and how to fix each one."
weight: 40
---

Most of these come down to network reality or how Airbnb serves its data, not a
bug. Each case maps to an exit code so a script can tell them apart.

## A command exits 4

Every live surface sits behind Airbnb's edge bot manager (Cloudflare and
DataDome), which classifies a request before the application sees it on IP
reputation and TLS fingerprint, and walls datacenter IPs. From a home or mobile
network the surfaces usually answer; from a datacenter or a cloud host they hit
the wall, and `airbnb` reports need-auth (exit 4) rather than pretending the
result was empty. There is no official API to fall back to, so the only remedy is
to run from a residential or mobile network. `airbnb` does not forge a TLS
fingerprint and does not rent or rotate IPs to get past the edge. See
[what anonymous access reaches](/getting-started/introduction/#what-anonymous-access-reaches).

The `ref id` and `ref url` commands are not affected; they resolve offline with
no request.

## Requests start failing or returning 429

Airbnb rate-limits like any public site. `airbnb` already paces requests and
retries the transient failures, but a hard limit still means backing off, and it
reports rate-limited (exit 5). Raise the delay between requests with `--rate`
(for example `--rate 1s`) and retry later. A burst of 429 or 5xx responses is the
site asking you to slow down, not a defect.

## A reference is not found

An unknown id, a removed listing, or a reference `airbnb` cannot classify reports
not-found (exit 6). Check that the id is spelled the way Airbnb uses it, and that
the listing still exists in a private browser window before assuming it is gone.

## A listing or search comes back with no price

A nightly price only exists for specific dates, so `room` and `search` leave the
price empty unless you give `--checkin` and `--checkout` (with `--adults` and
`--children`). Add the dates and the price fills in. The price and its currency
are read together, so each record carries an explicit `currency` field alongside
the number; set `--currency` and `--locale` to ask for another.

## The binary is not on your PATH

`go install` puts the binary in `$(go env GOPATH)/bin` (usually `~/go/bin`), and
a release archive leaves it wherever you unpacked it. If your shell cannot find
`airbnb`, add that directory to your `PATH`. See
[installation](/getting-started/installation/).

## Seeing what airbnb actually did

When something behaves unexpectedly, `-v` adds per-request detail so you can see
the URLs it hit and the responses it got. That is usually enough to tell the edge
wall apart from a rate limit apart from a genuinely empty result. Add
`--no-cache` to force a fresh fetch when you suspect a stale cached page.
