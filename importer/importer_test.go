package importer

import (
	"context"
	"linebackerr/matcher"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportFileMovePlacesInMediaLibraryStructure(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	libraryDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "raw-game-file.mkv")
	if err := os.WriteFile(sourcePath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	svc, err := NewService(Options{LibraryRoot: libraryDir, Mode: ModeMove})
	if err != nil {
		t.Fatalf("NewService error: %v", err)
	}

	match := matcher.Match{
		GameType:   matcher.GameTypeRegularSeason,
		GameDate:   "2024-09-08",
		SeasonYear: "2024",
		AwayTeam:   "BUF",
		HomeTeam:   "NE",
		GameWeek:   "1",
	}

	result, err := svc.ImportFile(context.Background(), match, sourcePath)
	if err != nil {
		t.Fatalf("ImportFile error: %v", err)
	}

	if result.Mode != ModeMove {
		t.Fatalf("result mode = %s, want %s", result.Mode, ModeMove)
	}
	if result.RelativePath == "" {
		t.Fatalf("relative path should be set")
	}
	if _, err := os.Stat(result.DestinationPath); err != nil {
		t.Fatalf("expected destination file to exist: %v", err)
	}
	if _, err := os.Stat(sourcePath); !os.IsNotExist(err) {
		t.Fatalf("expected source file to be moved, err=%v", err)
	}

	normalized := filepath.ToSlash(result.RelativePath)
	if !strings.HasPrefix(normalized, "NFL/Season 2024/Regular Season/") {
		t.Fatalf("relative path = %q, expected NFL season structure", normalized)
	}
}

func TestImportFileHardlinkKeepsSourceAndCreatesLinkedDestination(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	libraryDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "postseason.ts")
	if err := os.WriteFile(sourcePath, []byte("clip-data"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	svc, err := NewService(Options{LibraryRoot: libraryDir, Mode: ModeHardlink})
	if err != nil {
		t.Fatalf("NewService error: %v", err)
	}

	match := matcher.Match{
		GameType:   matcher.GameTypeConference,
		SeasonYear: "2023",
		AwayTeam:   "KC",
		HomeTeam:   "BAL",
		GameWeek:   "20",
	}

	result, err := svc.ImportFile(context.Background(), match, sourcePath)
	if err != nil {
		t.Fatalf("ImportFile error: %v", err)
	}

	if _, err := os.Stat(sourcePath); err != nil {
		t.Fatalf("expected source file to remain for hardlink mode: %v", err)
	}
	if _, err := os.Stat(result.DestinationPath); err != nil {
		t.Fatalf("expected destination file to exist: %v", err)
	}

	srcInfo, err := os.Stat(sourcePath)
	if err != nil {
		t.Fatalf("stat source file: %v", err)
	}
	dstInfo, err := os.Stat(result.DestinationPath)
	if err != nil {
		t.Fatalf("stat destination file: %v", err)
	}
	if !os.SameFile(srcInfo, dstInfo) {
		t.Fatalf("expected hardlink destination to reference same file")
	}

	normalized := filepath.ToSlash(result.RelativePath)
	if !strings.HasPrefix(normalized, "NFL/Season 2023/Postseason/") {
		t.Fatalf("relative path = %q, expected postseason structure", normalized)
	}
}

func TestImportFileSymlinkCreatesSymlinkDestination(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	libraryDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "sb.mkv")
	if err := os.WriteFile(sourcePath, []byte("sb-data"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	svc, err := NewService(Options{LibraryRoot: libraryDir, Mode: ModeSymlink})
	if err != nil {
		t.Fatalf("NewService error: %v", err)
	}

	result, err := svc.ImportFile(context.Background(), matcher.Match{
		GameType:   matcher.GameTypeSuperBowl,
		SeasonYear: "2023",
		AwayTeam:   "SF",
		HomeTeam:   "KC",
		GameWeek:   "LVIII",
	}, sourcePath)
	if err != nil {
		t.Fatalf("ImportFile error: %v", err)
	}

	linkTarget, err := os.Readlink(result.DestinationPath)
	if err != nil {
		t.Fatalf("expected destination to be symlink: %v", err)
	}
	if linkTarget != sourcePath {
		t.Fatalf("symlink target = %q, want %q", linkTarget, sourcePath)
	}
}
