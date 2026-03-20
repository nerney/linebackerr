package importer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"linebackerr/matcher"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService_DefaultModeAndValidation(t *testing.T) {
	t.Run("defaults to move mode", func(t *testing.T) {
		svc, err := NewService(Options{LibraryRoot: t.TempDir()})
		require.NoError(t, err)
		require.NotNil(t, svc)
		assert.Equal(t, ModeMove, svc.mode)
	})

	t.Run("rejects unsupported mode", func(t *testing.T) {
		svc, err := NewService(Options{LibraryRoot: t.TempDir(), Mode: Mode("teleport")})
		require.Error(t, err)
		assert.Nil(t, svc)
	})
}

func TestImportFile_InputValidationErrors(t *testing.T) {
	t.Run("nil service", func(t *testing.T) {
		_, err := (*Service)(nil).ImportFile(context.Background(), matcher.Match{}, "x")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil importer service")
	})

	t.Run("context canceled", func(t *testing.T) {
		svc, err := NewService(Options{LibraryRoot: t.TempDir(), Mode: ModeCopy})
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err = svc.ImportFile(ctx, matcher.Match{}, "whatever")
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("missing source path", func(t *testing.T) {
		svc, err := NewService(Options{LibraryRoot: t.TempDir(), Mode: ModeCopy})
		require.NoError(t, err)

		_, err = svc.ImportFile(context.Background(), matcher.Match{}, "/definitely/not/a/real/source/file.mkv")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "stat source file")
	})
}

func TestImportFiles_StopsOnFirstErrorAndReturnsPartialResults(t *testing.T) {
	library := t.TempDir()
	sourceDir := t.TempDir()
	svc, err := NewService(Options{LibraryRoot: library, Mode: ModeCopy})
	require.NoError(t, err)

	existing := filepath.Join(sourceDir, "one.mkv")
	require.NoError(t, os.WriteFile(existing, []byte("one"), 0o644))
	missing := filepath.Join(sourceDir, "missing.mkv")

	results, err := svc.ImportFiles(context.Background(), matcher.Match{
		GameType:   matcher.GameTypeRegularSeason,
		GameDate:   "2024-09-08",
		SeasonYear: "2024",
		AwayTeam:   "BUF",
		HomeTeam:   "NE",
		GameWeek:   "1",
	}, []string{existing, missing})

	require.Error(t, err)
	require.Len(t, results, 1)
	assert.FileExists(t, results[0].DestinationPath)
}

func TestImportFile_CopyModeCopiesAndKeepsSource(t *testing.T) {
	sourceDir := t.TempDir()
	libraryDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "game.ts")
	require.NoError(t, os.WriteFile(sourcePath, []byte("payload"), 0o600))

	svc, err := NewService(Options{LibraryRoot: libraryDir, Mode: ModeCopy})
	require.NoError(t, err)

	result, err := svc.ImportFile(context.Background(), matcher.Match{
		GameType:   matcher.GameTypeRegularSeason,
		GameDate:   "2024-11-03",
		SeasonYear: "2024",
		AwayTeam:   "KC",
		HomeTeam:   "BUF",
		GameWeek:   "9",
	}, sourcePath)
	require.NoError(t, err)

	assert.FileExists(t, sourcePath)
	assert.FileExists(t, result.DestinationPath)
	src, err := os.ReadFile(sourcePath)
	require.NoError(t, err)
	dst, err := os.ReadFile(result.DestinationPath)
	require.NoError(t, err)
	assert.Equal(t, src, dst)
}
