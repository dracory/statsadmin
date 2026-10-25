---
created: '2026-10-25'
priority: high
status: done
tags: []
title: Session grouping in statsstore (30-min window)
---

Prerequisite for bounce rate, session duration, and entry/exit pages.

Group visitor rows into sessions: same fingerprint (fallback IP+UA) with gaps <= 30 min.

Likely lives in statsstore: add `session_id` to the visitor table or compute session windows in queries. Expose `SessionQuery` / `SessionList` / `SessionCount`.

See docs/compare/openpanel.md