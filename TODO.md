# Linebackerr Project Queue

## Working Rules
- All task runs should be executed by a subagent using the same config/model defaults as the main primary agent unless we explicitly decide otherwise.
- Only one TODO task may be running at a time. Never run concurrent tasks from this queue.
- When a subagent starts work on a task, update that task status in this file.
- When a subagent finishes, record the result here and move the task to Completed.

## Active Tasks (In Progress)
- [ ] **matcher - implement GameWeek extraction stage**
  - Gameplan:
    - support week/w/wk numeric patterns
    - support Super Bowl roman numeral extraction when appropriate
    - remove matched week token from working string
    - add numeric + roman numeral coverage
  - *Status:* Queued / promoted, not started

- [ ] **matcher - build nflverse-driven team alias inventory**
  - Gameplan:
    - build alias sets from nflverse names/abbreviations/history where practical
    - capture/document ambiguity tradeoffs
    - add representative alias tests
  - *Status:* Queued / promoted, not started

## Backlog (Upcoming)

Let's make our matcher more robust!
All matching against input strings should be case insensitive. Spaces, dashes, periods, and other non-alphanumeric characters should be considered separators.
The pipeline should progressively extract fields and sometimes remove matched substrings before passing the transformed string to the next stage.
`MatchCandidate` should also retain the original unaltered input string.

- [ ] **matcher - implement home/away team extraction stage**
  - Run the working string through the team matcher.
  - First matched team becomes `AwayTeam`, second becomes `HomeTeam`.
  - Do not mutate the working string in this stage.
  - Add tests for common patterns like `Patriots.at.Browns` and similar variants.

- [ ] **matcher - implement MatchCandidate extraction pipeline function**
  - Create a pipeline entrypoint that takes a single input string and returns a `MatchCandidate`.
  - Execute stages progressively in order:
    1. GameDate
    2. SeasonYear
    3. GameType
    4. GameWeek
    5. Away/Home team extraction
  - Preserve both original input and transformed intermediate behavior through tests.

- [ ] **matcher - expand array-based matcher flow to use pipeline**
  - Update the existing matcher package flow that processes arrays of release strings so it calls the new single-string pipeline function.
  - Add tests using real examples from `files.txt`.

- [ ] **matcher - add ValidateMatch on MatchCandidate**
  - Add only the skeleton for a `ValidateMatch` function/method that takes a `MatchCandidate` and returns a `Match`.
  - Do **not** implement the detailed validation/resolution logic yet.
  - This will eventually use nflverse extensively, but for this task we only want the shape/scaffolding.
  - Add minimal tests for the skeleton only.
  - We will break validation stages into separate tasks later.

## Completed
- [x] **matcher - implement GameType extraction stage**
  - *Result:* Added non-mutating `extractGameTypeStage` with case-insensitive mappings for `sb`/`super.bowl`/`superbowl` => `SB`, `conference`/`con`/`championship` => `CON`, `div`/`division`/`divisional` => `DIV`, `wc`/`wildcard`/`wild.card` => `WC`, and regular-season default to `RS`; added focused matcher tests for each mapping; `go test ./...` passes.
- [x] **matcher - implement SeasonYear extraction/derivation stage**
  - *Result:* Added `extractSeasonYearStage` to derive `SeasonYear` from `GameDate` (with Jan/Feb rolling back to the prior season), otherwise extract a standalone season-year token from the normalized working string and remove it for downstream stages; added focused matcher tests for date-derived and direct extraction cases, and `go test ./...` passes.
- [x] **matcher - implement GameDate extraction stage**
  - *Result:* Added an explicit `extractGameDateStage` that extracts `YYYY-MM-DD`, `YYYY.MM.DD`, `YYYY/MM/DD`, and `YYYYMMDD` after normalization, standardizes to `YYYY-MM-DD`, removes the matched date from the working string for downstream stages, added transformation-focused matcher tests, and `go test ./...` passes.
- [x] **matcher - add normalization helpers**
  - *Result:* Added explicit normalization/tokenization helpers for case-insensitive matching with non-alphanumeric separator collapsing, wired the existing matcher flow to use them, extended focused matcher tests for separator-heavy inputs and postseason/date extraction, and `go test ./...` passes.
- [x] **matcher - add OriginalInput field to MatchCandidate**
  - *Result:* Added `OriginalInput` to `matcher.MatchCandidate`, mirrored it on `matcher.Match` to keep resolved matches carrying all candidate fields, updated focused matcher tests, and `go test ./...` passes.
- [x] **matcher - create Match struct**
  - *Result:* Added `matcher.Match` with all `MatchCandidate` fields plus `NflverseID` and `Error`, added a focused matcher test for the new struct, and `go test ./...` passes.
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
