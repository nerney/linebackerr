package prowlarr

import (
	"encoding/json"
	"fmt"
	"linebackerr/matcher"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	defaultBaseURL = "http://localhost:9696"
)

var nflTokenRegex = regexp.MustCompile(`(?i)(^|[^a-z0-9])nfl([^a-z0-9]|$)`)

// Client manages connectivity and API calls to a Prowlarr instance.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// SearchResult is the subset of raw Prowlarr result fields we care about.
type SearchResult struct {
	Title       string `json:"title"`
	GUID        string `json:"guid"`
	IndexerID   int    `json:"indexerId"`
	Indexer     string `json:"indexer"`
	Size        int64  `json:"size"`
	PublishDate string `json:"publishDate"`
	DownloadURL string `json:"downloadUrl"`
}

// NFLRelease maps a Prowlarr search result into Linebackerr matcher structures.
type NFLRelease struct {
	Raw       SearchResult
	Candidate matcher.MatchCandidate
	Match     matcher.Match
}

// NewClient creates a Prowlarr client with explicit settings.
func NewClient(baseURL, apiKey string, httpClient *http.Client) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		APIKey:     strings.TrimSpace(apiKey),
		HTTPClient: httpClient,
	}
}

// NewClientFromEnv creates a Prowlarr client from environment variables:
// PROWLARR_URL and PROWLARR_API_KEY.
func NewClientFromEnv() *Client {
	return NewClient(os.Getenv("PROWLARR_URL"), os.Getenv("PROWLARR_API_KEY"), nil)
}

// SearchNFLReleases queries Prowlarr search and maps NFL-like results into
// Linebackerr's internal matcher structures.
func (c *Client) SearchNFLReleases(query string, indexerIDs []int) ([]NFLRelease, error) {
	if c == nil {
		return nil, fmt.Errorf("nil prowlarr client")
	}
	if c.APIKey == "" {
		return nil, fmt.Errorf("missing prowlarr api key")
	}

	endpoint, err := url.Parse(c.BaseURL + "/api/v1/search")
	if err != nil {
		return nil, fmt.Errorf("build prowlarr search url: %w", err)
	}

	params := endpoint.Query()
	params.Set("query", query)
	for _, id := range indexerIDs {
		params.Add("indexerIds", strconv.Itoa(id))
	}
	endpoint.RawQuery = params.Encode()

	req, err := http.NewRequest(http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create prowlarr request: %w", err)
	}
	req.Header.Set("X-Api-Key", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform prowlarr request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prowlarr search failed: status %d", resp.StatusCode)
	}

	var raw []SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode prowlarr response: %w", err)
	}

	mapped := make([]NFLRelease, 0, len(raw))
	for _, item := range raw {
		if !isNFLReleaseTitle(item.Title) {
			continue
		}

		candidate := matcher.Pipeline(item.Title)
		match := candidate.Validate()
		mapped = append(mapped, NFLRelease{
			Raw:       item,
			Candidate: candidate,
			Match:     match,
		})
	}

	return mapped, nil
}

func isNFLReleaseTitle(title string) bool {
	return nflTokenRegex.MatchString(strings.ToLower(title))
}
