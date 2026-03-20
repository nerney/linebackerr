package importer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"linebackerr/matcher"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

type Mode string

const (
	ModeMove     Mode = "move"
	ModeHardlink Mode = "hardlink"
	ModeSymlink  Mode = "symlink"
	ModeCopy     Mode = "copy"
)

type Options struct {
	LibraryRoot string
	Mode        Mode
}

type ImportResult struct {
	SourcePath      string
	DestinationPath string
	RelativePath    string
	Mode            Mode
	AlreadyPresent  bool
}

type Service struct {
	libraryRoot string
	mode        Mode
}

var sanitizeRegex = regexp.MustCompile(`[^a-zA-Z0-9._ -]+`)

func NewService(options Options) (*Service, error) {
	root := strings.TrimSpace(options.LibraryRoot)
	if root == "" {
		return nil, errors.New("library root is required")
	}

	mode := options.Mode
	if mode == "" {
		mode = ModeMove
	}
	switch mode {
	case ModeMove, ModeHardlink, ModeSymlink, ModeCopy:
	default:
		return nil, fmt.Errorf("unsupported importer mode: %s", mode)
	}

	return &Service{libraryRoot: filepath.Clean(root), mode: mode}, nil
}

func (s *Service) ImportFile(ctx context.Context, match matcher.Match, sourcePath string) (ImportResult, error) {
	if s == nil {
		return ImportResult{}, errors.New("nil importer service")
	}
	if err := ctx.Err(); err != nil {
		return ImportResult{}, err
	}

	sourcePath = filepath.Clean(strings.TrimSpace(sourcePath))
	if sourcePath == "" {
		return ImportResult{}, errors.New("source path is required")
	}
	if _, err := os.Stat(sourcePath); err != nil {
		return ImportResult{}, fmt.Errorf("stat source file: %w", err)
	}

	destinationPath, relativePath, err := s.destinationPath(match, sourcePath)
	if err != nil {
		return ImportResult{}, err
	}

	if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
		return ImportResult{}, fmt.Errorf("create library directories: %w", err)
	}

	if sameFile(sourcePath, destinationPath) {
		return ImportResult{
			SourcePath:      sourcePath,
			DestinationPath: destinationPath,
			RelativePath:    relativePath,
			Mode:            s.mode,
			AlreadyPresent:  true,
		}, nil
	}

	if err := s.placeFile(sourcePath, destinationPath); err != nil {
		return ImportResult{}, err
	}

	return ImportResult{
		SourcePath:      sourcePath,
		DestinationPath: destinationPath,
		RelativePath:    relativePath,
		Mode:            s.mode,
	}, nil
}

func (s *Service) ImportFiles(ctx context.Context, match matcher.Match, sourcePaths []string) ([]ImportResult, error) {
	results := make([]ImportResult, 0, len(sourcePaths))
	for _, sourcePath := range sourcePaths {
		if err := ctx.Err(); err != nil {
			return results, err
		}
		result, err := s.ImportFile(ctx, match, sourcePath)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *Service) destinationPath(match matcher.Match, sourcePath string) (string, string, error) {
	season := strings.TrimSpace(match.SeasonYear)
	if season == "" {
		season = deriveSeasonYear(match.GameDate)
	}
	if season == "" {
		season = "Unknown"
	}

	stageFolder := "Regular Season"
	if match.GameType != matcher.GameTypeRegularSeason {
		stageFolder = "Postseason"
	}

	libraryDir := filepath.Join(s.libraryRoot, "NFL", "Season "+season, stageFolder)

	ext := strings.TrimSpace(filepath.Ext(sourcePath))
	if ext == "" {
		ext = ".mkv"
	}

	fileBase := sanitizeFilename(buildMediaBaseName(match))
	if fileBase == "" {
		fileBase = sanitizeFilename(strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath)))
	}
	if fileBase == "" {
		fileBase = "NFL Game"
	}

	targetPath := filepath.Join(libraryDir, fileBase+ext)
	targetPath, err := nextAvailablePath(targetPath)
	if err != nil {
		return "", "", err
	}

	rel, err := filepath.Rel(s.libraryRoot, targetPath)
	if err != nil {
		return "", "", fmt.Errorf("compute destination relative path: %w", err)
	}
	return targetPath, rel, nil
}

