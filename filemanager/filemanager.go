package filemanager

import (
	"context"
	"errors"
	"fmt"
	"linebackerr/matcher"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var sanitizeRegex = regexp.MustCompile(`[^a-zA-Z0-9._ -]+`)

type Manager struct {
	libraryRoot string
}

type Placement struct {
	AbsolutePath string
	RelativePath string
}

type UpdateResult struct {
	OriginalPath string
	UpdatedPath  string
	RelativePath string
	Updated      bool
}

type ReconcileEntry struct {
	Match       matcher.Match
	CurrentPath string
}

type MonitorOptions struct {
	Interval time.Duration
	Entries  func(context.Context) ([]ReconcileEntry, error)
	OnCycle  func([]UpdateResult, error)
}

func New(libraryRoot string) (*Manager, error) {
	root := strings.TrimSpace(libraryRoot)
	if root == "" {
		return nil, errors.New("library root is required")
	}
	return &Manager{libraryRoot: filepath.Clean(root)}, nil
}

func (m *Manager) LibraryRoot() string {
	if m == nil {
		return ""
	}
	return m.libraryRoot
}

func (m *Manager) PrepareImportTarget(match matcher.Match, sourcePath string) (Placement, error) {
	if m == nil {
		return Placement{}, errors.New("nil filemanager")
	}
	sourcePath = filepath.Clean(strings.TrimSpace(sourcePath))
	if sourcePath == "" {
		return Placement{}, errors.New("source path is required")
	}

	expected, err := m.ExpectedPath(match, sourcePath)
	if err != nil {
		return Placement{}, err
	}
	available, err := nextAvailablePath(expected)
	if err != nil {
		return Placement{}, err
	}
	rel, err := filepath.Rel(m.libraryRoot, available)
	if err != nil {
		return Placement{}, fmt.Errorf("compute destination relative path: %w", err)
	}

	return Placement{AbsolutePath: available, RelativePath: rel}, nil
}

func (m *Manager) ExpectedPath(match matcher.Match, sourcePath string) (string, error) {
	rel, err := m.ExpectedRelativePath(match, sourcePath)
	if err != nil {
		return "", err
	}
	return filepath.Join(m.libraryRoot, rel), nil
}

func (m *Manager) ExpectedRelativePath(match matcher.Match, sourcePath string) (string, error) {
	if m == nil {
		return "", errors.New("nil filemanager")
	}
	sourcePath = filepath.Clean(strings.TrimSpace(sourcePath))
	if sourcePath == "" {
		return "", errors.New("source path is required")
	}

	ext := strings.TrimSpace(filepath.Ext(sourcePath))
	if ext == "" {
		ext = ".mkv"
	}

	fileBase := sanitizeFilename(BuildMediaBaseName(match))
	if fileBase == "" {
		fileBase = sanitizeFilename(strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath)))
	}
	if fileBase == "" {
		fileBase = "NFL Game"
	}

	return filepath.Join(SeasonFolder(match), fileBase+ext), nil
}

func (m *Manager) EnsureFileConsistency(match matcher.Match, currentPath string) (UpdateResult, error) {
	if m == nil {
		return UpdateResult{}, errors.New("nil filemanager")
	}
	currentPath = filepath.Clean(strings.TrimSpace(currentPath))
	if currentPath == "" {
		return UpdateResult{}, errors.New("current path is required")
	}
	if _, err := os.Stat(currentPath); err != nil {
		return UpdateResult{}, fmt.Errorf("stat current file: %w", err)
	}

	expected, err := m.ExpectedPath(match, currentPath)
	if err != nil {
		return UpdateResult{}, err
	}
	rel, err := filepath.Rel(m.libraryRoot, expected)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("compute expected relative path: %w", err)
	}

	if samePath(currentPath, expected) {
		return UpdateResult{OriginalPath: currentPath, UpdatedPath: currentPath, RelativePath: rel, Updated: false}, nil
	}

	target, err := nextAvailablePath(expected)
	if err != nil {
		return UpdateResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return UpdateResult{}, fmt.Errorf("create directories for normalized path: %w", err)
	}
	if err := os.Rename(currentPath, target); err != nil {
		return UpdateResult{}, fmt.Errorf("move file to normalized path: %w", err)
	}

	targetRel, err := filepath.Rel(m.libraryRoot, target)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("compute updated relative path: %w", err)
	}
	return UpdateResult{OriginalPath: currentPath, UpdatedPath: target, RelativePath: targetRel, Updated: true}, nil
}

func (m *Manager) ReconcileLibrary(ctx context.Context, entries []ReconcileEntry) ([]UpdateResult, error) {
	results := make([]UpdateResult, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return results, err
		}
		result, err := m.EnsureFileConsistency(entry.Match, entry.CurrentPath)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (m *Manager) StartMonitor(ctx context.Context, options MonitorOptions) error {
	if m == nil {
		return errors.New("nil filemanager")
	}
	if options.Entries == nil {
		return errors.New("monitor entries provider is required")
	}
	interval := options.Interval
	if interval <= 0 {
		interval = time.Minute
	}

	runCycle := func() error {
		entries, err := options.Entries(ctx)
		if err != nil {
			if options.OnCycle != nil {
				options.OnCycle(nil, err)
			}
			return err
		}
		results, runErr := m.ReconcileLibrary(ctx, entries)
		if options.OnCycle != nil {
			options.OnCycle(results, runErr)
		}
		return runErr
	}

	if err := runCycle(); err != nil && ctx.Err() == nil {
		return err
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := runCycle(); err != nil && ctx.Err() == nil {
				return err
			}
		}
	}
}

func SeasonFolder(match matcher.Match) string {
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

	return filepath.Join("NFL", "Season "+season, stageFolder)
}

func BuildMediaBaseName(match matcher.Match) string {
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

func samePath(first, second string) bool {
	return filepath.Clean(first) == filepath.Clean(second)
}
