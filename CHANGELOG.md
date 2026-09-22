# Changelog

## v0.11.0 — 2026-09-20

> **Wavelog API v2.** CQOps now speaks only the Wavelog API v2: Bearer tokens instead of v1 API keys, granular scopes, typed lookups, and a paginated ADIF sync. Legacy v1 keys are no longer accepted — create a v2 token (`wl2_…`) in Wavelog and paste it into the logbook form.

### Wavelog API v2
- **Bearer authentication**: the token travels in the `Authorization` header only — no more API key in URLs or request bodies.
- **Connection test**: uses the public `/api/v2/status` endpoint plus `/api/v2/token` (whoami) to verify the token and its scopes.
- **Station profiles** come from `GET /api/v2/station`.
- **Callsign lookup** uses `GET /api/v2/lookup?detail=full` with typed booleans — the legacy truthy-string guessing is gone.
- **QSO upload** posts ADIF via `POST /api/v2/qso` (`import_type: adif`); duplicates are detected from the `parsed/imported/skipped` summary instead of message string matching.
- **Download/sync** uses `GET /api/v2/qso?format=adif&since_id=…` with pagination (`has_more`, 250 rows per page): pages stream straight into the temp file instead of being buffered in memory, later page headers are stripped so the result stays one ADIF document, the download bar advances per page against `meta.total`, and a safety guard aborts if the server stops advancing `lastfetchedid`.
- **Remote QSO ids**: every downloaded QSO stores its Wavelog primary key (`wavelog_id`, captured from the JSON list sidecar aligned 1:1 with the ADIF rows) — the foundation for future edit/delete sync via `PATCH`/`DELETE /api/v2/qso/{id}`.
- **Upload round trip**: single-QSO uploads (manual save, WSJT-X auto-log, editor individual uploads) now use the v2 single-JSON create, whose response carries the created QSO id — it is stored locally immediately. If the JSON create is rejected, CQOps automatically falls back to the ADIF import path. Batch editor uploads and duplicate results look the id up on Wavelog right after the upload, so the local log converges to `wavelog_id > 0` without waiting for a download.
- **`wavelog_id` is the single source of truth for uploaded state**: the legacy `wavelog_uploaded` flag column is dropped during migration (existing databases upgrade automatically). A QSO counts as uploaded exactly when its `wavelog_id` is set; the logbook editor's upload filter, the recent-QSO `WL` column and all statistics now derive their answer from the remote id.
- **Migration reconciliation**: logbooks migrated from older versions have no remote ids, so a large "send to Wavelog" first reconciles against the remote QSO list (paginated, 5000 rows per page) and assigns ids locally instead of re-uploading — only genuinely new QSOs are uploaded. Small batches still rely on the server's duplicate detection, and a duplicate upload learns the remote id from the server and stores it locally.
- **Edit sync**: opening a QSO that exists in Wavelog refreshes the edit form with the server's current copy (GET `/api/v2/qso/{id}`), and saving it pushes the edit back via `PATCH /api/v2/qso/{id}` — the local save always succeeds first and never depends on Wavelog. If the remote copy was deleted in the meantime, the local remote id is cleared honestly. Deleting a synced QSO also removes the Wavelog copy (`DELETE /api/v2/qso/{id}`), and both confirmation dialogs state the Wavelog effect (or that the operation is local-only while offline).
- **Live radio state**: after the connection check, CQOps ensures a radio named `CQOps` exists on the Wavelog radio resource (creating it on first use) and pushes the QSO form's frequency, RX frequency, mode and power to it every 15 seconds. If the radio no longer exists (e.g. deleted in Wavelog), it is recreated and the push retried.
- **Wavelog station sync**: when a Wavelog station profile is selected, the logbook station is kept in sync — on save (wizard and logbook config menu) the profile's grid, callsign, DXCC entity, CQ/ITU zones and SOTA/POTA/WWFF/SIG reference fields are mirrored locally (`GET /api/v2/station/{id}`, short timeout, best-effort). Selecting or cycling a station (Space over the Station ID field) fills the whole form — callsign and grid immediately, and the remaining fields as soon as the profile arrives — and stale results from a previous selection never overwrite the current one.
- **Simpler wizard**: the optional SOTA/POTA/WWFF, CQ/ITU zone, DXCC and SIG fields are hidden in the first-run wizard by default — Ctrl+A reveals them. The IARU Region selector is hidden too (the wizard shows the Continent selector only); both remain available in the config menu.
- **Save & Next button**: every wizard step ends with a visible `[ Save & Next ]` / `[ Save & Start ]` button with a `(Space)` hint — Space or Enter activates it, and Tab hands focus to it from the form's last field (Shift+Tab goes back into the form). Up / Shift+Tab from the first field now reaches the button too, so upward navigation never skips it. The step indicator and summary point to Space instead of relying on a hidden Enter convention.
- **Wizard banner**: the setup wizard opens with a big ASCII-art CQOps logo and a single line combining the version with the GitHub link. Small terminals automatically get a compact banner so the Save button always stays visible.
- **Three-step wizard**: the timezone step is gone — CQOps uses the timezone Go detects from the system and shows it on the summary screen. First-run setup is now Station & Logbook → Rig → Summary.
- **Error handling** maps the v2 error envelope (`invalid_token`, `token_expired`, `insufficient_scope`, `rate_limited`, `conflict`, …) to actionable messages.

### Configuration
- The logbook form shows a green **v2** badge next to the API key when a `wl2_` token is entered, and a warning — *"Wavelog API v2 token (wl2_) required since CQOps 0.11.0"* — when a legacy v1 key is present.
- **Migration surfacing**: users still on a legacy v1 key get the migration message as a one-time warning toast at startup, the status bar shows a warning `WL!` indicator instead of plain offline, and uploads with a v1 key fail fast with the same guidance (no silent ADIF fallback).
- **Shift+Backspace** in the QSO form instantly clears the focused field (listed in the `?` help overlay); the call field's side effects (partner data, name/QTH/grid clearing) apply as with manual editing.
- **APRS convenience**: turning APRS TX on in the logbook station form now also enables the global APRS integration — no separate visit to the Integrations menu needed. Turning TX off never disables the global integration.
- **DX Cluster login prefill**: the Integrations menu pre-fills the DX Cluster login with the first available station callsign from your logbooks (deterministic order) when no login is configured — editable as usual.
- **Consistent toasts**: every toast now starts with its module name (`Wavelog:`, `DXC:`, `QRZ:`, `Rig:`, `QSO:`, …), including dynamic messages from menus and the editor — at a glance you always know which integration or screen produced the message.
- **Save & Back button**: every config form and edit/create screen — logbooks, rigs, operators, contests, the QSO editor, and the General / Integrations / Callbook / Notifications menus — now ends with a visible `[ Save & Back ]` button with a `(Space)` hint. Tab reaches it from the last field, Up from the first, and Space or Enter activates it, so nobody gets lost in a form anymore.
- **Unified menu engine**: all config menus and forms (General, Notifications, Integrations, Callbook, Contest, Rig, Operator, the QSO editor, the logbook station form, and the setup wizard) now share one focus/navigation engine and shared row renderers — a single implementation of Tab/Shift+Tab navigation, the Save & Back button, focus markers, hidden-row skipping, and viewport scrolling, guarded by one invariant test matrix that runs against every menu. This also fixes a whole class of focus bugs: double active markers, Save button focus leaking between forms, stray text cursors, and unreachable fields (e.g. Prefill Exchange Rcvd or APRS TX depending on neighboring toggles).

