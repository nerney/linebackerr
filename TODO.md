# Linebackerr Project Queue

## Working Rules
- All task runs should be executed by a subagent using the same config/model defaults as the main primary agent unless we explicitly decide otherwise.
- Only one **Linebackerr TODO** task may be running at a time.
- For this TODO queue, do not spawn work if **any other subagent is already running**.
- This concurrency limit applies only to the Linebackerr TODO queue. It does **not** block user-requested `dev` subagent work outside this queue.
- When a subagent starts work on a task, update that task status in this file.
- When a subagent finishes, record the result here and move the task to Completed.

## Active Tasks (In Progress)

## Backlog (Upcoming)

Let's make our matcher more robust!
All matching against input strings should be case insensitive. Spaces, dashes, periods, and other non-alphanumeric characters should be considered separators.
The pipeline should progressively extract fields and sometimes remove matched substrings before passing the transformed string to the next stage.
`MatchCandidate` should also retain the original unaltered input string.

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

- [ ] **matcher - add Validate() entrypoint on MatchCandidate**
  - Add the entrypoint:
    - `func (mc MatchCandidate) Validate() Match`
  - For now, keep this task focused on wiring the method/entrypoint shape and connecting it to a staged validation pipeline.
  - Add only minimal scaffolding/tests if needed; detailed stage behavior should be implemented in separate tasks below.

- [ ] **matcher validation - stage 1: exact GameDate + teams nflverse lookup**
  - If `GameDate` is set, search nflverse for games on that date.
  - If a game exists on that date with matching `HomeTeam` and `AwayTeam`, populate and return a `Match` with nflverse data.
  - Add a code comment / TODO noting future fallback handling if the exact date+teams lookup fails.
  - If not matched, continue to the next validation stage.

- [ ] **matcher validation - stage 2: regular season early failure rule**
  - If `GameType == RS` and `GameDate` was not set (the date stage already had its chance), fail validation here with an error.
  - Add focused tests for this failure path.

- [ ] **matcher validation - stage 3: Super Bowl roman numeral lookup**
  - If `GameType == SB` and `GameWeek` contains a Roman numeral, search nflverse using that signal.
  - If found, return a positive match.
  - Add focused tests around the roman numeral path.

- [ ] **matcher validation - stage 4: GameType + SeasonYear + teams nflverse lookup**
  - Search nflverse by `GameType`, `SeasonYear`, and participating teams.
  - For example: `CON`, `YYYY`, `HomeTeam`, `AwayTeam` should all match.
  - If required fields are missing at this point, return an error.
  - Add focused tests for successful and missing-field/error cases.

- [ ] **matcher validation - populate Match from nflverse result**
  - Centralize how a matched nflverse game populates the final `Match` object.
  - Reuse this population logic across validation stages.
  - Add focused tests for field population.

- [ ] **matcher validation - final pipeline return / unmatched handling**
  - If validation reaches the end without finding a match, return a `Match` with all `MatchCandidate` values copied over, `NflverseID` unset/zero, and `Error` populated.
  - The caller will distinguish success/failure from the returned `Match`.
  - Consider whether a derived `Matched` field/helper should exist, based on:
    - `err == nil && nflverse_id != 0`
  - Add focused tests for unmatched/error returns.

## Completed
- [x] **matcher - enrich team aliases from nflverse records and handle ambiguous city matches explicitly**
  - *Result:* Expanded the nflverse-driven alias inventory with practical alternate abbreviations/history markers (including `LAR`, `NWE`, `SDG`, `RAI`, `WSH`, etc.), kept ambiguous shared-city aliases such as `los angeles`/`la`/`new york` out of direct matching, surfaced them as an explicit `AmbiguousTeamAliasError` for downstream detection, added focused matcher tests for Rams/Chargers/Raiders/Washington coverage plus ambiguous city behavior, and `go test ./...` passes.
- [x] **matcher - implement home/away team extraction stage**
  - *Result:* Added non-mutating `extractTeamsStage` in `matcher/team_stage.go` that runs the working string through the existing team alias matcher twice, assigning first match to `AwayTeam` and second to `HomeTeam`; added focused tests for mascot aliases, abbreviations, historical aliases, and one-team failure cases; `go test ./...` passes.
- [x] **matcher - build nflverse-driven team alias inventory**
  - *Result:* Added a matcher team alias inventory keyed by current nflverse-style abbreviations, including practical historical/relocation aliases (for example STL/LA Rams, SD/LAC Chargers, OAK/LV Raiders, WFT/Commanders), documented ambiguity tradeoffs for intentionally skipped shortcuts like bare `la`/`los angeles`/`new york`, added longest-alias team matching scaffolding, and added representative alias tests; `go test ./...` passes.
- [x] **matcher - implement GameWeek extraction stage**
  - *Result:* Added `extractGameWeekStage` with numeric support for `week.#`, `week.##`, `w#`, `w##`, `wk.#`, and `wk.##`, plus valid Roman numeral extraction only for `GameType == SB`; matched week tokens are removed from the working string before downstream stages; added focused numeric and Super Bowl Roman numeral tests; `go test ./...` passes.
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
