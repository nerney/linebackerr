package filemanager

import (
	"context"
	"linebackerr/matcher"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpectedRelativePathRegularSeasonNaming(t *testing.T) {
	t.Parallel()

	manager, err := New(t.TempDir())
	require.NoError(t, err)

	rel, err := manager.ExpectedRelativePath(matcher.Match{
		GameType:   matcher.GameTypeRegularSeason,
		GameDate:   "2024-09-08",
		SeasonYear: "2024",
		AwayTeam:   "BUF",
		HomeTeam:   "NE",
		GameWeek:   "1",
	}, "/downloads/game-file.mkv")
	require.NoError(t, err)

	normalized := filepath.ToSlash(rel)
	assert.True(t, strings.HasPrefix(normalized, "NFL/Season 2024/Regular Season/"), "path prefix")
	assert.Contains(t, normalized, "NFL - 2024-09-08 - BUF NE - Week 1.mkv")
}

func TestPrepareImportTargetHandlesCollisions(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager, err := New(root)
	require.NoError(t, err)

	match := matcher.Match{GameType: matcher.GameTypeConference, SeasonYear: "2023", AwayTeam: "KC", HomeTeam: "BAL", GameWeek: "20"}
	first, err := manager.ExpectedPath(match, "first.ts")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(first), 0o755))
	require.NoError(t, os.WriteFile(first, []byte("existing"), 0o644))

	placement, err := manager.PrepareImportTarget(match, "fresh.ts")
	require.NoError(t, err)
	assert.NotEqual(t, first, placement.AbsolutePath)
	assert.Contains(t, filepath.Base(placement.AbsolutePath), "(1)")
}

func TestEnsureFileConsistencyRenamesFileToCanonicalPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager, err := New(root)
	require.NoError(t, err)

	oldPath := filepath.Join(root, "misc", "random-name.mkv")
	require.NoError(t, os.MkdirAll(filepath.Dir(oldPath), 0o755))
	require.NoError(t, os.WriteFile(oldPath, []byte("video"), 0o644))

	result, err := manager.EnsureFileConsistency(matcher.Match{
		GameType:   matcher.GameTypeRegularSeason,
		GameDate:   "2024-10-14",
		SeasonYear: "2024",
		AwayTeam:   "NYJ",
		HomeTeam:   "BUF",
	}, oldPath)
	require.NoError(t, err)
	assert.True(t, result.Updated)
	_, statErr := os.Stat(oldPath)
	assert.True(t, os.IsNotExist(statErr), "expected old path removed")
	_, statErr = os.Stat(result.UpdatedPath)
	require.NoError(t, statErr)
	assert.True(t, strings.HasPrefix(filepath.ToSlash(result.RelativePath), "NFL/Season 2024/Regular Season/"))
}

func TestStartMonitorRunsCycle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager, err := New(root)
	require.NoError(t, err)

	path := filepath.Join(root, "tmp", "game.mkv")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("video"), 0o644))

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
	assert.Greater(t, called, 0)
}

func TestBuildMediaBaseNameAndSeasonHelpers(t *testing.T) {
	t.Parallel()

	base := BuildMediaBaseName(matcher.Match{
		GameType:   matcher.GameTypeSuperBowl,
		GameDate:   "2025-02-09",
		SeasonYear: "2024",
		AwayTeam:   "KC",
		HomeTeam:   "SF",
		GameWeek:   "SB",
	})
	assert.Equal(t, "NFL - 2025-02-09 - KC @ SF - Super Bowl - Week SB", base)

	assert.Equal(t, "2023", deriveSeasonYear("2024-01-07"))
	assert.Equal(t, "2024", deriveSeasonYear("2024-10-14"))
	assert.Equal(t, "", deriveSeasonYear("not-a-date"))

	assert.Equal(t, "bad name.mkv", sanitizeFilename("bad///name.mkv"))
	assert.Equal(t, "", sanitizeFilename("***"))
}
