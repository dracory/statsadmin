# e2e

End-to-end tests that drive the statsadmin panel in a real headless Chrome
via [chromedp](https://github.com/chromedp/chromedp) (CDP). Unlike go-rod,
chromedp has no `leakless.exe` helper, so it avoids the Windows Defender
false positive entirely.

## How it works

- `TestMain` mounts the admin on `httptest.NewServer` (in-process, random
  port) backed by a seeded in-memory SQLite `statsstore` — no binary build
  needed. Four visitor records are seeded across three IPs (one a
  self-identifying Googlebot UA with the bot flag left `no`).
- One shared browser is launched via `chromedp.NewExecAllocator` for the
  whole run; each test opens a fresh tab with a 30s deadline (`newTab`).
- Chrome binary resolution order: `CHROME_BIN` env → Windows Playwright
  `chromium_headless_shell` → WSL-side Windows shell → Linux Playwright
  shell → chromedp's default lookup (Chrome on PATH / install dirs).

## Run

```bash
go test -v ./e2e/     # or: task e2e
```

Note: on WSL, run tests with the Windows Go toolchain (`go.exe`) —
Windows-side Chrome binds 127.0.0.1, which WSL cannot reach (no
WSL→Windows localhost forwarding). The Linux Playwright shell also
requires system libs (libnspr4 etc.) that may be missing.

## Coverage

- dashboard — stat cards and Top Paths table render seeded data
- visitors — visitor table lists seeded IPs/paths
- sessions — per-IP session cards render with visit rows
- settings — Vue app mounts (`v-cloak` removed), bot section renders
- ip-details — seeded IP summary card and path history render

## Gotchas learned

- Prefer `chromedp.WaitReady` over `chromedp.WaitVisible`: on some pages
  `WaitVisible` polls forever even though the DOM is fully loaded.
- Routing is query-param based: `?controller=<name>` plus `&ip=<ip>` for
  the ip-details page (see `shared/constants.go`).
- Pages render via Vue after an AJAX fetch — wait for a generated element
  (e.g. a `v-for` table row or the `.card` shown in the `v-else` branch)
  before asserting, not for static template markup.
- The settings page is the only one using `v-cloak`;
  `#stats-settings-app:not([v-cloak])` only matches after Vue mounts.
- Tests share one store/browser across the whole run — don't assert on
  fixed row indices and don't delete seeded fixtures.
- chromedp v0.23 rewrote the API (generic `Run[T]`); this suite pins the
  stable v0.14 line with matching `cdproto`/`go-json-experiment` versions.
