# Bookmark manager templates (design 1a/2a — "Quiet Precision")

One responsive design: desktop shows inline row actions; below the `sm` breakpoint (640px)
rows switch to a single 44px "⋯" button opening a bottom action sheet, and Add becomes a FAB.
Dark mode via Tailwind `darkMode: 'class'` + the ◐ toggle (persisted in localStorage).

## Files
- `index.html` — full page: header, search, tag filters, list, pagination, FAB. Contains one example row — extract it as your row partial and loop server-side.
- `_modal_form.html` — add/edit form (centered dialog on desktop, bottom sheet on mobile). Serve from `GET /bookmarks/new` and `GET /bookmarks/:id/edit`, swapped into `#modal-root`.
- `_delete_confirm.html` — delete confirmation dialog. Serve from `GET /bookmarks/:id/confirm-delete`.
- `_action_sheet.html` — mobile row actions. Serve from `GET /bookmarks/:id/actions`.

## Wiring notes
- Replace hardcoded ids (`bookmark-1`, `/bookmarks/1`) with your template engine's variables.
- Search input hits `GET /bookmarks/search?q=…` and swaps `#bookmark-list` innerHTML — return just the rows.
- Edit form uses `hx-put` targeting the row (`outerHTML`) so the row updates in place; create should use `hx-post` with `hx-swap="afterbegin"` on `#bookmark-list`.
- The `_="on htmx:afterRequest remove #modal"` attributes assume hyperscript; if you don't use it, close the modal with `htmx:afterRequest` listener or an `HX-Trigger` response header instead.
- Tailwind is loaded from the CDN for preview; in production use your existing Tailwind build (config: `darkMode: 'class'`, font "Instrument Sans").