func (s *Service) placeFile(sourcePath, destinationPath string) error {
	switch s.mode {
	case ModeMove:
		if err := os.Rename(sourcePath, destinationPath); err == nil {
			return nil
		} else {
			if err := copyFile(sourcePath, destinationPath); err != nil {
				return err
			}
			if err := os.Remove(sourcePath); err != nil {
				return fmt.Errorf("remove source after move fallback: %w", err)
			}
			return nil
		}
	case ModeHardlink:
		if err := os.Link(sourcePath, destinationPath); err != nil {
			return fmt.Errorf("create hardlink: %w", err)
		}
		return nil
	case ModeSymlink:
		if err := os.Symlink(sourcePath, destinationPath); err != nil {
			return fmt.Errorf("create symlink: %w", err)
		}
		return nil
	case ModeCopy:
		if err := copyFile(sourcePath, destinationPath); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported importer mode: %s", s.mode)
	}
}

func buildMediaBaseName(match matcher.Match) string {
	dateToken := strings.TrimSpace(match.GameDate)
	if dateToken == "" {
		dateToken = strings.TrimSpace(match.SeasonYear)
	}
	if dateToken == "" {
		dateToken = "unknown-date"
	}

	away := strings.TrimSpace(match.AwayTeam)
	home := strings.TrimSpace(match.HomeTeam)
	if away == "" || home == "" {
		if strings.TrimSpace(match.OriginalInput) != "" {
			return fmt.Sprintf("NFL - %s - %s", dateToken, strings.TrimSpace(match.OriginalInput))
		}
		return fmt.Sprintf("NFL - %s", dateToken)
	}

	suffixParts := make([]string, 0, 2)
	if label := gameTypeLabel(match.GameType); label != "" {
		suffixParts = append(suffixParts, label)
	}
	if week := strings.TrimSpace(match.GameWeek); week != "" {
		suffixParts = append(suffixParts, "Week "+week)
	}

	title := fmt.Sprintf("NFL - %s - %s @ %s", dateToken, away, home)
	if len(suffixParts) > 0 {
		title += " - " + strings.Join(suffixParts, " - ")
	}
	return title
}

func gameTypeLabel(gameType matcher.GameType) string {
	switch gameType {
	case matcher.GameTypeSuperBowl:
		return "Super Bowl"
	case matcher.GameTypeConference:
		return "Conference Championship"
	case matcher.GameTypeDivisional:
		return "Divisional"
	case matcher.GameTypeWildcard:
		return "Wildcard"
	default:
		return ""
	}
}

func deriveSeasonYear(gameDate string) string {
	parts := strings.Split(strings.TrimSpace(gameDate), "-")
	if len(parts) < 1 {
		return ""
	}
	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return ""
	}
	month := 9
	if len(parts) > 1 {
		if m, err := strconv.Atoi(parts[1]); err == nil {
			month = m
		}
	}
	if month <= 2 {
		year--
	}
	return strconv.Itoa(year)
}

func sanitizeFilename(value string) string {
	value = sanitizeRegex.ReplaceAllString(value, " ")
	value = strings.TrimSpace(strings.Join(strings.Fields(value), " "))
	return value
}

func nextAvailablePath(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return path, nil
		}
		return "", fmt.Errorf("stat destination file: %w", err)
	}

	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(filepath.Base(path), ext)

	for i := 1; i < 1000; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		} else if err != nil {
			return "", fmt.Errorf("stat destination candidate: %w", err)
		}
	}
	return "", fmt.Errorf("could not allocate destination path for %s", path)
}

func sameFile(sourcePath, destinationPath string) bool {
	src, err := os.Stat(sourcePath)
	if err != nil {
		return false
	}
	dst, err := os.Stat(destinationPath)
	if err != nil {
		return false
	}
	return os.SameFile(src, dst)
}

func copyFile(sourcePath, destinationPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}
	defer source.Close()

	info, err := source.Stat()
	if err != nil {
		return fmt.Errorf("stat source file: %w", err)
	}

	destination, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("open destination file: %w", err)
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return fmt.Errorf("copy file content: %w", err)
	}

	if runtime.GOOS != "windows" {
		_ = os.Chmod(destinationPath, info.Mode().Perm())
	}

	return nil
}
