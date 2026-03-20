package qbit

import (
	"context"
	"encoding/json"
	"fmt"
	"linebackerr/prowlarr"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

const defaultBaseURL = "http://localhost:8080"

var completedStates = map[string]bool{
	"uploading": true,
	"stalledUP": true,
	"pausedUP":  true,
	"queuedUP":  true,
	"forcedUP":  true,
	"checkingUP": true,
}

// Client manages connectivity and API calls to qBittorrent.
type Client struct {
	BaseURL    string
	Username   string
	Password   string
	HTTPClient *http.Client
}

// DownloadOptions controls how torrents are added and observed.
type DownloadOptions struct {
	SavePath          string
	Category          string
	Tags              []string
	Paused            bool
	SkipChecking      bool
	AutoTMM           bool
	PollInterval      time.Duration
	DetectTimeout     time.Duration
	CompletionTimeout time.Duration
	WaitForCompletion bool
}

// TorrentInfo is the subset of qBittorrent torrent fields used by Linebackerr.
type TorrentInfo struct {
	Hash     string  `json:"hash"`
	Name     string  `json:"name"`
	State    string  `json:"state"`
	Progress float64 `json:"progress"`
	Category string  `json:"category"`
	SavePath string  `json:"save_path"`
}

// DownloadJob tracks a submitted download and the final observed state.
type DownloadJob struct {
	Release prowlarr.NFLRelease
	Torrent TorrentInfo
}

// NewClient creates a qBittorrent client with explicit settings.
func NewClient(baseURL, username, password string, httpClient *http.Client) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	if httpClient == nil {
		jar, _ := cookiejar.New(nil)
		httpClient = &http.Client{Jar: jar}
	}

	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Username:   strings.TrimSpace(username),
		Password:   strings.TrimSpace(password),
		HTTPClient: httpClient,
	}
}

// NewClientFromEnv creates a qBittorrent client from environment variables:
// QBIT_URL, QBIT_USERNAME, and QBIT_PASSWORD.
func NewClientFromEnv() *Client {
	return NewClient(
		os.Getenv("QBIT_URL"),
		os.Getenv("QBIT_USERNAME"),
		os.Getenv("QBIT_PASSWORD"),
		nil,
	)
}

// Login authenticates against qBittorrent and stores the session cookie in the client's cookie jar.
func (c *Client) Login(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("nil qbit client")
	}
	if c.Username == "" || c.Password == "" {
		return fmt.Errorf("missing qbit credentials")
	}

	form := url.Values{}
	form.Set("username", c.Username)
	form.Set("password", c.Password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v2/auth/login", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("create qbit login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("perform qbit login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qbit login failed: status %d", resp.StatusCode)
	}

	return nil
}

// DownloadFromProwlarrReleases submits all valid Prowlarr release download URLs to qBittorrent.
// If WaitForCompletion is true, each torrent is polled until it enters a completed state.
func (c *Client) DownloadFromProwlarrReleases(ctx context.Context, releases []prowlarr.NFLRelease, options DownloadOptions) ([]DownloadJob, error) {
	if c == nil {
		return nil, fmt.Errorf("nil qbit client")
	}
	if err := c.Login(ctx); err != nil {
		return nil, err
	}

	torrents, err := c.listTorrents(ctx)
	if err != nil {
		return nil, err
	}
	knownHashes := make(map[string]struct{}, len(torrents))
	for _, t := range torrents {
		knownHashes[t.Hash] = struct{}{}
	}

	jobs := make([]DownloadJob, 0, len(releases))
	for _, release := range releases {
		if strings.TrimSpace(release.Raw.DownloadURL) == "" {
			continue
		}

		if err := c.addTorrentURL(ctx, release.Raw.DownloadURL, options); err != nil {
			return nil, err
		}

		torrent, err := c.waitForNewTorrent(ctx, knownHashes, options)
		if err != nil {
			return nil, err
		}
		knownHashes[torrent.Hash] = struct{}{}

		if options.WaitForCompletion {
			torrent, err = c.waitForCompletion(ctx, torrent.Hash, options)
			if err != nil {
				return nil, err
			}
		}

		jobs = append(jobs, DownloadJob{
			Release: release,
			Torrent: torrent,
		})
	}

	return jobs, nil
}

func (c *Client) addTorrentURL(ctx context.Context, downloadURL string, options DownloadOptions) error {
	form := url.Values{}
	form.Set("urls", strings.TrimSpace(downloadURL))
	if options.SavePath != "" {
		form.Set("savepath", options.SavePath)
	}
	if options.Category != "" {
		form.Set("category", options.Category)
	}
	if len(options.Tags) > 0 {
		form.Set("tags", strings.Join(options.Tags, ","))
	}
	form.Set("paused", boolString(options.Paused))
	form.Set("skip_checking", boolString(options.SkipChecking))
	form.Set("autoTMM", boolString(options.AutoTMM))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v2/torrents/add", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("create qbit add torrent request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("perform qbit add torrent request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qbit add torrent failed: status %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) listTorrents(ctx context.Context) ([]TorrentInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/v2/torrents/info", nil)
	if err != nil {
		return nil, fmt.Errorf("create qbit list torrents request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform qbit list torrents request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qbit list torrents failed: status %d", resp.StatusCode)
	}

	var torrents []TorrentInfo
	if err := json.NewDecoder(resp.Body).Decode(&torrents); err != nil {
		return nil, fmt.Errorf("decode qbit torrents info: %w", err)
	}

	sort.SliceStable(torrents, func(i, j int) bool { return torrents[i].Hash < torrents[j].Hash })
	return torrents, nil
}

func (c *Client) waitForNewTorrent(ctx context.Context, knownHashes map[string]struct{}, options DownloadOptions) (TorrentInfo, error) {
	timeout := options.DetectTimeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	poll := options.PollInterval
	if poll <= 0 {
		poll = 2 * time.Second
	}

	deadline := time.Now().Add(timeout)
	for {
		torrents, err := c.listTorrents(ctx)
		if err != nil {
			return TorrentInfo{}, err
		}
		for _, torrent := range torrents {
			if _, ok := knownHashes[torrent.Hash]; !ok {
				return torrent, nil
			}
		}

		if time.Now().After(deadline) {
			return TorrentInfo{}, fmt.Errorf("timed out waiting for newly added torrent")
		}

		select {
		case <-ctx.Done():
			return TorrentInfo{}, ctx.Err()
		case <-time.After(poll):
		}
	}
}

func (c *Client) waitForCompletion(ctx context.Context, hash string, options DownloadOptions) (TorrentInfo, error) {
	timeout := options.CompletionTimeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	poll := options.PollInterval
	if poll <= 0 {
		poll = 2 * time.Second
	}

	deadline := time.Now().Add(timeout)
	for {
		torrents, err := c.listTorrents(ctx)
		if err != nil {
			return TorrentInfo{}, err
		}
		for _, torrent := range torrents {
			if torrent.Hash != hash {
				continue
			}
			if completedStates[torrent.State] || torrent.Progress >= 1 {
				return torrent, nil
			}
		}

		if time.Now().After(deadline) {
			return TorrentInfo{}, fmt.Errorf("timed out waiting for torrent completion: %s", hash)
		}

		select {
		case <-ctx.Done():
			return TorrentInfo{}, ctx.Err()
		case <-time.After(poll):
		}
	}
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
