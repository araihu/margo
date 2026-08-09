---
title: Inside the Margo rendering atelier
description: A practical tour of reusable HTML, optional publication policy, and resilient image delivery.
language: en
slug: inside-margo-rendering-atelier
authors:
  - Margo Studio
publishedAt: "2026-08-09T15:30:00-03:00"
modifiedAt: "2026-08-09T15:30:00-03:00"
tags:
  - HTML
  - Publishing
  - Images
---

# Inside the Margo rendering atelier

A useful document tool should make strong technical guarantees without pretending to own the consumer's editorial identity. That boundary now shapes Margo's HTML stack.

## The core stays technical

`RenderHTML` produces one immutable fragment and its requirements. `RenderHTMLPage` adds a complete document shell, theme, color mode, and dependency strategy. Neither API knows a canonical domain, publishing route, Open Graph identity, or article type.

This example adds those claims in consumer-local components passed through `HTMLPageInput`. PDF and other consumers can keep using the semantic result or generic HTML without inheriting web-publication policy.

## One hero, three delivery formats

The hero above is one generated illustration encoded as AVIF, WebP, and JPEG. The browser chooses the first supported source while the JPEG remains the durable fallback.

## PNG for the high-fidelity study

The lossless PNG keeps the fine paper grain, translucent color swatches, and small tonal transitions of the format study.

![A square editorial still life with photo proofs, color swatches, a camera lens, and printing tools.](assets/format-study.png)

## GIF as a compatibility baseline

GIF is deliberately included as a familiar legacy format. It is not the smallest or richest choice here; it proves the generated page does not silently assume only modern decoders.

![The same editorial format study encoded as GIF.](assets/format-study.gif)

## A resilient publishing contract

The important distinction is now explicit: fragment versus complete document is an engine choice; public-web authority is entirely a consumer choice. Images follow the same principle—modern sources where useful, conservative fallbacks where necessary.
