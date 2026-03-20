package search

import (
	"context"
	"errors"
	"fmt"
	"linebackerr/matcher"
	"linebackerr/prowlarr"
	"sort"
	"strings"
)

var (
	ErrMissingSearcher  = errors.New("missing prowlarr searcher")
	ErrMissingDownloader = errors.New("missing download handler")
	ErrUnresolvedMatch  = errors.New("match is unresolved")
	ErrNoReleasesFound  = errors.New("no matching releases found")
)

// ProwlarrSearcher describes the prowlarr search behavior used by this package.
type ProwlarrSearcher interface {
	SearchNFLReleases(query string, indexerIDs []int) ([]prowlarr.NFLRelease, error)
}

// DownloadFunc is invoked after releases are selected from Prowlarr results.
type DownloadFunc func(ctx context.Context, releases []prowlarr.NFLRelease) error

// Service coordinates the full search lifecycle from a resolved matcher.Match
// through Prowlarr querying and final download submission.
type Service struct {
	Searcher    ProwlarrSearcher
	Downloader  DownloadFunc
	IndexerIDs  []int
	MaxDownloads int
}

// LifecycleResult captures the key lifecycle details from search to download.
type LifecycleResult struct {
	Match            matcher.Match
	QueriesAttempted []string
	ReleasesFound    []prowlarr.NFLRelease
	SelectedReleases []prowlarr.NFLRelease
}

// NewService builds a search lifecycle service.
func NewService(searcher ProwlarrSearcher, downloader DownloadFunc, indexerIDs []int) *Service {
	copied := append([]int(nil), indexerIDs...)
	return &Service{
		Searcher:    searcher,
		Downloader:  downloader,
		IndexerIDs:  copied,
		MaxDownloads: 1,
	}
}

// SearchAndDownload resolves candidate release queries for a match, submits them
// to Prowlarr, filters + ranks results, then passes selected releases to the
// configured downloader handler.
func (s *Service) SearchAndDownload(ctx context.Context, match matcher.Match) (LifecycleResult, error) {
	if s == nil || s.Searcher == nil {
		return LifecycleResult{}, ErrMissingSearcher
	}
	if s.Downloader == nil {
		return LifecycleResult{}, ErrMissingDownloader
	}
	if !match.Matched() {
		return LifecycleResult{}, ErrUnresolvedMatch
	}

	queries := BuildQueries(match)
	foundByGUID := map[string]prowlarr.NFLRelease{}
	queriesAttempted := make([]string, 0, len(queries))

	for _, query := range queries {
		queriesAttempted = append(queriesAttempted, query)
		releases, err := s.Searcher.SearchNFLReleases(query, append([]int(nil), s.IndexerIDs...))
		if err != nil {
			return LifecycleResult{}, err
		}

		for _, release := range releases {
			if !releaseMatchesTarget(match, release) {
				continue
			}
			key := strings.TrimSpace(release.Raw.GUID)
			if key == "" {
				key = strings.TrimSpace(release.Raw.Title)
			}
			if key == "" {
				continue
			}
			if _, exists := foundByGUID[key]; !exists {
				foundByGUID[key] = release
			}
		}
	}

	found := mapToSortedReleases(foundByGUID)
	if len(found) == 0 {
		return LifecycleResult{
			Match:            match,
			QueriesAttempted: queriesAttempted,
			ReleasesFound:    nil,
			SelectedReleases: nil,
		}, ErrNoReleasesFound
	}

	maxDownloads := s.MaxDownloads
	if maxDownloads <= 0 {
		maxDownloads = 1
	}
	if maxDownloads > len(found) {
		maxDownloads = len(found)
	}

	selected := found[:maxDownloads]
	if err := s.Downloader(ctx, selected); err != nil {
		return LifecycleResult{}, err
	}

	return LifecycleResult{
		Match:            match,
		QueriesAttempted: queriesAttempted,
		ReleasesFound:    found,
		SelectedReleases: selected,
	}, nil
}

// BuildQueries returns prioritized search strings for a resolved match.
// More specific queries are attempted first, then broader fallbacks.
func BuildQueries(match matcher.Match) []string {
	candidates := []string{
		strings.Join(compactParts("nfl", match.GameDate, match.AwayTeam, "vs", match.HomeTeam), " "),
		strings.Join(compactParts("nfl", match.SeasonYear, gameTypeSearchLabel(match.GameType), "week", match.GameWeek, match.AwayTeam, match.HomeTeam), " "),
		strings.Join(compactParts("nfl", match.SeasonYear, match.AwayTeam, match.HomeTeam), " "),
		strings.Join(compactParts("nfl", match.AwayTeam, match.HomeTeam), " "),
		"nfl",
	}

	seen := map[string]struct{}{}
	queries := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(strings.Join(strings.Fields(strings.ToLower(candidate)), " "))
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		queries = append(queries, candidate)
	}

	return queries
}

func releaseMatchesTarget(match matcher.Match, release prowlarr.NFLRelease) bool {
	if !release.Match.Matched() {
		return false
	}

	if strings.TrimSpace(match.NflverseID) != "" && strings.TrimSpace(release.Match.NflverseID) != strings.TrimSpace(match.NflverseID) {
		return false
	}

	if match.GameDate != "" && release.Match.GameDate != "" && match.GameDate != release.Match.GameDate {
		return false
	}

	return true
}

func mapToSortedReleases(foundByGUID map[string]prowlarr.NFLRelease) []prowlarr.NFLRelease {
	keys := make([]string, 0, len(foundByGUID))
	for key := range foundByGUID {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	releases := make([]prowlarr.NFLRelease, 0, len(keys))
	for _, key := range keys {
		releases = append(releases, foundByGUID[key])
	}
	return releases
}

func compactParts(parts ...string) []string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		filtered = append(filtered, part)
	}
	return filtered
}

func gameTypeSearchLabel(gameType matcher.GameType) string {
	switch gameType {
	case matcher.GameTypeSuperBowl:
		return "super bowl"
	case matcher.GameTypeConference:
		return "conference championship"
	case matcher.GameTypeDivisional:
		return "divisional"
	case matcher.GameTypeWildcard:
		return "wildcard"
	default:
		return ""
	}
}

func (r LifecycleResult) String() string {
	return fmt.Sprintf("queries=%d releases=%d selected=%d", len(r.QueriesAttempted), len(r.ReleasesFound), len(r.SelectedReleases))
}
