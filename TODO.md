# Linebackerr Project Queue

## Working Rules
- All task runs should be executed by a subagent using the same config/model defaults as the main primary agent unless we explicitly decide otherwise.
- When a subagent starts work on a task, update that task status in this file.
- When a subagent finishes, record the result here and move the task to Completed.

## Active Tasks (In Progress)
- [ ] **matcher - create Match struct**
  - Gameplan:
    - define `Match` using all `MatchCandidate` fields
    - add nflverse ID field
    - add Error field
    - keep it staged only; do not start implementation until explicitly asked
  - *Status:* Queued / promoted, not started

## Backlog (Upcoming)

Let's make our matcher more robust! 
All matching against input strings should be case insensitive. spaces, dashes, periods, and other non-alphanumeric characters should be considered separators.
I will use . as the separator in my examples but it should work for all non-alphanumeric characters. 

- [ ] **matcher - create Match struct**
  - Match fields should be:
    - all fields from MatchCandidate
    - nflverse ID or 0 (when Error is not null)
    - Error - Error or null (when nflverse ID is not 0)
## Completed
- [x] **matcher - create MatchCandidate struct**
  - *Result:* Added `matcher.MatchCandidate` plus `matcher.GameType` string constants (`SB`, `CON`, `DIV`, `WC`, `RS`), kept existing matcher behavior unchanged, added a focused matcher test, and `go test ./...` passes.
- [x] **Add Unit Tests**
  - *Result:* Added package tests for `db`, `sportarr`, and `nflverse`, covering schema/init behavior plus mocked HTTP/CSV ingest paths. `go test ./...` passes.
- [x] **Implement 48-hour sync logic**
  - *Result:* Added `sync_state` table to DB. `main.go` now checks `db.NeedsSync(module)` before running `nflverse.Init()` or `sportarr.Load*()`.
- [x] **Breakout `sportarr` package**
  - *Result:* Created `linebackerr/sportarr`, moved API/scraping logic from `db.go`.
- [x] **Merge `nflverse` into `linebackerr.db`**
  - *Result:* Updated `nflverse.go` to accept shared `*sql.DB` connection.
- [x] **Prefix tables**
  - *Result:* Renamed tables to `sportarr_*` and `nflverse_*`. Removed old `nflverse.db` file.
