# Linebackerr Project Queue

## Active Tasks (In Progress)
- [ ] **Add Unit Tests**
  - Add test files alongside each non-main package (`db_test.go`, `sportarr_test.go`, `nflverse_test.go`).
  - Write tests targeting maximum coverage (mocking DB/HTTP where appropriate).
  - *Status:* Pending

## Backlog (Upcoming)
- [ ] *Add future tasks here...*
- [ ] *e.g., Wire up Plex integration*
- [ ] *e.g., Build HTTP API endpoints*

## Completed
- [x] **Implement 48-hour sync logic**
  - *Result:* Added `sync_state` table to DB. `main.go` now checks `db.NeedsSync(module)` before running `nflverse.Init()` or `sportarr.Load*()`.
- [x] **Breakout `sportarr` package**
  - *Result:* Created `linebackerr/sportarr`, moved API/scraping logic from `db.go`.
- [x] **Merge `nflverse` into `linebackerr.db`**
  - *Result:* Updated `nflverse.go` to accept shared `*sql.DB` connection.
- [x] **Prefix tables**
  - *Result:* Renamed tables to `sportarr_*` and `nflverse_*`. Removed old `nflverse.db` file.
