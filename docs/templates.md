# Templates and front-end architecture

The UI is **server-rendered HTML** using Go’s `html/template`. Templates live under [`internal/views/templates/`](../internal/views/templates/); locale JSON files under [`internal/views/locales/`](../internal/views/locales/).

For embedding behavior in production vs loading from disk in development, see [`internal/views/render.go`](../internal/views/render.go) and [`docs/deployment.md`](deployment.md).

## Principles

- Templates render **presentational** HTML from view data prepared in handlers; keep business logic in Go services.
- Support **full-page** responses and, where useful, **partial** responses for htmx.
- Localization: the `t` template helper reads nested keys from locale JSON (`en.json`, `ja.json`); stable domain values (enums, provider IDs) are not “translated” as product data.

## Interaction styles

The app is **not** limited to plain HTML form posts.

- **Standard HTML** — links and forms work without JavaScript.
- **htmx** — augments markup with `hx-get`, `hx-post`, etc., so the server returns **HTML fragments** for targeted swaps (faster feedback, same routes).
- **Vanilla JS** — used where the browser must own behavior (camera, barcode decoding, small page-scoped scripts). Scripts live under [`internal/static/public/js/`](../internal/static/public/js/) (served as `/static/...`).

Rules that do not change:

- The **backend** is authoritative for auth, catalog state, imports, and provider access.
- Do not introduce SPA-style global client state or move canonical rules into the browser.

## HTML-first rules

- Default navigation is **server-rendered** pages.
- **htmx** is an enhancement, not the core architecture.
- **JavaScript** is allowed and used where it materially improves UX (especially barcode scanning within Search).

## htmx (typical uses in this repo)

- Search: standard GET forms and pagination; fragment responses remain available when `HX-Request` is set.
- Catalog / wishlist: filter changes and pagination without a full reload.
- Inline notices: import feedback, media refresh feedback.

Vendored **htmx** is loaded from [`layout/app_shell.html`](../internal/views/templates/layout/app_shell.html) via `/static/js/htmx.min.js`.

## Vanilla JS (typical uses)

- **Search barcode section** — camera lifecycle, ZXing-based decoding, calling `/barcode/lookup` and rendering candidates; user must still confirm save via normal app actions.
- Optional small helpers under `internal/static/public/js/` as needed.

## Forms

- Semantic controls, localized labels, explicit submits.
- On validation errors, preserve submitted values where handlers supply them.
- Forms may POST as classic HTML, as htmx requests, or be supplemented by JS; choose the simplest approach that fits the page.

## Template layout (actual tree)

```txt
internal/views/templates/
  layout/
    base.html
    app_shell.html
    auth_shell.html
  partials/
    flash.html
    form_errors.html
    inline_notice.html
    language_switcher.html
    library_entry_list.html
    library_filters.html
    library_list_section.html
    nav.html
    pagination.html
    render_content.html
    search_result_card.html
    search_results.html
    scan.html
  pages/
    home.html
    library_detail.html
    library_form.html
    library_list.html
    login.html
    register.html
    search.html
    search_detail.html
    settings.html
```

Partials are composed by layouts and `render_content` / page templates as needed.

## Workspace navigation

My Library is the default page, with Collection and Wishlist as local views and counts. Both support cover images and List/Grid layouts, preserved through filters and pagination. Add item opens Search. Search combines title lookup with a collapsible barcode input/camera section and an optional media type selector above the title field. All media shows up to three results per type with See all links into type-filtered pagination. Search defaults to a cover grid within each group, with a List alternative; the active layout control is disabled. The layout remains selected through searches, See all, and pagination. Clicking a result opens full metadata in a native dialog, where Collection and Wishlist save actions are offered separately. Manual creation is a fallback within this flow. Settings lives in the header.

`scan.js` owns barcode candidates and camera cleanup (including closing the disclosure). `search.js` handles read-only preview loading, dialog cancellation/retry, focus restoration, and manual creation links. Preview links support full-page navigation without JavaScript. Both English and Japanese use the same page structure.

Run the UI behavior checks with `node --test scripts/search-ui.test.cjs`; Go template and redirect checks run with `go test ./...`.