### Fixes
- Re-running a Wavelog download when the local log is already up to date no longer shows a confusing "Downloaded 0 QSOs." dialog — it now says "Wavelog is up to date — no new contacts to download." (aborted downloads still report their real count).
- PSK Reporter only toasts an update when the fetch actually returned spots; an empty fetch stays silent.
- Background Wavelog uploads and WSJT-X enrichment now capture an immutable operation context (database, logbook identity, destination credentials) at command creation. Switching logbooks mid-operation can no longer enrich or upload a contact from the wrong logbook — the captured database is kept alive until the operation finishes, and result messages carry the logbook identity so stale results are dropped.
- Fixed a permanent shutdown deadlock in the WSJT-X listener: `Stop` held the listener mutex while waiting for the event loop, while the loop needed the same mutex for its callback snapshots. Shutdown now detaches under the mutex and closes the socket and joins the UDP reader and event loop outside it.
- APRS lifecycle is now single-owner: the debounce timer and APRS workers hand events back to the TUI main loop instead of running lifecycle transitions or touching live config themselves. Workers (beacon, pruner) receive immutable snapshots and are joined before their resources are replaced, and `config.Save` marshals a scrubbed copy instead of temporarily clearing credentials on the live config — eliminating concurrent map access and mismatched beacon state across logbook switches and shutdown.
- The HTTPS dashboard no longer dies or stalls from a single TCP connection: connection classification now runs under a deadline, and a peer's EOF/timeout is treated as an ordinary per-connection failure instead of a fatal listener error that terminated the whole server.
- Dashboard map popups and tooltips now escape every log field (callsign, band, mode, grid, country, APRS callsign/comments), so stored HTML from imported contacts can no longer execute. The theme bootstrap moved to an external script and inline event handlers were replaced with JS listeners, allowing the CSP to drop `'unsafe-inline'` from `script-src`.
- Wavelog download checkpoints now advance only through durably processed contacts: the first record whose insert fails freezes the `last_fetched_id` cursor so the next incremental download retries it instead of skipping past it (with a "deferred — will retry" note in the result dialog). ADIF import/export completions no longer touch the Wavelog cursor at all.
- Opening a contact in the logbook editor now populates every field on every load, explicitly clearing absent numeric values — editing a contact after one with frequency/distance/bearing/serial data can no longer persist the previous contact's values.
- Upload normalization now rewrites only the fields flagged as mismatching in the confirmation: accepting an operator or locator fix no longer erases the original station callsign, and callsign mismatches are detected (using the logbook station callsign) and normalized only when confirmed.
- Secret-store failures now abort configuration saves instead of silently dropping credentials: if `secrets.enc` cannot be persisted, `config.yaml` is not replaced. Clearing a credential field now also deletes the stored secret, so removed passwords/API keys no longer return after a restart.
- A failed logbook switch no longer leaves the current database closed: the replacement database is opened and validated first, and the previous database is retired only after the transition commits.
- Import/download/export cancellation can no longer be lost: operations now run on an immutable operation object (channel + cancellable context) captured at launch. Aborting cancels in-flight Wavelog requests via context, progress sends never block, and the worker always delivers its final result.
- DX Cluster reconnection now has one owner: after the first successful connection the client reconnects itself, and the TUI no longer drops the client pointer on disconnect (which left abandoned clients reconnecting and created duplicate cluster sessions). Client state is per-connection-generation and synchronized, and `Stop` joins the client's goroutines before it is replaced.
- Editing a contact now keeps its derived database metadata consistent: `base_call` is always recomputed from the callsign, and when a contact's callsign changes the stale prefix-derived DXCC is invalidated (cleared) so the DXCC backfill recomputes it — worked-call searches and statistics no longer disagree with the displayed contact.
- Saving an existing synced contact in `--offline` mode no longer contacts Wavelog: the edit is kept locally (remote id preserved) and reported as "Wavelog sync deferred (offline)".
- Clearing a field while editing a synced contact now clears the Wavelog copy too: the PATCH separates field presence from value and sends explicit `null` for emptied fields (comment, locator, references, RX frequency, TX power) instead of silently leaving the remote value behind for a later refresh to restore.
- Batch uploads no longer report partial success as full success: the result now carries explicit sent / already-on-Wavelog / failed / unresolved tallies. A failed chunk keeps its contacts unsent locally and shows up as "N failed" instead of inflating the sent count, and contacts the server accepted but whose remote id could not be stored locally are reported separately instead of passing silently.
- ADIF import/export failures can no longer end on success-looking result screens: every operation now ends with a single terminal result carrying its counts and error. A truncated or corrupt import reports the scanner error together with the partial counts, a failed export reports its error, and exports are written to a temporary file that is only promoted to the target path after a successful write and close — an aborted or failed export never leaves a partial file behind.
- Dashboard shutdown now terminates SSE clients: the server owns a stream context that ends every active event-stream handler when stopping, so changing dashboard settings with a browser connected no longer blocks the TUI for the shutdown timeout. SSE writes carry a bounded per-write deadline (a stalled client can no longer strand a handler), write errors end the stream, and a failed graceful shutdown force-closes the listener and connections instead of forgetting the server. The dashboard server is also stopped on application exit.
- Rotor (rotctld) responses are now framed correctly: a persistent buffered reader consumes whole lines up to the `RPRT` terminator instead of assuming one TCP read returns the complete reply. Responses split across reads are reassembled, leftover bytes can no longer contaminate the next command's reply, an acknowledgement cut short is reported as an error, and multi-line payloads (like the rotor name) no longer include the `RPRT 0` terminator line.
- Reference-data downloads are now bounded and asynchronous: the Big CTY catalog and archive requests use a context-aware HTTP client with a timeout (a stalling server fails the request instead of hanging the whole sequential refresh), the expanded `cty.csv` is capped so an unexpectedly large archive cannot exhaust memory, and refreshed databases are returned as messages and installed on the main loop — the background worker never assigns application pointers itself.
- Upload preparation no longer freezes the TUI on large logs: unsent contacts are counted with a single SQL COUNT and only the eligible (never-uploaded) rows are fetched, preparation runs in the background as a message-producing worker, and full-log listing uses keyset pagination instead of increasingly expensive OFFSET pages.
- Logs no longer expose precise operator positions: log files and the log directory are owner-only (0600/0700, existing files tightened on rotation), INFO logs carry only the GPS grid truncated to the configured grid precision, and six-decimal coordinates are logged exclusively behind the debug-mode diagnostics opt-in.
- The wizard's final step returned the `tea.Quit` function instead of a quit message, so completing the summary never actually exited the wizard — it now sends a proper quit message and launches CQOps.
- The QSO form's DX Cluster line no longer silently drops the continent filter when no same-continent spots are near the frequency — spots from other continents cannot leak into the form (or into the Ctrl+P spot cycling). The DXC pane's explicit continent filter now also overrides the station continent for that line.
- Recent QSOs now appear immediately after a Wavelog download or ADIF import: the deferred QSO-refresh command was silently discarded by the update loop (the flag was consumed but the refresh never ran), leaving the QSO pane empty until restart.
- The HTTP dashboard's recent/today/stats panels now refresh after downloads, imports, editor saves, deletes and purges — previously their change-detection cache was only invalidated on QSO-form saves, so the dashboard could stay stale until restart.
- Leaving the callsign field no longer causes a one-frame layout shift: while the local log statistics and callbook lookups are still pending there is nothing to show (no badges, no grids), and the info row above the QSO form briefly collapsed, pulling the form border up one row until the lookups landed. The row now keeps its fixed height (rendering blank) from the moment a callsign is entered, so badges appear in place without moving anything.
- A config file that fails to parse (e.g. a YAML syntax error) or cannot be read is no longer silently replaced with defaults: startup now reports the error and leaves your `config.yaml` untouched. Defaults are written only on a genuine first run, when no config file exists.
- Retrying a failed secrets save can no longer silently lose credentials: the encrypted store now tracks unpersisted changes, so a retry after a transient write failure persists the pending passwords/API keys (or deletions) before writing the scrubbed YAML — previously the retry skipped the store write and saved the config with the credentials stripped.
- A Wavelog download can no longer assign another contact's remote ID: remote IDs are now matched to downloaded contacts by verified identity (call/band/mode/date/time) instead of by list position. When an ID page fails or its row count does not match the ADIF page, its contacts are imported without remote IDs (and the download cursor freezes on them) — previously the missing page shifted later IDs onto earlier contacts, so editing or deleting could have targeted the wrong remote QSO.
- Rotor (rotctld) polling works again with real rotctld responses: every command is now sent with the `+` prefix, which requests the documented Extended Response Protocol — replies are terminated by `RPRT`, and azimuth/elevation are parsed from the `Azimuth:`/`Elevation:` records (the model name from `Info:`). Previously the client waited for an `RPRT` terminator on default-protocol replies, which never send one for `p` — so a healthy rotor's position reply timed out and dropped the connection.
- WSJT-X listener shutdown can no longer deadlock with backed-up traffic: stopping the listener now keeps draining the reader's message/error channels until the UDP reader exits. `ListenToWsjtx` uses blocking channel sends, and closing the socket cannot unblock a reader already waiting to send into a full channel — with the event loop gone, nothing drained the queue, so quitting or restarting could hang while traffic was queued.
- Remote refreshes can no longer overwrite your edits: opening a synced contact still fetches the server's copy, but a late result is discarded when you have typed since the fetch began, and the editor now durably records pending local changes (offline saves and failed PATCHes mark the row) — such rows are never overwritten by a refresh, opening one edits the local copy with a notice, and a successful sync clears the marker. Previously a pending GET could silently replace your typed comment (or a previously saved offline edit) with the old server copy.
- Async upload preparation can no longer cross logbook boundaries: every editor instance now carries a unique generation, and the preparation result (plus the follow-up normalize step) is bound to the originating editor's generation and database. A result that arrives after a logbook switch is discarded — previously the new logbook's editor would have uploaded the old logbook's contacts using its own Wavelog configuration and backfilled IDs against its own rows.
- SQLite configuration now actually takes effect: the database DSNs used mattn-style parameter names (`_journal_mode`, `_busy_timeout`, `_foreign_keys`) that the `modernc.org/sqlite` driver silently ignores — the main logbook and the reference database really ran with `journal_mode=delete`, no busy timeout, and foreign keys off. Both now use the driver's `_pragma` parameters, applied to every pooled connection (WAL, 5s busy timeout, foreign keys; the reference DB additionally keeps its synchronous/cache tuning), so background imports, WSJT-X logging and UI reads resolve locks by waiting instead of failing immediately.
- Background workers no longer touch owner-thread state: the callsign filter query now returns a typed result that is applied to the Recent-QSOs table only in `Update` (stale results are dropped), the post-save Wavelog station sync only fetches in its worker and applies config/save/toasts on the main loop, and WSJT-X enrichment consumes snapshots of the callbook registry, config flags and database — its dashboard push moved into the result handlers, which also enforce the same-logbook check.
- DX Cluster reconnects now restore the UI: connection-status events are drained whenever a client exists, not only while the UI believes it is online — after a drop the client reconnects on its own, but the reconnect event was never consumed (`ConnectedOnce` short-circuited the tick), leaving CQOps stuck offline and spot draining stopped until a manual reconnect.
- Late query results can no longer replace the current logbook's UI data: QSO-refresh and worked-statistics messages now carry the logbook identity they were queried against, and results from a previous logbook are discarded. Previously a slow refresh from logbook A arriving after a switch to B replaced B's QSO table, and a delayed statistics result could show worked/new-call badges computed from the wrong logbook.
- DXC caches now invalidate correctly: the path-line duplicate cache key includes the logbook identity and a dupe revision that is bumped on every QSO mutation (the key previously only carried date + contest, so logging a contact left stale markers and switching logbooks with the same contest reused the other logbook's results), and the DB spot fallback now expires after two minutes instead of serving time-windowed spots indefinitely.
- GPS fix loss and movement are now propagated correctly: a void GGA or RMC sentence explicitly invalidates the previous fix (losing reception can no longer advertise a stale valid position), the UI expires any fix whose last update is older than 15 seconds, every position change is pushed to the app (APRS beacons follow movement instead of freezing at the first acquisition), movement updates the station-grid override in place, and losing the fix restores the configured fallback grid.
- ADIF export now streams through a single read-only snapshot transaction with a keyset cursor: a QSO logged while the export runs (e.g. WSJT-X) can no longer shift the `LIMIT/OFFSET` pages — previously one record could be duplicated and another silently omitted while the export still reported success.
- HTTPS classification no longer serializes client stalls: TLS/HTTP classification of dashboard connections now runs concurrently behind a bounded slot pool, so idle connections burn only their own first-byte deadline — ten stalled connections ahead of a browser no longer delay it by ~30 seconds, and a continuous stream of idle connections cannot starve the dashboard.
- The single-instance lock is now an OS-backed exclusive lock (`flock` on Unix, `LockFileEx` on Windows) taken before any configuration is read or written and held for the process lifetime. The kernel releases it automatically even after a crash, so no stale-PID cleanup is needed, and racing instances are serialized by the OS instead of a read-then-write PID check. The lock is explicitly released when initialization fails.
- The local callbook provider no longer keeps querying a retired database: the provider registry is rebuilt on every logbook switch (cycle and chooser paths), and in-flight callbook/Wavelog lookup results carry the logbook they ran against so a late result from the previous logbook is discarded instead of filling the form with the old logbook's history. A lookup for the current callsign is re-triggered against the new logbook after the switch.
- A late Wavelog remote refresh can no longer overwrite a different logbook's contact: the fetch now carries the editor generation, database, local QSO id and edit-session revision, and all of them are validated before the database or form is touched. A refresh started in logbook A that lands after switching to logbook B — even with a matching remote id — is discarded, and a response is only applied to the exact row it was issued for.
- Saving a synced contact now marks it durably pending before the Wavelog PATCH starts: the edit, the dirty flag and a bumped pending-sync revision are committed atomically, so killing the app while a PATCH is in flight (or saving without credentials configured) leaves a pending row that a later remote refresh cannot overwrite. The flag is cleared only when the server acknowledges the exact revision it was saved under — an older successful request can no longer clear a newer pending edit.
- Creating a logbook no longer changes the active identity before the new database opens: `SwitchLogbook` now exclusively commits the active logbook, station, and database together. When database creation fails (permissions or initialization errors), the provisional logbook entry is rolled back and the previous logbook stays fully active with its own database — previously the new station identity stayed active while writes kept going to the old database.
- The snapshot-based ADIF export no longer silently omits DXCC: the shared snapshot projection and scanner now include the `dxcc` column, so a contact with a stored DXCC value exports its ADIF DXCC field and reimporting the backup keeps it. A new export/import fidelity test round-trips every supported persisted ADIF field, which count- and ordering-only checks cannot catch.
- Dashboard shutdown no longer leaks accepted connections: the hybrid TLS/HTTP listener now implements explicit shutdown — it cancels slot acquisition and result delivery, closes connections that were accepted but never dispatched (and connections accepted after shutdown began), joins the accept loop and classification workers, and only then closes the result channel. Previously a client waiting to send its first byte kept an open socket after the server exited (a read timeout instead of a close), and workers blocked forever when the result queue was saturated with no consumer.
- DXC connection setup no longer races the TUI owner loop: the connect command captures the connection settings, the existing client, and a connection generation before launching, and returns the constructed client in a generation-tagged result that the owner loop installs. Results from a generation that was invalidated by a reset, disable, or offline transition are discarded and their client stopped — previously the worker read mutable model state and published `m.dxc.client` directly, which a focused race probe flagged.
- GPS grid overrides no longer cross logbook boundaries: the station grid override used to mutate the configured `Station.Grid` and remember the saved fallback implicitly for whichever logbook enabled it first — after switching to another logbook, losing the fix wrote the first logbook's grid into the second one's station state. The effective grid is now derived purely from live GPS state (the configured grid is never modified), so every logbook keeps its own configured fallback and a fix loss simply falls back to it.
- Wavelog downloads no longer advance past contacts whose remote identities could not be resolved: when the identity sidecar request fails (e.g. HTTP 500), the imported contacts stay at `WavelogID=0` and the download cursor now freezes at the last verified identity instead of jumping to `LastFetchedID` — the previous behaviour let later incremental downloads skip that range, leaving the contacts eligible for duplicate uploads with no way to update their remote copies. The affected contacts are reported as deferred, and a subsequent download with a healthy identity request reconciles them in place.
- Wavelog station synchronization is no longer lost when the logbook chooser closes first: the station-sync completion and the logbook-switch bookkeeping are now handled globally on the owner loop, independent of the visible screen. Sync results carry a save generation and are discarded when a newer save superseded them, so old station data can never overwrite newer configuration; the switch bookkeeping (cache resets, callbook rebuild, QSO refresh) runs as a separate step whether or not network synchronization is involved.
- The rendered DXC path line no longer bypasses spot expiry: the cached line itself now expires after the spot TTL and is re-rendered, so aged-out spots disappear and an expired DB fallback triggers a re-fetch even when every other input is unchanged. Newly arrived fallback spot data also invalidates the rendered line immediately. Previously a cached line rendered from in-memory spots stayed displayed forever because the cache-return path never consulted the time-dependent filters.
- Overlapping saves of the same synced contact can no longer leave the remote copy outdated: remote PATCHes are now serialized per contact — only one PATCH may be in flight per QSO, and a save arriving while one is pending is queued and coalesced to the newest revision, which is pushed by a follow-up when the in-flight PATCH completes. The pending-sync flag survives until that newest revision is acknowledged. Previously a stalled older PATCH could complete after a newer one and overwrite the server with stale values while the local row looked synced.
- Reopening the same contact can no longer defeat stale-refresh protection: each contact open now starts a new edit session with its own monotonically increasing generation, which is bound into the remote-refresh request. A refresh from a previous session for the same contact is rejected even when editor generation, database, row, and revision all coincide — previously `editRev` reset to zero on every open, so two sessions shared the same identity and the older response could overwrite the newer one.
- Save, delete, and upload completions no longer affect unrelated editing sessions: every completion now carries the editor generation and (for saves) the edit session that initiated it, and the editor only returns to the list view when the completion belongs to the unchanged session that started the operation. A delayed completion from another logbook's editor, a superseded session, or a serialized follow-up PATCH can no longer close the form currently being edited.
- Editor operations now retain their database across logbook switches: every background editor worker (save PATCHes and their serialized follow-ups, single/batch uploads, normalization, deletes, purges, downloads, imports, exports) acquires a database lease when it is dispatched and releases it only after its local writes finish. Switching logbooks retires — but no longer closes — the old database while a worker still holds it, so acknowledgement persistence can no longer fail against a closed database. A remote PATCH whose local acknowledgement still could not be persisted is reported as an incomplete synchronization (the row stays durably pending for a retry) instead of full success.
- Upload normalization no longer mutates the contact list from its worker goroutine: the normalize worker is limited to database work and returns the changed fields and affected IDs as immutable result data, which the owner loop applies to the in-memory list after validating the editor generation. Previously the worker rewrote `Operator`/`MyGridSquare`/station callsign directly on the list the owner loop reads during table rebuilds (a focused race probe reported conflicting accesses).
- An invalid downloaded record can no longer advance the download cursor past an unresolved identity: the invalid-record branch now obeys the same checkpoint-eligibility rule as every other branch (no failed insert, no identity gap), so an earlier page's failed identity request keeps the cursor behind the gap even when a later invalid contact carries a known remote id (probe: `checkpoint=11, unresolved=1, failed=1` — the next incremental download would have skipped the unresolved contact). Persisting a recovered remote id for a duplicate row now also freezes advancement on failure, so the recovery is retried by the next download instead of being skipped.
- Delayed station synchronization no longer clears the current contact's exchange fields: the station-sync completion used to emit the logbook-switch message, which ran full switch bookkeeping (clearing sent/received exchanges) even when no switch had occurred — saving logbook settings and entering an exchange before the sync completed lost the entry. Logbook-switch bookkeeping now runs immediately and exactly once at each successful switch (cycling, chooser Enter, logbook creation), while the station-sync completion follows a separate path that only refreshes station-derived state (status line, partner map and path-line caches, APRS beacon restart, dashboard push) without touching the in-progress contact.
- A queued PATCH can no longer send another logbook's contact to the wrong server: per-contact remote-save serialization is now bound to an immutable originating context (database, Wavelog endpoint, editor generation) captured when the first PATCH was dispatched, plus a database lease that covers the whole queued chain — the in-flight PATCH, follow-ups, and the interval between workers. After a logbook switch the follow-up reads and writes only the originating database and PATCHes only the originating endpoint; previously it read `App.DB` (the newly active logbook's row with the same local id) and sent it to the old logbook's server — contact corruption and unintended data disclosure. Abandoned chains (replaced editor) release their lease and leave the row durably pending for later reconciliation.
- A save completion can no longer close a form containing newer unsaved edits: `doSave` now captures the form revision when the save is dispatched, and the completion closes the edit form only when both the edit session and the revision still match. Saving a contact and continuing to type (while the PATCH runs) keeps the form open with the newer input preserved instead of returning to the list and abandoning it.
- Reopening the editor no longer bypasses PATCH serialization: the per-contact coordination state (in-flight PATCH, queued revision) now lives on the model, keyed by persistent logbook/contact identity and remote endpoint — not on the disposable editor instance. Pressing F8 while a PATCH is still on the wire recreates the editor, but the next save of the same contact is queued and pushed by the follow-up after the in-flight PATCH completes; previously the new editor's empty maps let the second request run independently and the delayed first request overwrote it on the server (`remote="first"`, `local="second"`, `dirty=false`).
- Pending lookups can no longer consume PATCH completion messages: the per-contact serialization completion now runs before the deferred-pending-request block, which could early-return after dispatching an unrelated pending lookup (DXC/QRZ/Wavelog) and swallow the incoming completion — leaving the serialization slot occupied so every later save kept queueing with no worker to drain it. The incoming command is accumulated through the pending-request path, so the dispatched follow-up survives the early return.
- Generation checks now also protect model-level completion side effects: completions (save/delete/purge/upload/download) whose operation identity belongs to a replaced editor no longer apply toasts, refreshes, editor mutations, or cursor changes to the VISIBLE logbook. Logbook-scoped persistence is handled globally and independently of the visible editor: the purge cursor reset and the Wavelog download cursor are written against the ORIGINATING logbook carried by each completion — purging logbook A and switching to B before the result arrives resets A's download cursor and leaves B's untouched (previously the foreign result reset B's cursor from 99 to 0).
- Pending lookups can no longer leave downloads stuck: dispatching an unrelated pending lookup no longer consumes the incoming editor message — download/import/export progress and terminal messages fall through to the message pump and the editor screen even when a DXC/QRZ/Wavelog lookup is dispatched on the same update. Previously the early return swallowed the message, the channel-read was never re-scheduled, `dlActive` stayed true (navigation blocked), and the worker could block on its terminal send while holding the database lease.
- Late delete and upload completions no longer discard another contact's unsaved form: delete, upload, and purge completions are now bound to the edit session they were initiated in, like saves. Deleting contact A, opening B and typing while A's remote deletion is pending keeps B's form open when A's completion arrives — the list is refreshed behind the open form instead of switching the editor into list mode. An editor-generation match alone never meant the operator was still performing the original action.
- CQ and ITU zone edits now synchronize to Wavelog: `buildUpdateInput` omitted both zones, so a successful PATCH cleared the dirty flag while the remote refresh afterwards restored the server's old zone values (changing CQ from 15 to 16 saved “successfully”, then reopening the contact showed 15 again). The PATCH now sends both zones (cleared via null, like the other clearable fields), and the synchronization contract is enforced by round-trip tests: every editable synchronized field is edited, saved (PATCH applied to a stateful mock), re-fetched, and refreshed — the new value must survive.
- Session-zero deletes no longer bypass form protection: zero is the legitimate initial edit session of a fresh editor, but the session checks treated zero as "unbound", so deleting a contact directly from the list and then opening another contact let the completion close the newly opened form. Session binding is now explicit (`opSessionSet`): real dispatches validate their captured session even when it is zero, while genuinely unbound legacy messages keep their old behavior.
- Pending lookups can no longer discard upload-preparation results: the pending-request early return used to consume the incoming message except for a growing list of exceptions (`editorMsg` was exempted, `uploadPrepMsg` was not) — a DXC lookup pending at the moment upload preparation finished swallowed the result, so the upload never started and the normalization prompt never appeared. The early return is gone entirely: pending lookups accumulate their commands, and every incoming message — operation results and ordinary input alike — continues to its handlers.
- Remote clears now apply locally: the remote refresh applied CQ/ITU zones only when positive and frequencies only when nonempty, so clearing them in Wavelog left the old local values (probe: 15, 28, 14.2 MHz) — a later unrelated save sent the stale values back and undid the clear. `QSOData` now records which JSON fields were explicitly present in the response (null included) and the refresh applies explicit clears while leaving absent fields untouched; literal (non-JSON) values keep the value-based behavior.
- Editing during an initial upload no longer leaves newer changes falsely marked synced: the upload sends a captured snapshot, and while the remote id is still zero local saves bypassed dirty tracking — the completion attached the id without checking whether the contact changed, leaving the server with the old content and the local row with a remote id and `dirty=false` (a later refresh overwrote the newer edit). Every local write now bumps the pending-sync revision, the upload completion attaches the id while checking the uploaded revision, and when the row changed it stays durably dirty and a follow-up PATCH of the latest revision is queued through the per-contact serialization coordinator.
- Failed remote-ID persistence is no longer reported as successful synchronization: when Wavelog accepts a contact but the local database rejects the id write, the completion now reports an unresolved result ("accepted but remote id not stored — will retry") instead of success with a stale id, the row stays locally unsent, and a one-shot id-attach retry is queued (a failed retry leaves the row re-offered on the next upload cycle). Remote acceptance and local persistence are separate outcomes throughout the upload paths.
- Upload data and revision can no longer describe different snapshots: the individual upload sent the data captured at preparation time but re-read the revision from the database immediately before sending, so a newer local edit made the old contents get attached with the new revision and the row was falsely marked clean. Unsent lists now capture each row's pending-sync revision together with its data, the individual upload uses that captured pair, the batch reconciliation and chunk id backfill attach ids with the revision check instead of the unchecked write, and the WSJT-X path no longer disables the check with -1. A row changed between preparation and attachment stays durably dirty and its id is queued for a follow-up PATCH of the latest revision.
- ID-attach retries no longer forget which revision actually reached the server: the retry read the latest local row and attached the id with its CURRENT revision, so an edit made after the failed id write left the row clean although the server held the older contents — and the id lookup used the edited identity, which could link the wrong remote contact. Upload completions now carry the accepted snapshot (identity + revision) as a pair, the retry looks the id up with the snapshot's identity and attaches it with the snapshot's revision, and the retried result carries the originating logbook so the existing logbook-scoping check applies.
- The database lease now spans the complete upload → ID retry → reconciliation chain: the upload worker used to release its lease before returning its result and the completion handler acquired a new one, so switching logbooks while an upload was pending closed the retired database at the worker's release and the follow-up received a closed handle ("sql: database is closed"). The worker's lease now transfers through the completion message to the follow-up chain (reconciliation PATCH or id-attach retry), transfers onward with the retried result, and is released only when the chain drains — the retired database stays open for the whole chain and closes at its end.
- Editor upload reconciliation no longer depends on the visible screen: the follow-up PATCH and id-attach retry were dispatched only by the logbook-editor screen handler (after its editor-generation check), so leaving the editor or recreating it before the completion dropped the required PATCH — the row stayed dirty and the newer revision was never sent. Upload completions now carry the originating logbook identity and their follow-up chains are queued globally, independent of the visible screen; screen and generation checks apply only to UI effects.
- Editor uploads no longer release the database lease before reconciliation: the single-upload and batch workers transferred their lease into the completion message (with the originating logbook), and the global completion handler hands it to the follow-up chain or releases it when no follow-up is needed — a logbook switch during an editor upload can no longer close the originating database before its PATCH runs.
- The remaining editor upload workers also transfer their database lease through their results: the individual-upload fallback (whose nested lease the batch worker releases while its own is still held), the batch preparation worker (whose lease passes to the batch upload launched from it), and the normalization worker (whose lease passes to the post-normalize upload) — a logbook switch during any of those stages can no longer close the retired database before the next stage or the reconciliation chain starts, and the batch completion retains ownership until every required follow-up finishes or acquires its own lease.

## v0.10.1 — 2026-09-20

> **HTTPS dashboard, unified keyboard conventions, and whole-logbook search.** The built-in dashboard can now serve over HTTPS with an auto-generated self-signed certificate, every form saves with Enter, and the logbook editor searches the entire logbook, not just the visible page.

### Dashboard
- **Optional HTTPS**: enable TLS in the Integration menu; CQOps generates a self-signed ECDSA P-256 certificate in the cache directory when none is configured, serves it with TLS 1.2+, and redirects plain HTTP on the TLS port to HTTPS.

### Logbook Editor
- **Whole-logbook search**: the search field now queries the entire logbook (callsign, name, country, contest-scoped) instead of filtering only the loaded page; the status bar shows a `Search N/M` counter.
- **Enter to save**: Enter opens a Save confirmation dialog; Cancel returns to the list. Esc exits the editor.
- **No vim keys**: `j`/`k` navigation removed everywhere so those letters reach text fields.

### Keyboard Conventions
- Enter saves on station, rig, operator, contest, and wizard forms; Space toggles checkboxes and triggers Test buttons; Esc goes back.
- Main menu: digit shortcuts 1–8 jump directly to screens.
- PSK Reporter filters now use `t` (time), `b` (band), `m` (mode) like the DX Cluster pane; Backspace clears all filters.
- DX Cluster and REF screens: Enter logs a QSO for the selected entry.

### Fixes
- DXC status indicator no longer blinks when spot batches rebuild the table.
- Toasts keep expiring while a confirm dialog is open.
- Wavelog downloads continue after leaving the logbook editor.
- APRS pane keeps manual filters across re-entry; sub-minute ages render as "less than a minute ago"; radar zoom follows the distance filter.
- QTH and grid fields track entry precedence to protect manual and WSJT-X values.
- Build scripts verify Go is on PATH before building.

## v0.10.0 — 2026-09-20

> **APRS nearby-stations pane, receive-only mode, and automatic passcodes.** CQOps gains a full APRS neighbourhood screen (F3) with filters, details and an ASCII radar, receive-only operation when the logbook has no APRS config, and the APRS-IS passcode is now computed from the callsign instead of being stored.

### APRS Nearby (F3)
- **New APRS screen**: F3 opens "APRS Nearby" — a station table (symbol, Yaesu-style type code, callsign, bearing, distance, age) styled like the DX Cluster pane, next to a detail panel showing the selected station's callsign and symbol, grid, course/speed, decoded weather, source and last heard.
- **Filters**: distance (All, 1, 5, 10, 25, 50, 100 km — the pane opens at the step closest to the configured range), last-heard time, and type (All / operators only, now including alternate-table symbols such as `\k`). Backspace clears all filters.
- **ASCII radar**: a round radar around your own position with Yaesu-style type markers, stacked-station counts, near-zero stations marked at the center, N/E/S/W labels, and a range caption with your grid and the selected station's bearing and distance.
- **Own-station filtering**: your own beacon echo is hidden (exact transmitting SSID; an omitted SSID equals -0 per the APRS spec). Receive-only mode hides all SSIDs of your own callsign.
- **Manual beacon**: press `b` to send a position beacon immediately.

### APRS Modes & Config
- **Receive-only mode**: enabling APRS in Integrations without a logbook APRS config now runs a receive-only client (default 100 km range filter) — stations are cached for the F3 pane and the dashboard map, nothing is ever transmitted. The status bar shows `APRS-RX`.
- **Automatic passcode**: the APRS-IS passcode is computed from the login callsign (verified against live servers) and removed from config and the station form. The APRS callsign field is prefilled from the station callsign and **Send beacons** defaults to on; the beacon interval minimum is 5 minutes.
- **Weather decode**: weather-station comments (`_` symbols) are decoded into readable wind, temperature, humidity and pressure, with metric/imperial units.
- **Clock-skewed WX stations**: future packet timestamps within a day are clamped to the arrival time instead of being rolled back a month, so stations with fast clocks no longer disappear from the list.

### Documentation
- **Manuals updated**: the APRS sections in all ten manuals (EN, DE, ES, FR, IT, JA, PL, PT, RU, ZH) now document the F3 pane, receive-only mode, filters, radar and automatic passcodes.

### Under the Hood
- **New tests** for the APRS pane (filters, radar geometry, detail lines, weather), passcode vectors, packet-type classification, symbol names and timestamp parsing. No config or database migration needed from v0.9.10.

## v0.9.10 — 2026-08-09

> **Configurable rig poll, log hygiene, and UI fixes.** Hamlib/flrig poll interval is now configurable per rig, application logs are quieter and safer, and several menu navigation/viewport refresh regressions are resolved.

### Rig Control
- **Configurable poll interval**: `poll_interval_s` (1–60 s, default 1) added to each rig preset. Adjust via Rig menu → Edit Rig, visible when a radio backend (Hamlib or Flrig) is selected. A warning toast appears if the value is clamped to the valid range.
- **Backoff summary**: a single summary line is logged after repeated hamlib connection retries instead of one log line per failed attempt.

### Log Hygiene
- **Duplicate QSO log removed**: Wavelog duplicate-skip no longer emits a DEBUG log per QSO; the existing summary line already counts skipped QSOs.
- **Wavelog raw JSON body removed**: the `private_lookup` raw response body is no longer logged (callers provide their own context).
- **Email redacted**: callbook provider data logs now show `email=***` instead of the actual email address.
- **Structured toast logging**: toast log lines use `msg="toast" text="..."` instead of plain concatenation.
- **GPS connection failure**: GPSD unreachable toast is now a warning (yellow) instead of an error (red).

### UI Fixes
- **Menu viewport refresh**: fixed stale rendering after Ctrl+S/Enter save in Rig, Logbook, and Operator menus — the viewport now properly refreshes when returning to list view and when re-entering edit forms.
- **Callbook toast shortened**: multi-provider toast now shows `"Callbook: CALL — PrimaryProvider +N more"` instead of listing every provider.

### Under the Hood
- **No config or database migration needed** from v0.9.9. New `poll_interval_s` field defaults to 1 when absent.

## v0.9.9 — 2026-08-09

> **Logbook flexibility, GPS grid fix, and WinGet CI hardening.** Multiple logbooks can now share the same callsign, the GPS-derived grid updates immediately on the path line, and the WinGet release automation has been hardened for reliable PR submission.

### Logbooks
- **Same-callsign logbooks allowed**: you can now create multiple logbooks with the same station callsign (e.g. "SP9SPM Home" and "SP9SPM Portable"). The logbook ID uses a collision-safe scheme that includes the logbook name; existing single-callsign logbooks keep their original IDs. No config or database migration needed.

### GPS
- **Path line cache fix**: the station info line now updates immediately when GPS acquires a fix or refines the grid precision. Previously the render cache used only the static `Station.Grid` field, so a GPS grid update (e.g. 6-char → 10-char) would not appear until a restart or unrelated cache bust.

### WinGet CI
- **Fork sync**: the release workflow now fast-forwards the `szporwolik/winget-pkgs` fork before submission, preventing the ~11k commit backlog from causing silent submission failures.
- **ReleaseNotes patch**: the generated WinGet manifest is now patched to include the `ReleaseNotes` text field required by winget-pkgs validation.
- **Reliable PR submission**: switched from `wingetcreate --submit` (which had intermittent GitHub connectivity issues from the runner) to a split generate + `gh` CLI PR workflow.

### Under the Hood
- **No config or database migration needed** from v0.9.8.

## v0.9.8 — 2026-08-07

> **Maintenance release.** First public publish to WinGet, with release automation kept in sync for the Windows installer and packaging flow.

### Distribution
- **WinGet publishing enabled**: CQOps is now submitted to WinGet as part of the release workflow, so the Windows package can be published automatically once the release is cut.
- **Installer flow aligned**: the release pipeline now uses the same Windows installer artifact for both GitHub Releases and WinGet submission.

### Under the Hood
- **Maintenance update**. No config or database migration needed from v0.9.7.

## v0.9.7 — 2026-08-03

> **GPS & APRS polish.** Faster GPS feedback, APRS now uses a live GPS-derived position when available, and batch uploads no longer hit the wrong canonical path.

### GPS & APRS
- **Immediate GPS polling**: the GPS status bar now refreshes right away on startup and after integration settings changes, so the live fix state appears without waiting for the next periodic poll.
- **Live APRS position**: APRS restarts with the current GPS-derived position as soon as a fix is acquired, and the APRS range filter now uses that live grid when available.
- **Batch upload path fix**: the batch upload flow no longer carries `MY_GRIDSQUARE` into the canonical upload path, avoiding mismatched routing.

### Under the Hood
- **~3 commits**, **~3 files changed**. No config or database migration needed from v0.9.6.

## v0.9.6 — 2026-08-01

> **Reliability & polish.** GPS status bar responsiveness, Wavelog download UX fixes, logbook QSO counts, DB perf, and CI tuning.

### GPS — Status Bar Responsiveness
- **Immediate first-tick poll**: GPS status bar now reflects the real fix state within 1 second of startup instead of waiting up to 60 seconds for the first periodic poll.
- **Instant update after config save**: enabling GPS from the integration menu and saving immediately polls the GPS client — no 60-second delay for the indicator to turn green.
- **Test button feedback**: a successful GPS test now triggers an immediate status bar refresh, turning the indicator green right away when a fix is present.
- **APRS range filter**: the APRS-IS connection now uses `EffectiveGrid()` (GPS-derived position when available) for the range filter instead of the static configured grid. Dashboard map now shows stations near your actual GPS location.

### Wavelog Download
- **Race condition fix**: the "Downloaded 0 QSOs" dialog after fast downloads is fixed. The render now falls back to the live progress counter (`dlCurrent`) when the done handler hasn't fired yet — matching the existing ADIF import/export pattern.
- **Log spam eliminated**: individual duplicate QSO detections moved from `WARN` to `DEBUG` level (10,708 lines → 0 at WARN). Message reworded from misleading "already imported this session" to accurate "already in local logbook". Final summary line (`inserted=3 dupes=10708`) still reports totals at INFO.

### UI — QSO Counts
- **Station profile line**: the info row between the QSO form and recent QSOs now shows logbook-wide counts — `"IC-7300 / HF9V · Grid KO00ca · 123 QSOs · 5 today"`. Today count only appears when non-zero and there's space. Counts rebuild automatically on QSO save, WSJT-X log, and logbook switch.
- **Configuration → Logbooks**: each logbook now shows its QSO count in the list (`"1661 QSOs"`, `"21 QSOs"`). Counts are fetched asynchronously when the screen opens.

### Config Menu — Logbook Switch
- **Immediate QSO refresh**: switching the active logbook via the config menu now immediately reloads the recent QSOs table and partner info caches — matching the hotkey (`Ctrl+L`) behavior. Previously the refresh was deferred and could miss on certain screen transitions.

### Database
- **Composite indexes**: added `(call, band, mode, qso_date)` and `(base_call, band, mode, qso_date)` for the dupe-check query that fires on every QSO form field change. Replaces 4 separate index lookups with a single B-tree seek.
- Schema version bumped to 2 (idempotent — indexes use `CREATE INDEX IF NOT EXISTS`).

### CI — Branding Validation Relaxed
- **ShellCheck**: severity lowered to `--severity=warning` — style nits (SC2034, SC2155) no longer fail the build.
- **`.syso` determinism**: changed from hard error to warning — `go-winres` output is not strictly byte-identical across versions.
- **AUR PKGBUILD**: exact `diff -u` replaced with per-field key comparison — tolerant of `makepkg` formatting differences between Arch versions.
- **Casing check**: exclusions expanded to `go.sum`, `go.mod`, `third_party/`, `licenses/`.

### Under the Hood
- **~12 commits**, **~11 files changed**. No config migration needed from v0.9.5. Database migration adds 2 indexes (idempotent).

## v0.9.5 — 2026-07-22

> **Dashboard map fix.** Switched MapLibre GL CDN from unpkg to jsDelivr — unpkg.com serves JavaScript with MIME `text/plain`, which browsers block under `X-Content-Type-Options: nosniff`. Also fixed a version compatibility gap between `maplibre-gl` and the Leaflet binding.

### Dashboard
- **CDN switch**: `unpkg.com` → `cdn.jsdelivr.net` for `maplibre-gl.js`, `maplibre-gl.css`, and `leaflet-maplibre-gl.js`. jsDelivr reliably serves correct `application/javascript` MIME types.
- **Pinned versions**: `maplibre-gl@5.24.0` and `@maplibre/maplibre-gl-leaflet@0.1.3` — the previous `0.0.22` binding only supported `maplibre-gl` up to 4.x, causing `maplibregl` to be undefined when paired with 5.x.
- **CSP updated**: `Content-Security-Policy` headers now allow `https://cdn.jsdelivr.net` (replaced `https://unpkg.com`) for `script-src`, `style-src`, and `worker-src`.

### Under the Hood
- **~4 files changed**. No config or database migration needed from v0.9.4.

## v0.9.4 — 2026-07-21

> **DXC path line, branding consistency, and dashboard polish.** Smart spot filtering, DXCC badge fix, hamlib mode recognition, band plan tune fix, and unified product identity across all packaging channels.

### Cross-Platform Branding Consistency
- **Unified product identity**: all package descriptions, desktop entries, AUR metadata, Windows installer, executable resources, README, CLI help, dashboard headers, documentation, and CI tooling now share a single canonical branding source (`scripts/branding.sh`) with automated validation (`scripts/validate-branding.sh`).
- **Tagline**: *"Less clicking. More radio."* now appears in README, dashboard, and all 10 manual translations.
- **Descriptions**: replaced implementation-centric *"Fast, minimal Go TUI ham radio logger"* with the canonical *"Fast, offline-first amateur radio logger for the terminal"* across all channels.
- **Icon fix**: AUR `.desktop` entry now uses `Icon=cqops` instead of the generic `Icon=utilities-terminal`. Desktop file simplified to `Categories=Network;HamRadio;`.
- **Windows metadata**: `FileDescription`, `CompanyName`, `LegalCopyright`, and `ProductVersion` corrected in both `winres.json` and compiled `.syso` resources. `.syso` determinism verified in CI.
- **AUR improvements**: PKGBUILD and `.SRCINFO` share a single `pkgdesc` variable to prevent drift. Canonical `.desktop` file sourced from the tagged release instead of a duplicated inline heredoc. Real SHA-256 checksum for the desktop file.
- **CI validation**: new `branding-check.yml` workflow (6 jobs) — ShellCheck + branding validation, DEB/RPM build + inspection, Arch `makepkg --printsrcinfo`, `CGO_ENABLED=1 go test -race`, NSIS compilation + metadata assertions, `.syso` determinism.
- **9-language docs**: dashboard Header 2 fallback updated to the tagline in all manual translations. Non-English introductory paragraphs preserved (awaiting native-speaker translation pass).

### DXC Path Line — Smart Filtering
- **Same-continent filter**: spots above the QSO form now default to showing only spots heard from the same continent as the station (`SpotCont` matches configured `Continent`). Cascading fallback if no matches.
- **Same-mode filter**: spots are filtered by mode category — `CW`, `DIGI` (FT8/FT4/RTTY/PKT), or `PHONE` (SSB/AM/FM). Switches automatically when you change mode on the radio.
- **15-minute time window**: spots older than 15 minutes are excluded by default. Falls back to time-only if continent+mode filters produce no results.
- **Cache invalidation**: added generation counter (`rawGen`) so the line redraws immediately when new spots arrive, even when the 600-spot buffer is full.
- **Ctrl+P reference auto-fill**: cycling through on-frequency spots now parses the spot comment for SOTA/POTA/WWFF/IOTA references and auto-fills the QSO form.
- **DB index**: new composite index `idx_dxc_spots_band_time` on `dxc_spots(band, received_at)` for efficient band+time queries.

### Dashboard — New DXCC Badge Fix
- **Case-insensitive DXCC matching**: `countryWorkedBefore` now uses the same multi-strategy matching as the F2 partner page — DXCC entity number, case-insensitive country name, and prefix LIKE matching. Fixes "New DXCC" badge incorrectly showing when the DXCC was already worked but stored under a different country name casing (e.g. `SAUDI ARABIA` from Wavelog vs `Saudi Arabia` from QRZ).

### Mode & Rig Compatibility
- **Hamlib mode recognition**: `spotModeCategory` now recognizes `PKTUSB`, `PKTLSB`, `CWR`, `RTTYR`, `FMN`, `WFM`, `SSB`, and `PKT` variants — fixes mode filtering being silently disabled when the rig reports hamlib-specific mode names.
- **Rig mode normalization**: `rigModeMap` now maps `PKTUSB`/`PKTLSB`/`PKTFM` → `PKT` and `FMN` → `FM`.

### Fixes
- **Band plan SAT false positive**: tuning a broadcast station (e.g. "Radio Antena Satelor") no longer incorrectly selects FM mode. Satellite detection now uses word-boundary matching.
- **Default continent**: logbooks without an explicit `continent` field now default to `EU`, so the DXC path line continent filter works on upgraded configs.
- **Test suite**: two stale tests fixed. All 32 test packages pass.
- **Code cleanup**: removed dead `gpsTickMsg` type, converted callbook test handler to tagged switch, marked unused parameter.

### Under the Hood
- **~25 commits**, **~27 files changed**. New DB index added (idempotent `CREATE INDEX IF NOT EXISTS`). New CI workflow for cross-platform branding/packaging validation. No config migration needed from v0.9.2/v0.9.3.

## v0.9.3 — 2026-07-21

> **Polish release.** First-run wizard improvements, secret masking, navigation fixes, and AUR packaging complete.

### First-Run Wizard
- **Enter = save & next**: Enter now works as Ctrl+S in all wizard steps (Station, Rig, General, Summary). Wizard header shows `[Enter — save & next]` hint.
- **Operator field removed**: the wizard no longer shows the operator selector — operators don't exist yet during first-run setup.
- **Navigation fixes**: Tab/Shift+Tab now correctly wraps between fields. Fixed rig name field not receiving focus (missing `case rigFieldName` in `focusField()`). Fixed off-screen navigation when Wavelog or WSJT-X is disabled.

### Security — Secret Masking
- **Wavelog API key**: now masked as `***` in the station config form. Shows plaintext only when the field is focused for editing. Same for APRS passcode.
- **Masking logic**: secrets are masked on blur, revealed on focus, and re-masked before save. The underlying `secrets.enc` encryption was already in place — this change only affects UI display.

### UI Polish
- **Rotator hint**: the "Experimental feature — use with caution" text is now hidden on terminals narrower than 85 columns — no more wrapping.
- **AUR PKGBUILD**: now includes SVG app icon and `.desktop` entry so CQOps appears in the application menu with the proper icon on Arch-based systems.

### Under the Hood
- **~10 commits**, **5 files changed**. No config or database migration needed from v0.9.2.

## v0.9.2 — 2026-07-18

> **Maintenance release.** New packaging target and distribution polish — no application code changes from v0.9.1.

### Arch Linux / AUR
- **`cqops-bin` published on AUR**: Arch, Manjaro, CachyOS, and other Arch-based distros can now install via `yay -S cqops-bin`. PKGBUILD downloads the pre-built binary from GitHub Releases — no compilation needed.
- **Automated updates**: the release workflow now pushes PKGBUILD and `.SRCINFO` updates to AUR on every release, keeping `pkgver` and `sha256sums` in sync automatically.
- **AUR badge**: version shield added to README header alongside Cloudsmith.

### Distribution
- **README**: added installation section for Arch Linux, Manjaro, and CachyOS with AUR instructions.
- **Release workflow**: AUR publish job fixed — switched from `.deb` extraction to `.tar.gz`, added `.SRCINFO` generation, and configured `git` identity for automated commits.

### Under the Hood
- **~5 commits**, **2 files changed**. No config or database migration needed from v0.9.1. Same binary — only packaging and docs updated.

## v0.9.1 — 2026-07-16

> **Performance release.** CPU usage on low-end hardware (Raspberry Pi, old laptops, portable field setups) dropped from ~107% to ~13% — a 92% reduction.

### Performance — Potato PC Optimizations
- **FPS cap**: renderer limited to 20 FPS via `tea.WithFPS(20)`. CQOps is tick-driven at 1 Hz — higher frame rates provide no visual benefit but waste CPU on low-end hardware.
- **GPS polling merged**: GPS position now polls every 60 seconds inside the main tick instead of running a separate 1-second ticker. Eliminates one full Update→View cycle per second.
- **Dashboard stats event-driven**: today QSOs, aggregate stats, and recent QSOs are now recomputed only when a QSO is saved, logbook changes, or data is imported — not on a 5-second timer. Avoids 4 expensive COUNT/DISTINCT aggregation queries every 5 seconds.
- **WorkedSummary cache**: `GetWorkedSummary()` (16 SQL queries) was running inside `View()` on every frame. Now cached with a `call|grid|DXCC|country` key and only recomputed when inputs change. `viewPartner` CPU dropped from 16.68% to 2.61%.
- **stripANSI rewrite**: regex-based ANSI escape stripping replaced with a byte scanner including a fast-path for strings without escape sequences. Removes the `regexp` import from the dashboard push path. ~5× faster on low-end CPUs.
- **countryWorkedBefore cache**: per-country `COUNT(*)` query now cached per `(country, baseCall)` pair. Cleared on QSO save.
- **View() JoinVertical → plain newline**: the main view compositing (status + tabs + body + help) now uses `strings.Join` instead of `lipgloss.JoinVertical`. Avoids the Lip Gloss line-measurement pass on every frame.
- **Profiling**: new `--pprof` flag starts a `net/http/pprof` server on `127.0.0.1:6060` for CPU, heap, goroutine, mutex, and block profiling. Zero overhead when not used.

### New Callbook Provider — QRZ.RU
- **QRZ.RU**: free global callbook with name, QTH, grid, DXCC, and photo. Russian-hosted; good coverage for Eastern European and Asian callsigns. Config, secrets, menu integration, and dashboard support included.
- **HamQTH default images**: placeholder images from `hamqth.com/images/default/` (e.g. `paddle_and_notebook.jpg`) are now treated as "no image" so lower-priority callbooks (QRZ, Callook) can supply real photos. Genuine HamQTH operator photos pass through unchanged.

### UI Polish
- **Compact status bar**: operator callsign shown without parenthetical name (`Op SP9SPM` instead of `Op SP9SPM (Szymon)`). Log/Call/Op labels merged when values are identical — e.g. `Log/Call/Op SP9SPM` instead of three separate fields. Rig name shown when model differs from operator.
- **Partner identity line**: US state (short codes), short continent names, and local time for DX stations.
- **Worked panel**: distribution labels prefixed with scope (`Call Bands`, `DXCC Modes`, `Grid Grids`). Local logbook and Wavelog data merged into a single unified panel with compact row format.
- **DXC table**: new DXCC, band, and mode cells highlighted with color when they're new for the operator.
- **Scroll indicators**: BPL, DXC, and REF views now show `▲ more above` / `▼ more below` hints when content overflows.
- **REF view**: title header, scroll indicator, and proper table height constants added.
- **Logbook editor**: new search-by-call, country, and name filter bar.
- **PSK Reporter**: side-by-side layout replaced with DXC-style filter bar and full-width map.

### Fixes
- **Wavelog download key**: changed from `Ctrl+W` to `Alt+W` to avoid conflict with the `Ctrl+W` close-tab shortcut in terminal multiplexers.
- **Contest QSO matching**: all contest-related queries now match by `contest_adif_id` — fixes edge cases where QSOs from different contests with the same callsign were incorrectly cross-matched.
- **Big CTY**: longitude sign convention corrected (west positive → negative). Redundant DXCC grid auto-fill removed. DXCC lookup migrated from `cty.dat` binary format to Big CTY CSV with ADIF entity numbers. Full DXCC backfill after bulk Wavelog download.
- **Map**: station marker always renders on initial load. Map height cap removed — fills available space on tall terminals. Tile CRS correctly re-initialized on internet restore.
- **Dashboard**: SSE subscriber buffer increased from 16 to 128 to prevent event loss. Redundant operator/logbook SSE events suppressed. Force-push on reconnect and logbook/rig/contest/operator toggle. Debug mode propagated from TUI.
- **Internet detection**: adaptive polling with debounced offline detection; toast spam suppressed after first disconnect.
- **Help bar**: `operatorForm` cache key fixed for edit mode. `WL` abbreviation expanded to `Wavelog`.
- **Rotor**: `Ctrl+F1` stop binding removed; status dot resets correctly on stop.
- **Contest duration**: format simplified to minutes or `H:MM` — seconds dropped.
- **HTTP dashboard**: address text input replaced with a toggle (`127.0.0.1` / `0.0.0.0` / custom).
- **Station info**: `stripNonDigits` moved to `qso` package. Flrig/hamlib defaults and unused helpers removed. GeneralMenu cursor index fixed for 10-item list.

### Under the Hood
- **~70 commits**, **~45 files changed**. All 30 test packages pass. No new dependencies, zero cgo. Backward-compatible — no config or database migration needed from v0.9.0.

## v0.9.0 — 2026-07-14

> **Breaking change.** This release introduces `config_version: 1` and database `PRAGMA user_version = 1`. Config is auto-migrated on load (legacy keys → nested callbook structure). Database migrations are applied once and skipped on subsequent starts. No manual intervention required for upgrades from v0.8.7+; pre-v0.8.7 databases are re-backfilled automatically.

### Config v1 — Nested Callbook & Legacy Migration
- **Callbook group**: all callbook providers moved under `integrations.callbook:` (QRZ.com, HamQTH, Callook.info) with per-provider priority and credentials. The old flat keys (`QRZLegacy`, `qrzcom_callbook`, etc.) are auto-migrated on load.
- **Config versioning**: `config_version: 1` written on save. Future migrators can branch on this field.
- **Legacy key migration**: `picture_at_qrz_pane` → `picture_at_partner_pane`, `wavelog_sent` → `qso_sent`, `wavelog_errors` → `all_errors`. Old keys are silently upgraded.
- **CTY.DAT always-on**: the prefix-to-DXCC lookup now runs unconditionally — removed from the General menu toggle. Always available as the ultimate callbook fallback.

### Multi-Provider Callbook
- **HamQTH**: free global callbook with name, QTH, grid, country, CQ/ITU zones, and DXCC. Requires a free account.
- **Callook.info**: free US-focused callbook. No account needed — fast FCC lookups for US callsigns.
- **Priority cascading**: providers are tried in configured priority order. When a higher-priority provider fails or is disabled, the next is queried automatically.
- **Base-call fallback**: when enabled (default: on), CQOps also tries the base callsign (e.g. `SP9MOA` from `DL/SP9MOA/P`) if the full call returns no match from any provider.
- **Callbook menu**: new `F9 → Callbook` top-level menu — separate from the Integration menu. Configure providers, test connections, set priorities.
- **Wavelog lookup**: moved into the callbook pipeline as a provider with its own priority slot. Worked/confirmed status from Wavelog is merged with callbook data.

### Offline Resilience
- **Embedded world map**: a ~150 KB equirectangular map image is compiled into the binary. The dashboard falls back to it when internet is unavailable, using EPSG:4326 CRS for correct alignment.
- **Graceful degradation**: dashboard tiles, weather, radar, and QR codes all degrade cleanly offline. No broken images, no JS errors.
- **Offline toast suppression**: network-error toasts fire once on first detection, then go silent. Enables `--offline` for clean portable/field operation without notification spam.
- **Beep suppression**: desktop notification sounds are suppressed when offline — no alert-spam during field ops.

### Dashboard Overhaul
- **Dark theme**: full dark mode with OpenFreeMap dark tiles, dark-themed CSS, and localStorage persistence (applied before first paint). Theme names `bright`, `dark`, `yl`, and `hivis`.
- **Operator badges**: deterministic colour-hashed operator badges in the recent-QSOs table. Consistent colours per operator across sessions.
- **QR link**: configurable QR code link in the dashboard header — defaults to `docs.cqops.com`.
- **Top-QSOs table**: replaced the old definition list with a proper styled table matching the recent-QSOs column layout.
- **Disconnected overlay**: logo + elapsed-time timer when SSE connection drops. Auto-recovers on reconnect, including CRS reset for tile maps.
- **Mid-width responsive breakpoint**: stats panel and table columns adapt at intermediate widths (was missing between narrow and wide tiers).

### Database — Schema Consolidation & Versioning
- **Clean migrations**: all historical `ALTER TABLE` additions folded into the base `CREATE TABLE`. Removed the botched-migration recovery `DELETE` and startup messages to stderr. Migrations are silent and idempotent.
- **Schema versioning**: uses SQLite `PRAGMA user_version` (currently `1`) — migrations are skipped when the database is already at the current version. Future schema changes can target specific version gaps without re-running already-applied work.
- **DXCC column**: `dxcc` entity number added for remote stations (populated by callbook providers).

### Security Hardening
- **Dashboard default bind**: changed from `0.0.0.0` (all interfaces) to `127.0.0.1` (localhost only). Users who need LAN access must set the address explicitly — the safe default protects field operators on public networks.
- **Radar proxy sanitization**: `/radar-proxy/` endpoint now rejects paths containing `..` — prevents path traversal to the upstream CDN.

### Bug Fixes
- **Portable dupe detection**: `IsDuplicateQSO` now matches on `base_call` in addition to exact `call` — logging `SP9MOA` after `DL/SP9MOA/P` on the same band/mode/date correctly shows `DUPE!`.
- **Nil deref guards**: `cycleActiveContest()` and `cycleActiveOperator()` now check `m.App.Logbook != nil` before accessing logbook fields — prevents panic when the active logbook is unset.
- **PSK spot count**: `InsertPSKSpots` now returns `0` (not the pre-commit tally) when `tx.Commit()` fails — the caller no longer receives a misleading count of unpersisted inserts.
- **SSB submode**: force-corrected to `SSB` on frequency change to prevent stale submode values from carrying over between bands.
- **APRS beacon clamp**: interval clamped to 5–180 minutes with save-time validation — out-of-range values no longer produce beacon storms or silent failures.
- **ADIF STX_STRING/SRX_STRING**: now exports the exchange stripped of the RST prefix per ADIF spec (`STX=599` → `STX_STRING=001`, not `STX_STRING=599 001`).
- **Help bar**: `operatorForm` cache key fixed for edit mode — switching between logbook list and operator editor no longer shows stale shortcuts.
- **Wavelog lookup guard**: skips Wavelog result when only DXCC prefix data (not actual Wavelog worked/confirmed) was returned.

### Translations
- **9 languages**: English, Polski, Deutsch, Español, 日本語, Français, Italiano, **Português (BR)**, and **Русский**. All manuals updated with multi-provider callbook, offline map, and config restructure.
- **Exchange markers**: all manuals corrected for the 8-template-marker set (`@rst`, `@serial`, `@cqz`, `@mycqz`, `@itu`, `@myitu`, `@grid`, `@mygrid`).

### CI / Build
- **UPX compression**: binaries are now compressed with `upx --best` in CI — Linux amd64/arm64/armhf and Darwin amd64/arm64. Typical 40–60% size reduction.
- **Winget**: disabled until the manifest PR is accepted by Microsoft. Will re-enable as a fast-follow release.

### Under the Hood
- **~70 commits**, **~95 files changed**. Config auto-migration tested with real v0.8.x `config.yaml`. All 30 test packages pass. No new dependencies, no cgo, no runtime API changes for ADIF, Wavelog, WSJT-X, flrig, or rigctld backends.

## v0.8.13 — 2026-07-12

### Contest Statistics Panel
- **Live stats panel**: when a contest is active and the terminal is wide enough, a compact statistics panel appears to the right of the QSO form with a yellow border. Shows Rate (last 10/100 QSOs), Count (last 60m / current hour), Peak (best 1m/10m/60m sliding window), Avg (session average + duration).
- **Activity chart**: Unicode block-character (`█`) vertical bar chart showing QSOs per minute over the last 60 minutes, scaled to 4 rows.
- **Bottom status bar**: contest line shows ID, name, total QSOs, first QSO time, time since last QSO, next serial number, and on-air time. Responsive — fields hide on narrow terminals.
- **On-air time**: computed as sum of inter-QSO gaps shorter than 30 minutes — approximates active operating time vs idle.
- **Performance**: panel render is signature-cached (like solar panel) — rebuilds only when data changes, not every frame. Data refreshes every 5 seconds. Pre-sized allocations, no goroutine leaks.
- **Accurate totals**: TotalQSOs uses `COUNT(*)` query instead of `len(qsos)` — no longer capped at 1000 rows for active contests.
- **DB index**: composite index `idx_qsos_contest_date_time` so the ListQSOs contest query satisfies both WHERE and ORDER BY from a single index scan.

### REF Search — Diacritic and Case Insensitive
- **Unicode-aware search**: `normalizeForSearch()` strips diacritics and lowercases — `ćwilin`, `cwilin`, and `Ćwilin` all find `Ćwilin`. Uses `golang.org/x/text` NFD normalization.
- **search column**: new column in the refs table populated during rebuild. Existing databases auto-detect missing backfill and trigger a rebuild on next restart. Fallback preserves old behavior for unpopulated databases.
- **Backspace fix**: Backspace now works as normal character deletion in the REF search box. **Delete** key clears the entire search. Help overlay updated (`Del → Clear`).

### Keybinding Consistency Pass
- **Rotor**: removed `Ctrl+↑/↓` and `Ctrl+A` — `Alt+;`/`Alt+'`/`Alt+\` are the only rotor shortcuts now. No more conflict with rig tune or "select all" surprise.
- **DXC help**: continent filter label `\ → Sp Cont` (was vague `\ → Continent`).
- **Standardized**: all help bars now say `Space` instead of `Spc`.
- **Stale comment**: rotor handler doc fixed (was `Ctrl+R`, actually `Alt+\`).
- **Manual**: full keybinding section updated to match actual bindings. Favorites section corrected (3 slots via Alt+Ins/Home/PgUp, not 10 via Alt+0–9).

### Duration Display
- **Seconds dropped**: `formatDurationShort` now returns `H:MM` (≥1 hour) or `M` (<1 hour). Per-minute refresh makes seconds meaningless. `Sess 1:18` instead of `Sess 1:18:55`.

### ADIF Export — Contest Filenames
- **Contest-aware filenames**: when a contest filter is active, the exported filename includes the contest ADIF ID and date: `20260712_150405_sp9spm_IARU-HF_20260712.adi`.
- **OS-safe sanitization**: all filename-unsafe characters (`/ \ : * ? " < > |`) replaced with `-`, spaces with `_`.

### Distribution & Packaging
- **nfpm.yaml**: removed bogus `libc6` dependency (binary is statically linked, CGO_ENABLED=0). License corrected to Apache-2.0. RPM packaging support added.
- **Release workflow**: rewritten with `validate-version` job, RPM packages (x86_64 + aarch64), Cloudsmith publishing via OIDC, versionless `cqops_amd64.deb` for stable download links. SHA-256 checksums generated for all assets.
- **winres.json**: version auto-injected from `VERSION` file by build scripts. File description updated.
- **Package metadata**: descriptions in nfpm, NSIS installer, Windows resource file, install scripts, and .desktop file all updated to match README tone.
- **README**: new Installation section with WinGet, Cloudsmith APT/RPM, AUR, and Go methods. Release assets table includes RPM. Cloudsmith OSS hosting badge and attribution.

### Licenses
- Added `CHARM-X-TERM-MIT-LICENSE` for `github.com/charmbracelet/x/term`.
- Updated `third_party/NOTICE.md` with the new entry.

## v0.8.12 — 2026-07-12

### Recent QSOs Table — Full-Width + Smart Columns
- **Full terminal width**: the recent QSOs table is no longer capped at 140/200 columns. On large screens, it uses all available space — richer column tiers appear naturally and text-heavy columns (Name, QTH, DXCC) stop truncating. Small-screen behavior is unchanged.
- **Smart column caps**: every column has a reasonable maximum width — `Call` caps at 12, `Comment` at 30, `Band` at 7, `Mode` at 6, etc. Extra space on ultra-wide screens flows to text-heavy columns via iterative redistribution instead of blowing up short fields.
- **Notes column removed**: the rarely-used Notes column is removed from all tiers. Its 12-char allocation is redistributed to Name, QTH, Comment, and reference fields.
- **Reference fields breathe**: SOTA, POTA, WWFF, IOTA, SIG caps raised so they absorb leftover space on huge monitors instead of it all dumping to the last column.
- **Contest exchange columns**: when contest mode is active, `ExchSent` and `ExchRcvd` replace SOTA/POTA/WWFF/IOTA/SIG at the wide tiers. Non-contest behavior is unchanged.

### DXC Dupe Markers — Spotter-Aware
- **DXC table**: already-worked spots show a `D ` prefix before the callsign (dimmed) — visually distinct from new spots. A single batch query (`DXCDupeSet`) checks all spots against logged QSOs with zero per-spot DB access.
- **DXC path line**: dupe spots in the band-line above the QSO form use the same `D ` prefix convention for consistency.
- **Contest-aware**: in contest mode, dupe checks span the entire contest (48h+), not just today's date. Switching logbooks or contests invalidates the dupe cache automatically.
- **Instant refresh**: dupe markers update immediately after logging a QSO — no waiting for the next spot drain cycle.
- **SQLite covering indexes**: `idx_qsos_date_call_band_mode` and `idx_qsos_contest_call_band_mode` let SQLite answer dupe queries from the index alone, avoiding table scans on every DXC table rebuild.
- **Monochrome-safe**: all dupe markers use text characters (`D ` prefix), not just color, so they work on simple terminals and SSH sessions.

### IARU Region Fix
- **Region 0 default**: `Normalize()` now defaults unset `IARURegion` to 1 (Europe) for all logbooks. Previously, a missing config key silently mapped to Region 2 (widest) limits, causing incorrect out-of-band frequency warnings on 40m (red at 7.300 instead of 7.200 for EU stations).
- **Tests**: 40 new test cases for `IsInHamBand` covering all three IARU regions, band edges, and out-of-band frequencies.

### DXC Continent Filter Fix
- **Spotter continent**: the continent filter now operates on the spotter's continent (`SpotCont`) instead of the spotted station's continent (`DXCont`). Press `\` to filter for spots heard FROM a specific continent. Filter label updated to `Sp Cont`.

### UI Polish
- **Help bar decluttered**: `Ctrl+F` (Spot→Call), `Ctrl+↑` (Rig +step), and `Ctrl+↓` (Rig −step) removed from the default bottom bar. Still available via the `?` help overlay — keeps the bottom line clean on portable/small screens.
- **Dashboard favicon**: updated to the rebranded CQOps icon.

### Under the Hood
- **14 files changed**, 1 new test file (40 cases). No dependency changes, no config format changes, no breaking API changes.

## v0.8.11 — 2026-07-10

### Critical Fixes
- **Database orphan**: `NewID()` now uses deterministic SHA-256 hashing instead of `time.Now().UnixNano()`. Previously, running the wizard twice could produce different database filenames, leaving imported QSOs stranded in an orphaned file. The database is now also reopened after the wizard when the logbook changes.
- **WSJT-X auto-recovery**: tick handler no longer checks `rp.WsjtxEnabled` directly — it always delegates to `MaybeRestartWSJTX`, which has its own change-detection. This prevents the auto-recovery from re-enabling WSJT-X after the user intentionally disabled it.
- **Desktop notifications on Windows**: `desktopAvailable()` now returns `true` on Windows (`runtime.GOOS` check), fixing silent notification failures on Windows 10/11.

### Integration Fixes
- **Hamlib VFO name query spam**: VFO name query ("v" command) is attempted only once per connection via `vfoNameOK` flag. On rigs that don't support it (e.g., Xiegu G90), this eliminates 12,000+ retries per session.
- **Power clamping**: `clampRigPower()` applies `math.Floor` then clamps to the rig preset's max power, fixing a Xiegu G90 displaying 21W when set to 20W due to firmware rounding.
- **WSJT-X power priority**: `txPowerForWSJTX` now directly sets QSO power before `ApplyStationDefaults`. Previously, `ApplyStationDefaults` only filled empty fields, so the WSJT-X ADIF `tx_pwr=10W` survived even when the form showed 21W from hamlib.
- **Kitty guard**: `ensureKitty()` now checks `kittyTerminalEnv()` in addition to `picture.KittySupported()`, preventing false Kitty activation on terminals that pass the probe but lack true graphics support.
- **Dashboard HTTP/DXC mid-run enable**: stale backoff timers and incomplete state reset are now cleared unconditionally when disabled or offline, fixing services that wouldn't start after being toggled on mid-session with the "Enable when CQOps starts" checkbox off.
- **APRS double log**: removed redundant "APRS: connected" log from the client run-loop (the app-level `OnStatus` callback already logs it).

### Dashboard Performance
- **Active QSO dedup**: field-level cache comparison in `pushDashboardFast` skips `SetActiveQSO` and its debug log when nothing changed since the previous tick, reducing ~10,000 redundant pushes per session to 1 per change.
- **Partner dedup**: same field-level cache for partner lookups — `partner pushed` log fires only when QRZ/Wavelog data actually changes, not every tick.
- **Empty partner guard**: `partnerEmpty` flag prevents the "partner cleared" debug log from firing every tick when no call is entered.
- **Dashboard throttle**: `pushDashboardState` now throttles to every 2 ticks (~2s) instead of every tick (~1s). The dashboard SSE push is already change-detected, so the slightly slower poll rate is imperceptible while halving per-tick CPU overhead on low-end hardware.

### Log Cleanup
- **`!BADKEY` fixes**: structured log keys added for dashboard listening URL and APRS reconnect delay.
- **Duplicate debug logs**: `desktopAvailable()` result is cached via `sync.Once` — the "notify: desktop check" debug line fires once at startup instead of twice (or more on repeated calls).
- **Double cursor on Windows**: `\033[?25l` at startup hides the conhost block cursor, preventing a double-cursor artifact alongside Bubble Tea's text cursor.

### Under the Hood
- **14 commits**, 11 files changed (162 insertions, 46 deletions).

## v0.8.10 — 2026-07-09

### Kitty Graphics Protocol — Terminal-Native Images
- **Kitty graphics support** for partner map, PSK Reporter map, inline partner photo, and full-screen photo viewer (F2). No external viewer needed — images render directly in supporting terminals (Kitty, WezTerm, Konsole ≥24.08, Ghostty).
- **Graceful fallback**: ANSI half-block map + Unicode glyph photo placeholder on non-Kitty terminals. Zero configuration — `charm.land/bubbles/v2/picture` handles capability detection.
- **Kitty toggle** in General Settings (`kittyGraphics`) — can be disabled to force ANSI/glyph rendering.
- **Photo viewer**: full-screen image (F2) with ESC to close; Kitty protocol handles sizing and placement automatically.
- **Map cache**: Kitty image dimensions are tracked to avoid redundant re-encodes; only rebuilds when inputs (grid, grayline, window size) actually change.

### GPS Integration — Serial NMEA + GPSD
- **Serial GPS**: connect a USB/RS-232 NMEA receiver (e.g. u-blox) — configure port, baud rate, DTR/RTS in F9 → Integrations → GPS.
- **GPSD**: TCP connection to a local `gpsd` daemon with host/port configuration.
- **Grid precision control**: choose 6, 8, or 10-character Maidenhead grid (F9 → Integrations → GPS Precision). Controls accuracy of position shared via APRS beacons, QSO logging, and dashboard.
- **GPS Grid** toggle on station form: use GPS-derived grid instead of fixed station grid. Auto-updates on position change.
- **APRS beacon grid**: respects GPS precision setting — never transmits a more accurate grid than the user configured.
- **Dashboard weather**: falls back to GPS-derived coordinates when available for Open-Meteo location resolution.

### APRS — KISS TNC Support
- **KISS TNC** (serial/TCP): send APRS position beacons and receive packets via a hardware TNC or software modem (Direwolf, QtSoundModem). Configure in F9 → Integrations → APRS.
- **KISS Server** mode: connect to a remote KISS-over-TCP server for shared TNC access.
- **APRS-IS** (existing): unchanged — internet-based APRS reporting continues to work alongside KISS.
- **Station trails**: dashboard shows last 5 position points for each APRS station with directional arrows on the map.
- **AX.25/KISS tests**: comprehensive test coverage for frame encoding/decoding.

### Portable SOTA/POTA Starting Areas
- **New "Portable" tab** on the Band Plan screen (F7 → right-arrow to PORT). Per-IARU-region suggested CW and SSB starting areas for QRP/portable/SOTA/POTA operations (40m–10m).
- **Not official channels** — clearly labeled as suggestions. Always check band plans, listen, ask QRL, spot exact frequency.
- **Markdown export** (Ctrl+E) includes the Portable section.
- **Data sourced** from IARU Region 1/2/3 band plans and practical field reports.

### Dashboard Enhancements
- **Metric/imperial units**: temperature (°C/°F), wind speed (km|m/h, mph, kn), precipitation (mm/in) — configurable in F9 → General.
- **APRS station trails**: directional path history on the Leaflet map with marker arrows.
- **Wind speed & precipitation formatting**: unit-aware display in the weather module.
- **APRS map**: nearby station markers now use standard APRS symbol icons with improved popup positioning.

### Linux TTY & Bare Terminal Support
- **Bare TTY detection**: auto-detects `TERM=linux`, `XDG_SESSION_TYPE=tty`, or framebuffer console (no `$DISPLAY`).
- **Forced screen clear**: on bare TTYs, `tea.ClearScreen` is issued on every keypress at the outermost `Update()` level — unstoppable by screen handlers.
- **ANSI 16-color palette**: automatic fallback when terminal doesn't support 256 colors.
- **tmux auto-launch**: on Linux console (no desktop), CQOps auto-launches inside `tmux` for proper function-key support (F1–F12).
- **Window size probe**: terminal dimensions are probed at startup to eliminate resize flash on slow machines (Raspberry Pi).

### Map & Partner View Polish
- **Partner map centering**: map and legend now centered horizontally with `lipgloss.PlaceHorizontal`.
- **PSK map centering**: same centering applied to Heard/PSK pane.
- **Map width**: increased from 128→140 chars on large screens; uses full column width on partner page.
- **Inline photo**: properly positioned with asymmetric padding; cache respects `PictureAtQRZPane` toggle (no restart needed).
- **Kitty F2 viewer**: full-screen dimensions match content area; exit properly resets photo dimensions for inline view.

### Config Menu Redesign
- **Borderless menus**: all config choosers (logbook, rig, contest, operator, integration, notifications) use `menuBoxStyle` — no ANSI border escapes that corrupt Kitty graphics placement.
- **Viewport scrolling**: all menu list and edit views now use `bubbles/viewport` with auto-scroll that follows cursor focus.
- **PgUp/PgDown/Home/End** support in all viewport-backed menus.
- **Integration menu**: blank row between header and content for visual breathing room.

### Rig Power Handling
- **Power clamping**: rig power values are floored and clamped to the rig preset's configured maximum — a Xiegu G90 set to 20W will never display 21W due to firmware rounding.
- **WSJT-X power priority**: `txPowerForWSJTX` now directly sets the QSO power before `ApplyStationDefaults`, so the hamlib/flrig form value always wins over WSJT-X ADIF `tx_pwr` — fixes QSOs logged with 10W while the form showed 21W.

### Integration Lifecycle Fixes
- **HTTP server mid-run enable**: stale backoff timer is now reset when HTTP is disabled; server restarts when config is re-saved even if address/port haven't changed.
- **DXC mid-run enable**: `connecting` and `lastAttempt` state is fully reset on disable and internet loss — no more silent failures when toggling DXC on mid-session.
- **WSJT-X CQ transition**: form is cleared when the user starts calling CQ (DX call → empty + transmitting), removing the previous partner's data.

### Key Bindings & Navigation
- **Favorites**: Ctrl+V/B/N to recall favorites 1/2/3; Ctrl+Shift+V/B/N to save.
- **Rotor controls**: Alt+←/→/↑/↓ for azimuth and elevation (Alt only, no Ctrl required).
- **Pane navigation**: Ctrl+←/→ to switch between QSO form, Recent QSOs, and partner/map panes.
- **Comment retention**: Ctrl+K toggles keep-comment mode.
- **Form holding**: Ctrl+H toggles retain-form mode.
- **Tab shortcuts**: Alt+digit labels for Linux console compatibility.
- **Focusable item hints**: space-key indicator on toggles and buttons throughout menus.

### Wizard Cleanup
- **APRS section removed** from first-run wizard (callsign, passcode, TX beacon, interval, radius, symbol, comment, test button).
- **GPS Grid checkbox removed** from first-run wizard.
- Both remain fully available in the regular Settings → Station screen.

### Security & Safety
- **Single-instance guard**: file-lock prevents running two CQOps instances against the same config directory — protects SQLite from concurrent write corruption.
- **QRZ password sanitizing**: password is redacted in error log messages.

### Bug Fixes & Polish
- **Photo cache invalidation**: partner view cache now includes `PictureAtQRZPane` flag — toggling the setting mid-run no longer shows a stale empty column.
- **Toast simplification**: removed internal caching from ToastQueue; dedup window unchanged.
- **ADIF export**: bearing is validated before writing; contest exchange fields use standard ADIF keys.
- **Rig edit restart**: WSJT-X listener is immediately restarted when rig configuration changes, no app restart needed.
- **Rig preset duplication**: Ctrl+D in the rig chooser copies the selected preset.
- **Linux console**: `TERM=xterm-256color` is set as fallback for proper color and key support.
- **Config reset**: Ctrl+Alt+R with confirmation dialog resets configuration to defaults.
- **Cache reset**: Ctrl+Alt+C clears all render caches.
- **Terminal capability logging**: comprehensive environment diagnostics at startup for debugging Linux framebuffer console issues.

### Under the Hood
- **91 commits**, 92 files changed (10,189 insertions, 1,121 deletions).
- **ntcharts v2.2.0**: `picture.Model` and `pictureurl.Model` for Kitty graphics.
- **Dependencies bumped**: Bubble Tea v2, Bubbles v2, Lip Gloss v2, and all Charm ecosystem packages.
- **Build scripts**: `build.sh`/`build.ps1` use correct module path for ldflags version embedding.
- **All tests pass**: `go test ./...` — 34 TUI test files, comprehensive coverage for new GPS, APRS KISS, and power handling code.

## v0.8.9 — 2026-07-05

### CQOps Live — Built-in Browser Dashboard
- **Real-time web dashboard** with SSE push, Leaflet map, and live station display. Enable in F9 → Integrations, then open `http://localhost:8073` in any browser.
- **Live map** with QSO paths, active QSO tracking, partner photo display, day/night terminator overlay, and RainViewer weather radar.
- **Stats panel**: today's QSOs, unique calls, 5m/15m/60m rate tracking, top operators.
- **Recent QSOs table**: 7-row live feed with band/mode color badges, auto-scroll.
- **Band conditions module**: day/night propagation per band group (80–40m, 30–20m, 17–15m, 12–10m) from HamQSL solar data. Always renders full-width in the info box.
- **Solar & geomagnetic modules**: SFI, sunspots, A-index, K-index with color-coded condition thresholds.
- **DXC & PSK Reporter modules**: last spotted station, per-band report counts.
- **Weather row**: current conditions from Open-Meteo (temp, wind, humidity, icon) for the station's grid locator.
- **APRS integration**: nearby stations on the local map with standard APRS symbol icons, range circle, callsign popups, and auto-fit. Optional periodic position beacon with grid locator.
- **QRZ photos** displayed inline in the hero panel when available.
- **Responsive design**: FullHD+ optimized, breakpoints for small screens, narrow layouts, and short viewports. Works on Field Day projector displays.
- **Info box cycling**: modules rotate every 5 seconds, 1 or 2 columns depending on width.
- **Offline-safe**: all third-party services degrade gracefully; dashboard works with cached/local assets.

### ADIF 3.1.7 Compliance
- **FT8** is now exported as a standalone mode (not MFSK+FT8), per ADIF 3.1.7 spec.
- **FT4 and FT2** exported as MFSK with submode FT4/FT2.
- **Mode normalization**: `NormalizeMode` converts standalone FT4/FT2→MFSK+submode, and legacy MFSK+FT8→standalone FT8.
- **Submode display**: rig info and QSO form now include submode; dashboard shows submode via smart `submode||mode` fallback.

### Stats & Rate Calculation
- **Three-tier rate display**: 5-minute, 15-minute, and 1-hour rates replace the single `RatePerHour` field.
- **Rate query robustness**: uses `printf('%s%06s', qso_date, time_on)` for reliable time comparison, fixing off-by-window errors.
- **Stats fields** nowrap+ellipsis for clean overflow handling at any screen width.

### Dashboard UI Polish
- **21 band colors + 4 mode group colors** as CSS variables, used consistently across badges, pills, and table cells.
- **Premium styling**: border strength 0.22→0.35, shadow 0.07→0.12, badge backgrounds 0.08→0.22 for better visibility.
- **Consolidated breakpoints**: weather 8→4, height 4→2, width 3→2 for simpler maintenance.
- **UTC clock** now displays seconds (`23:26:23Z`).
- **Top QSOs** compact redesign: no trophy icons, no rank numbers, km without space, 9 items visible at FullHD+.

### Bug Fixes
- **SQLITE_BUSY on Wavelog status update**: `UpdateWavelogStatus` now retries 5 times with exponential backoff (100ms→1.6s), preventing "database is locked" errors from leaving the local status as "no" when the upload succeeded.
- **WSJT-X event channel overflow**: removed dead `Events` channel write that caused "dropping events" warnings every ~2.6k events. Channel kept initialized for external consumers.
- **HTTP server restart**: now only restarts when address, port, or enabled state changes — header/logo edits no longer trigger unnecessary restarts.
- **WSJT-X TX power**: added `>0` guard with `strconv.ParseFloat` to prevent zero-watt power from rig-in-RX state overwriting WSJT-X reported power.
- **Dashboard enrichment race**: `forcePushDashboardRecent` clears `lastRecentIDs` before pushing enriched QSOs, so country/grid updates from QRZ reach the browser immediately.
- **Top QSOs without grids**: removed `km>0` filter so QSOs without grid squares still appear in the top list.
- **Extra modules cycling**: `cycleExtraModule` now delegates to `updateExtraBox` (was calling itself inconsistently).

### Rebranding
- **New brand colors**: cyan `#08F8F8` and magenta `#F80868` replace the previous green palette.
- **App icon**: `$c` in cyan, `q` in magenta on a dark rounded background. Regenerated across all formats (PNG, XPM, ICO, .syso).
- **README overhaul**: architecture Mermaid diagram showing Station→CQOps→Internet/Dashboard/File I/O flow, platform badges, Quick Install section, tightened feature list, screenshot grouping.

### Refactoring & Cleanup
- **Dead code removal**: ASCII world map rendering, unused functions in `queries_qso.go`, `dxc_filter.go`, `operator_menu.go`, and `styles.go`.
- **Geo package**: coordinate conversion utilities moved from `map_ascii.go` to new `internal/tui/geo.go` with comprehensive tests.
- **8-char grid support**: latitude/longitude calculation now handles extended Maidenhead locators.
- **Duplicate QSO notification**: system beep on dupe detection (configurable via notifications menu).

## v0.8.8 — 2026-06-29

### Hamlib Rigctld — Robust Multi-Rig Support
- **VFO probe overhaul**: try `f VFOA` first to avoid blocking the serial mutex on backends that require VFO-prefixed commands (model 1042). Detect non-VFO backends by inspecting RPRT -1 suffix rejection. Probe timeout increased from 300ms to 2s for slow serial rigs.
- **Drain-before-RPRT fix**: `cmd()` now drains the character-mode repeat BEFORE checking RPRT errors. Previously an RPRT -11 on the `v` command skipped the drain, leaking stale data that poisoned all subsequent reads on the shared connection → permanent `freq=0`.
- **Frequency validation**: values ≤100 kHz (stale "USB", "RPRT 0", "0") now trigger an immediate connection drop instead of silently showing 0 Hz forever.
- **Power query**: non-fatal — no longer drops the shared TCP connection on failure. `powerVfoOK` flag remembers VFO-form rejection and skips retries. Backends that don't support `l VFOA RFPOWER` (model 1042) fall back silently.
- **Disconnected backoff**: polling interval increases from 1s to 10s when rigctld is unreachable, preventing rapid connect/drop cycles that flooded rigctld with TIME_WAIT connections.
- **Rig config menu**: selecting a different rig now immediately disconnects the old hamlib client and connects to the new rig's host:port (`needsRefresh` flag). Previously required exiting the menu first.

### DXC Cluster
- **Band sort on new spots**: cached sort band is reset when fresh spots arrive, so the active band filter re-sorts correctly instead of showing stale order.
- **Logbook switch**: cycling logbooks now auto-requests `SH/FDX 50` so the DXC table is never empty on a fresh logbook.

### QRZ & Wavelog Lookups
- **Completion-aware skip**: QRZ and Wavelog lookups now skip dispatch if already completed for the same call sign, eliminating redundant HTTP requests.
- **Mode normalization**: rig mode (USB/LSB) is normalized to canonical form (SSB) before storing as `wlLastMode`, preventing spurious "pending" state on the Partner screen.
- **Wavelog timeout**: dispatch time is now reset after timeout fires, preventing repeated timeout toasts for the same call.
- **Field navigation**: Wavelog data is only cleared when the normalized band or mode actually changes, not on every keystroke in the QSO form.

### PSK Reporter
- **Band marker colors**: migrated from ANSI 8-bit codes (9–15, rendered dull/grey on modern terminals) to the semantic RGB palette (Primary, Success, Warning, Accent, Info, Error) for clearly distinguishable band dots and legend labels.

### Band Plan
- **Markdown export** (`Ctrl+E` on F7): exports the full IARU Region band plan as `cqops_bandplan.md` in the config directory, with a `Generated by CQOps vX.Y.Z on YYYY-MM-DD` footer linking to cqops.com.
- **FT2 mode**: added to digital mode and spot keyword lists.

### Bug Fixes
- **Windows secrets test**: `TestSave_WritesWithCorrectPermissions` now skipped on Windows (Unix permission bits don't apply).
- **DXC spot fill**: `dxcFillFromSelected` only clears lookup state when the spot call differs from the current form call, preserving in-progress QRZ/Wavelog data.
- **Duplicate check**: mode is now normalized via `NormalizeRigMode` before querying, matching the stored format.

### Polishing
- Toast: always "Hamlib: connected" — the `--vfo` flag cannot be reliably detected from the protocol alone, and guessing wrong produced misleading warnings on both backends.

## v0.8.7 — 2026-06-28

### Encrypted Secrets Store
- **New `internal/secrets` package** — AES-256-GCM encrypted storage for passwords and API keys
- Secrets live in `~/.config/cqops/secrets.enc` (0600 permissions), never in plaintext `config.yaml`
- Key derived from `/etc/machine-id` (Linux) or hostname fallback — tied to the machine
- Auto-migration: plaintext secrets from existing configs migrate to encrypted store on first run
- Protected: QRZ password, DXC login, Wavelog API keys (per logbook)
- Graceful degradation: corruption or wrong-machine → app starts normally, warning toast shown, secrets re-enterable via UI
- Zero CPU overhead after startup: decrypted secrets cached in memory

### Paste Support
- Clipboard paste now works in the **wizard** (station form, rig form, QRZ credentials)
- Clipboard paste now works in the **logbook editor** (inline QSO editing — callsign, comment, notes, etc.)
- Clipboard paste now works in the **station editor** (logbook chooser → Wavelog section)
- All paste targets respect field formatting (uppercase for callsigns, locator normalization, etc.)

### Operator Editor Improvements
- Callsign auto-uppercased on every keystroke (matches StationForm behavior)
- Validation toast shown when leaving callsign field with non-standard value (no digit)
- Validation fires on Tab, Shift+Tab, Up, Down, paste, and save (Ctrl+S)

### Toast System Overhaul
- UTF-8 symbols replace text prefixes: ● (info), ✓ (success), ▲ (warning), ✗ (error)
- Symbols are geometric characters, not emoji — render correctly on B&W terminals
- All integration toasts now use `Integration: message` prefix format:
  - Solar, flrig, Hamlib, Internet, REF, Band Plan, Rig tune
  - QRZ/Wavelog errors, DXC spotted-by notifications

### Help Bar — Visible Key Bindings
- Ins (Create) and Del (Delete) now visible in the bottom bar for:
  - Rig config menu, logbook config menu, contest config menu, operator config menu
- Previously only accessible via the ? help overlay

### Bug Fixes (New)
- **Wavelog upload race**: Recent QSOs table now refreshes immediately after upload completes, no longer shows stale "not sent" status
- **Favorite recall**: frequency now trims trailing zeros (e.g. `14.250000` → `14.25`), matching ADIF export formatting
- **Config validation**: `EnsureConfig()` now applies encrypted secrets before validating, so the app starts correctly with secrets in `secrets.enc`

### Performance — ~70 optimizations across 5 rounds
- Render caches with signature-based invalidation: contest menu, PSK map, solar panel, help overlay, buildContestLine, helpSuffix
- `lipgloss.NewStyle()` eliminated from every hot path: root View() clip styles, DXC spacer/table wrappers, logbook editor dialogs/edit forms, confirm/spot dialog buttons, notifications menu, help overlay
- `fmt.Sprintf` replaced with `strings.Builder`+`strconv` in all cache keys: PSK Reporter, BPL views, logbook editor, QSO form path row, DXC filter info
- DXC: filter-aware spot cache with in-memory raw cache, pre-allocated query slices, `strconv.FormatFloat` for frequency format, `formatDXCSpotTime()` avoids `time.Format`
- PSK Reporter: async DB loading, cached spot map markers, table rowStyle caching
- BPL: precomputed line lists at startup, `bplFreqStr()`/`bplBwStr()` helpers using `strconv`
- RecentQSOs: pre-computed tier max widths at `init()`, O(1) tier lookup
- flrig: 5 goroutines → sequential XML-RPC calls (~10,800 fewer goroutine spawns per 3h session)
- Toast dedup (2s window), `Active()` dirty-flag cache, overlay content cache
- Other: invariant styles promoted to package-level vars, pre-compiled regexps, wizard formBox style cache, logbook download progress message cache

### Code Quality — ~30 fixes across 3 rounds
- Error handling: solar parse errors now logged, tune verify errors logged, import_validate errors include callsign context, WSJT-X event overflow warning
- Refactoring: 130-line lookup result switch extracted from `Update()` to `handleLookupResultMsg()`, shared `handleTuneResult()` for DXC/BPL tunes, `dxcCycleFilter()`/`dxcCycleFilterBack()` generic filter cycling, `clearQRZFields()` reused
- Default host/port constants in `config/`, deprecated `backend` field now warns, `FriendlyError` handles all HTTP codes
- Nil guard on `cycleActiveContest()`, WSJT-X toast nil guard

### Features
- Wider Recent QSOs table when solar panel active — shows Operator + WL columns on ≥166-col terminals
- `map.go` → `map_ascii.go` clarity rename

### Bug Fixes
- WSJT-X status dot now turns green immediately on connect (cache key missing `wsjtx.online`)
- DXC/BPL tune now works when WSJT-X is listening but not transmitting (`wsjtx.online` → `wsjtx.tx`)
- Rig connect toasts suppressed on reconnect loops (`vfoWarned` flag)
- Toast overlay no longer caches full composite (was hiding content on screen switch)
- `nfpm.yaml` fixed: removed invalid `glibc` depends, unnecessary `libsqlite3-0` recommends
- `build.ps1` fixed: removed invalid `GOARCH=armhf`

### Tests
- `store/migrations_test.go` — migration application + idempotency tests
- `internal/rotor/rotor_test.go` — `Status` zero-value test

### Packaging & Scripts
- `uninstall.sh` now matches install-specific PATH line instead of deleting any line containing "cqops"
- `installer/cqops.nsi` comment no longer hardcodes version
- Backup file `build/cqops.exe~` removed

## v0.8.6 — 2026-06-24

### Multi-Operator & Club Station Support
- Operator profiles in config (callsign + name), per-logbook active operator
- Ctrl+O hot-swap through configured operators, space-toggleable in station form
- Operator menu (create/edit/delete) with validation and all-logbook cascade
- WSJT-X auto-log preserves WSJT-X operator; warns on mismatch with active operator
- Wizard auto-creates operator entry from callsign during first-run setup

### Hamlib Rigctld Backend & Rotor Control
- Backend-agnostic rig architecture: flrig (HTTP) and hamlib rigctld (TCP) via shared `RigClient` interface
- Hamlib rigctld support: frequency, mode, VFO, split, power with graceful VFO name query fallback for Xiegu radios
- Hamlib rotctld rotor control backend with TUI integration (azimuth, elevation, stop)
- Per-rig rotor config in rig presets (hamlib host/port)
- VFO mode auto-detection for split-capable radios

### Windows Installer (NSIS)
- `installer/cqops.nsi` — Start Menu shortcuts, PATH integration, Control Panel uninstall entry, license page, solid LZMA compression
- `scripts/build-installer.ps1` — local build with auto `.ico` generation from `cqops.png` via ImageMagick
- Shortcut targets `.exe` directly; Windows Terminal shows embedded icon in tab/taskbar

### Linux Packages (nfpm)
- `nfpm.yaml` — deb, rpm, and archlinux (`pkg.tar.zst`) for amd64 + arm64
- `installer/cqops.desktop` — freedesktop entry with enriched keywords
- `scripts/build-packages.sh` — local cross-platform package build

### Embedded App Icon & Console Icon
- `winres/winres.json` — go-winres config with icon, manifest (DPI-aware, long-path, Win7+), version metadata
- `cmd/cqops/rsrc_windows_*.syso` — compiled Windows resources (icon + manifest)
- Runtime `setConsoleIcon()` via Win32 API — Windows Terminal tab shows CQOps icon

### Error Persistence
- Top-level `recover()` in `main.go` pauses on panic/startup failure so the terminal stays open

### WSJT-X Fixes
- `QsoLoggedMessage` (field-based) now constructs ADIF and saves — no more silently dropped QSOs
- WSJT-X auto-logged QSOs now inherit `ContestID` so they appear in RecentQSOs when a contest is active

### Bug Fixes (Audit)
- Fixed nil map panic when saving operator to uninitialized Operators map
- Fixed `config.Upgrade()` not stamping `State.Version` (was empty stub)
- Fixed invalid `DROP INDEX IF EXISTS` in SQLite migrations; added dedup DELETE before DXC UNIQUE index
- Fixed WSJT-X `unsafe.Pointer` usage with `recover()` and `Kind` check
- Fixed Wavelog `AllDuplicates` detection: iterates all Messages, defaults to false when empty
- Fixed DXC goroutine leak: `stopCh` checks in time.Sleep goroutines, exponential reconnect backoff
- Fixed `ListAllQSOs` OOM risk: internal pagination (500 per page)
- Fixed toast unbounded growth: capped at 20 items
- Fixed `--version` flag: prints version and exits without TUI
- Fixed default `Debug: true` → `false`
- Fixed logbook delete: synchronous `os.Remove` before toast
- Fixed double Wavelog lookup; added retry for QRZ lookups
- Fixed Maidenhead grid calculation (`LatLonToLocator` replaced with correct algorithm)
- Fixed PSK Reporter: per-callsign fetch timestamps with 5-minute cooldown across logbook cycles
- Fixed DXC: selected-row highlight spans full row; filter columns indicated via header
- Fixed DXC: show "DX Cluster not configured" toast when DXC is disabled (F4)
- Fixed bandplan export to match TUI data and formatting
- Fixed photo cache invalidation to reduce CPU usage during rendering
- Fixed photo loading state management in partner view

### Wavelog
- Chunked batch upload (50 QSOs per HTTP call) with individual fallback on duplicate errors
- Operator/grid mismatch detection during upload with normalize-and-retry flow

### Logging & Performance
- Size-based log rotation (10 MB) to prevent disk exhaustion

### CI & Build
- `.github/workflows/release.yml` — 3-job pipeline: build-unix (Go + nfpm), build-installer (Windows NSIS), publish (GitHub Release)
- `Makefile` — added `installer`, `packages`, `installer-all` targets
- `.gitignore` — added `dist/`

### Cleanup
- Removed dead code: `pendingSave`, `screenCON`, `handleCONUpdate`, `viewCON`, tabbar F3 CON, `lookupTimeoutMsg`, `openLogFile()`
- `if`/`else if` chains converted to tagged `switch` (wizard, rig_menu)
- Split inefficient `WriteString` concatenation in `bpl_views.go`
- Removed stale `flrig_integration.go` and `flrig_interface.go` (replaced by `rig_poll.go` + `rig_client.go`)
- Updated README with downloads badge, screenshots, and Unicode normalization package reference

## v0.8.5 — 2026-06-22 (First Public Release)

CQOps is a fast, minimal Go TUI ham radio logger built with Bubble Tea v2.
It targets normal desktops, Raspberry Pi-class hardware, and field/portable setups.

### Core Logging
- Full QSO form with callsign, band, frequency, mode/submode, RST, grid, name, QTH, country, comment, notes
- Automatic WSJT-X QSO logging via UDP ADIF with duplicate detection
- Manual QSO save with dupe check (press Enter twice to confirm)
- Logbook editor: view, edit, delete, filter, and export QSOs
- Recent QSOs table with automatic refresh on new/edit/delete
- Contest mode with exchange fields (STX/SRX), serial parsing, and contest filtering
- Retain comment toggle for quick repeated entries
- ADIF import/export with validation, Wavelog status tracking, and download recovery
- SOTA, POTA, WWFF, IOTA reference fields with auto-fill from REF database

### Integration Suite
- **Wavelog** — upload, private lookup (worked/confirmed status), full download/import
- **QRZ.com** — callbook lookup for name, QTH, grid, country, CQ/ITU zone, and photo
- **flrig** — frequency, mode, and split detection via XML-RPC
- **WSJT-X** — UDP listener, auto-log, TX status indicator, frequency/mode sync
- **DX Cluster** — telnet client with band/continent/mode/time filters, spot dialog, rig tuning
- **PSK Reporter** — spot table with band/time/mode filters and map view
- **Solar data** — hamqsl.com integration with solar flux, A-index, K-index display
- **REF database** — SOTA, POTA, WWFF, IOTA reference search and auto-fill
- **DXCC/CTY.DAT** — country/continent/CQ/ITU zone from callsign prefix
- **SCP** — Super Check Partial database for callsign completion

### TUI & UX
- Status bar with callsign, logbook, rig, operator, UTC/LT clock
- Integration status indicators (Net, WSJT, Rig, WL, QRZ, DXC) with green/red/amber dots
- Partner view with callbook data, logbook stats, azimuthal map, and photo
- Band plan browser (HAM, VHF/UHF, CB, PMR446, Broadcast) with markdown export
- Broadcast presets (BBC, VOA, etc.) with tune-to-frequency
- First-run wizard with station, rig, QRZ, Wavelog, and timezone setup
- Config screen for all integrations, notifications, and appearance
- Log viewer with scrollable text output
- Toast notification system with expiration
- Keyboard-driven navigation with help bar

### Technical
- Pure-Go SQLite via `modernc.org/sqlite` — no CGO, portable to any platform
- Bubble Tea v2 architecture with ~94 TUI source files, 34 test files
- Centralized style/theme system via Lip Gloss v2
- Render caching for expensive views (RecentQSOs, REF table, DXC filter-info, partner map)
- Cross-platform: Windows, Linux, macOS (amd64 + arm64)
- Graceful offline mode — all integrations fail safe when disabled or unreachable
- Structured logging with file rotation
- YAML config with multiple logbook support
- Version check against GitHub releases

### Performance
- Fast startup on Raspberry Pi-class hardware
- No blocking I/O in `View()` — all rendering is pure
- Cached table/map recomputation avoids allocation-heavy frames
- Network calls are async via `tea.Cmd`, never blocking updates

### License
Apache 2.0. See `LICENSE` and `licenses/` for third-party notices.
