---
created: '2026-10-25'
priority: high
status: in_progress
tags: []
title: Capture UTM + source/medium/campaign at registration
---

Parse utm_source, utm_medium, utm_campaign, utm_term, utm_content (+ referrer host) in statsstore `VisitorRegister` and store as new columns.

Requires migration for the visitor table. Query filters: WithSource/WithMedium/WithCampaign.

See docs/compare/openpanel.md