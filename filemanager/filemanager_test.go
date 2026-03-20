package filemanager

import (
	"context"
	"linebackerr/matcher"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExpectedRelativePathRegularSeasonNaming(t *testing.T) {
	t.Parallel()

	manager, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	rel, err := manager.ExpectedRelativePath(matcher.Match{
		GameType:   matcher.GameTypeRegularSeason,
		GameDate:   "2024-09-08",
		SeasonYear: "2024",
		AwayTeam:   "BUF",
		HomeTeam:   "NE",
		GameWeek:   "1",
	}, "/downloads/game-file.mkv")
	if err != nil {
		t.Fatalf("ExpectedRelativePath error: %v", err)
	}

	normalized := filepath.ToSlash(rel)
	if !strings.HasPrefix(normalized, "NFL/Season 2024/Regular Season/") {
		t.Fatalf("unexpected path prefix: %s", normalized)
	}
	if !strings.Contains(normalized, "NFL - 2024-09-08 - BUF NE - Week 1.mkv") {
		t.Fatalf("unexpected file name in path: %s", normalized)
	}
}

func TestPrepareImportTargetHandlesCollisions(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager, err := New(root)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	match := matcher.Match{GameType: matcher.GameTypeConference, SeasonYear: "2023", AwayTeam: "KC", HomeTeam: "BAL", GameWeek: "20"}
	first, err := manager.ExpectedPath(match, "first.ts")
	if err != nil {
		t.Fatalf("ExpectedPath error: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(first), 0o755); err != nil {
		t.Fatalf("mkdir error: %v", err)
	}
	if err := os.WriteFile(first, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	placement, err := manager.PrepareImportTarget(match, "fresh.ts")
	if err != nil {
		t.Fatalf("PrepareImportTarget error: %v", err)
	}
	if placement.AbsolutePath == first {
		t.Fatalf("expected a non-colliding path, got %s", placement.AbsolutePath)
	}
	if !strings.Contains(filepath.Base(placement.AbsolutePath), "(1)") {
		t.Fatalf("expected collision suffix, got %s", placement.AbsolutePath)
	}
}

func TestEnsureFileConsistencyRenamesFileToCanonicalPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager, err := New(root)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	oldPath := filepath.Join(root, "misc", "random-name.mkv")
	if err := os.MkdirAll(filepath.Dir(oldPath), 0o755); err != nil {
		t.Fatalf("mkdir error: %v", err)
	}
	if err := os.WriteFile(oldPath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write old file: %v", err)
	}

	result, err := manager.EnsureFileConsistency(matcher.Match{
		GameType:   matcher.GameTypeRegularSeason,
		GameDate:   "2024-10-14",
		SeasonYear: "2024",
		AwayTeam:   "NYJ",
		HomeTeam:   "BUF",
	}, oldPath)
	if err != nil {
		t.Fatalf("EnsureFileConsistency error: %v", err)
	}
	if !result.Updated {
		t.Fatalf("expected file to be moved")
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected old path to be removed, err=%v", err)
	}
	if _, err := os.Stat(result.UpdatedPath); err != nil {
		t.Fatalf("expected updated path to exist: %v", err)
	}
	if !strings.HasPrefix(filepath.ToSlash(result.RelativePath), "NFL/Season 2024/Regular Season/") {
		t.Fatalf("unexpected relative path: %s", result.RelativePath)
	}
}

func TestStartMonitorRunsCycle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager, err := New(root)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	path := filepath.Join(root, "tmp", "game.mkv")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir error: %v", err)
	}
	if err := os.WriteFile(path, []byte("video"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	called := 0
	go func() {
		_ = manager.StartMonitor(ctx, MonitorOptions{
			Interval: 5 * time.Millisecond,
			Entries: func(context.Context) ([]ReconcileEntry, error) {
				return []ReconcileEntry{{
					Match:       matcher.Match{GameType: matcher.GameTypeRegularSeason, GameDate: "2024-10-14", SeasonYear: "2024", AwayTeam: "NYJ", HomeTeam: "BUF"},
					CurrentPath: path,
				}}, nil
			},
			OnCycle: func(_ []UpdateResult, _ error) {
				called++
				cancel()
			},
		})
	}()

	time.Sleep(30 * time.Millisecond)
	if called == 0 {
		t.Fatalf("expected monitor cycle callback to be called")
	}
}
