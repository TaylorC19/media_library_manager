# Browser QA — 2026-10-30

Tested the running app at `http://localhost:8080` in the Codex in-app browser, signed in as the existing administrator. This is a live browser review, with DOM checks and screenshots; it is separate from the earlier code-only improvement investigation.

## Scope and data handling

- Original viewport: 515 × 885. Detail dialogs were additionally checked at 390 × 844 and 1280 × 900, then the original viewport was restored.
- Existing data: two Collection entries and one Wishlist entry. These small counts limit Library pagination and bulk-operation coverage.
- Used existing records and read-only navigation. Library filters/layout were temporarily changed and restored to unfiltered List. Search was restored to the original All media / `avatar` / Grid / page size 10 URL.
- Typed a temporary, unsaved note in the edit form to check navigation behavior; never submitted Save. Verified the saved note remained empty afterward.
- Public MusicBrainz/Open Library connection checks were run. No personal API keys were entered, saved, cleared, or exposed.
- No entries were created, moved, deleted, or refreshed. No roles/accounts were changed. Backup export/restore and camera permission were not exercised.

## Confirmed visual defect

### BQA-01 — Detail layouts become too cramped just above the mobile breakpoint (P2, fixed)

**Reproduce**

1. Use a viewport around 515 px wide.
2. Search All media for `avatar` and open the 2009 movie's detail dialog.
3. Inspect the description and the Directors/Genres/Cast values.
4. Open the saved Avatar item in My Library and inspect the same fields.

**Actual:** The cover and text remain in two columns. A second two-column definition list inside the text column makes values extremely narrow: names and genres wrap mid-word, and dates split across multiple lines. The saved-item page's definition values measured only 82 px wide at the default viewport. The dialog is visibly more cramped.

**Expected:** Switch the cover/description layout to one column based on available component width, or raise the responsive breakpoint; allow the metadata list to stack when its text column is narrow.

**Comparison:** At 390 px, the existing mobile rule stacks the cover and text and the dialog values measured 136 px wide. At 1280 px, the desktop dialog had a 526 px text column. Neither comparison had document-level horizontal overflow. This isolates the defect to an intermediate width rather than every mobile/desktop layout.

**Evidence:** [Narrow dialog](saved-search-dialog.jpg), [narrow saved-item details](library-detail-narrow.jpg), [390 px dialog](dialog-390.jpg), [desktop dialog](dialog-desktop.jpg).

### Fix verification

The detail component now uses its own available width for responsive layout. At component widths up to 36rem, the cover and metadata stack; at widths up to 22rem, definition labels and values also stack. Desktop retains the cover/text columns.

Live verification after reloading the changed stylesheet:

- Search dialog at 515 px: one 420 px column, with metadata values 276 px wide; no document horizontal overflow.
- Search dialog at 390 px: one 280 px column, with stacked metadata labels/values; no document horizontal overflow.
- Search dialog at 1280 px: desktop columns of 180 px and 526 px; no document horizontal overflow.
- Saved-item page checked at 515 and 390 px; both use stacked cover/text and the narrower layout also stacks definition labels/values.
- Temporary viewport overrides were reset and the browser returned to Search.

Evidence: [Fixed 515 px dialog](dialog-515-fixed.jpg), [fixed 390 px dialog](dialog-390-fixed.jpg), [fixed saved-item page](library-detail-515-fixed.jpg).

## Browser-confirmed usability improvements

These are product gaps, not failures against a previously specified requirement.

| Backlog ID | Observation | Suggested improvement |
|---|---|---|
| I-01 | Avatar's result card says In Collection, but its loaded dialog still offers Add to collection and Add to wishlist, with no Open saved item action. | Make dialog actions aware of existing saves and support saving without leaving the results. Duplicate submission was not tested. |
| I-04 | Editing the saved movie presents formats including Hardcover, Paperback, Nintendo Switch, PlayStation 5, and Xbox. | Offer media-appropriate formats, while retaining any existing legacy value. Allow optional copy details when saving from Search. No invalid-format save was submitted. |
| I-10 | Typing a note and clicking Back to detail navigates away immediately, without an unsaved-changes warning. The original note remains empty. | Warn before discarding a changed edit form; optionally preserve a local draft. |
| I-11 | At the default narrow width, the header, search controls, full-width Clear link, barcode disclosure, and manual-create link place the first result group below the initial viewport. | Compact the search controls and add media-type jump links/collapsible groups to reduce scrolling. This is a usability observation, not a rendering failure. |

The earlier delete-confirmation recommendation remains high priority, but Delete was not clicked during this browser pass. Its destructive behavior is not recorded as an end-to-end browser test.

## Checks that passed

- All media `avatar` search reached five non-empty groups: Movies, TV, Music, Books, and Games; three preview cards per group and working See all links. Loading placeholders were visible before completion.
- Collection and Wishlist badges matched the existing saved movie examples.
- Movie search pages 1, 2, 5, 7, and 9 loaded different results. Page 9 showed `81–90 / 90`, confirming access beyond the initial provider batch.
- Pagination kept Previous on the left, page controls centered, and Next on the right. Unavailable controls and current page were disabled. No document horizontal overflow was measured on pages 2 or 9.
- Search details loaded full metadata. Close worked; Escape closed the dialog and returned keyboard focus to its originating result card.
- My Library filtering through HTMX reduced two Collection entries to one matching `avatar` entry.
- Grid plus the `avatar` Library filter survived navigation to Search and back. Japanese switching retained the query and layout and translated the controls. Clear restored the unfiltered list; List was restored afterward.
- Saved-item pages displayed cover, description, creators/media details, copy fields, and a top back link.
- Wishlist details displayed Move to Collection and its preservation hint. The action was not submitted.
- Admin General-role filtering showed seven General accounts, excluding the administrator and demo account. Clear restored all nine accounts.
- The signed-in administrator's own role selector and Save action were disabled.
- Saved-key management displayed status only and correctly omitted clear actions for an account with no personal keys. The confirmation flow could not be exercised without adding a key.
- Settings displayed provider purposes, acquisition/website links, and configuration status. MusicBrainz and Open Library connection tests both returned “Connection successful. The service returned a valid response.”

## Remaining coverage

Not exercised end to end: saving/importing a result, duplicate-save handling, Wishlist moves and conflicts, edit persistence, deleting/restoring items, key replacement/clearing, account creation/role changes, other-role sessions, backup export/restore, camera scanning, network failures/retries, or Library pagination with a large collection. Provider totals and results are a point-in-time observation and can change.

## Recommended next order

1. BQA-01's intermediate-width detail layout is fixed and browser-verified.
2. I-01: duplicate-aware dialog actions and save-and-continue browsing.
3. I-02 and I-10: deletion confirmation and unsaved-edit protection.
4. I-03 and I-04: sorting/tag filters and media-appropriate copy details.
