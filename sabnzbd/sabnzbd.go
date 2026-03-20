package sabnzbd

import (
	"context"
	"encoding/json"
	"fmt"
	"linebackerr/prowlarr"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "http://localhost:8081"

var completedStatuses = map[string]bool{
	"completed": true,
	"completed/quick": true,
	"failed":    true,
	"deleted":   true,
}

// Client manages connectivity and API calls to SABnzbd.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// DownloadOptions controls how NZBs are added and observed.
type DownloadOptions struct {
	Category          string
	Priority          int
	Paused            bool
	PollInterval      time.Duration
	DetectTimeout     time.Duration
	CompletionTimeout time.Duration
	WaitForCompletion bool
}

// QueueSlot is the subset of SABnzbd queue fields used by Linebackerr.
type QueueSlot struct {
	NZOID      string `json:"nzo_id"`
	Filename   string `json:"filename"`
	Status     string `json:"status"`
	Category   string `json:"cat"`
	Percentage string `json:"percentage"`
	TimeLeft   string `json:"timeleft"`
}

// HistorySlot is the subset of SABnzbd history fields used by Linebackerr.
type HistorySlot struct {
	NZOID       string `json:"nzo_id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	FailMessage string `json:"fail_message"`
	Category    string `json:"category"`
	Completed   string `json:"completed"`
}

// DownloadJob tracks a submitted download and observed lifecycle state.
type DownloadJob struct {
	Release      prowlarr.NFLRelease
	NZOID        string
	Queue        QueueSlot
	History      HistorySlot
	IsSuccessful bool
}

// NewClient creates a SABnzbd client with explicit settings.
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

// NewClientFromEnv creates a SABnzbd client from environment variables:
// SABNZBD_URL and SABNZBD_API_KEY.
func NewClientFromEnv() *Client {
	return NewClient(
		os.Getenv("SABNZBD_URL"),
		os.Getenv("SABNZBD_API_KEY"),
		nil,
	)
}

// DownloadFromProwlarrReleases submits all valid Prowlarr release URLs to SABnzbd.
// If WaitForCompletion is true, each job is polled until it appears in history with a final status.
func (c *Client) DownloadFromProwlarrReleases(ctx context.Context, releases []prowlarr.NFLRelease, options DownloadOptions) ([]DownloadJob, error) {
	if c == nil {
		return nil, fmt.Errorf("nil sabnzbd client")
	}
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, fmt.Errorf("missing sabnzbd api key")
	}

	// Verify API connectivity before attempting submissions.
	if _, err := c.listQueue(ctx); err != nil {
		return nil, err
	}

	jobs := make([]DownloadJob, 0, len(releases))
	for _, release := range releases {
		downloadURL := strings.TrimSpace(release.Raw.DownloadURL)
		if downloadURL == "" {
			continue
		}

		nzoID, err := c.addURL(ctx, downloadURL, options)
		if err != nil {
			return nil, err
		}

		job := DownloadJob{
			Release: release,
			NZOID:   nzoID,
		}

		if queue, err := c.waitForQueueEntry(ctx, nzoID, options); err == nil {
			job.Queue = queue
		}

		if options.WaitForCompletion {
			history, err := c.waitForCompletion(ctx, nzoID, options)
			if err != nil {
				return nil, err
			}
			job.History = history
			job.IsSuccessful = strings.EqualFold(history.Status, "completed") || strings.EqualFold(history.Status, "completed/quick")
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (c *Client) addURL(ctx context.Context, downloadURL string, options DownloadOptions) (string, error) {
	params := url.Values{}
	params.Set("mode", "addurl")
	params.Set("name", downloadURL)
	params.Set("apikey", c.APIKey)
	params.Set("output", "json")
	if options.Category != "" {
		params.Set("cat", options.Category)
	}
	if options.Priority != 0 {
		params.Set("priority", strconv.Itoa(options.Priority))
	}
	if options.Paused {
		params.Set("pp", "-1")
	}

	endpoint := c.BaseURL + "/api?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create sabnzbd addurl request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("perform sabnzbd addurl request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("sabnzbd addurl failed: status %d", resp.StatusCode)
	}

	var addResp struct {
		Status bool     `json:"status"`
		NZOIDs []string `json:"nzo_ids"`
		Error  string   `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		return "", fmt.Errorf("decode sabnzbd addurl response: %w", err)
	}
	if !addResp.Status {
		if addResp.Error != "" {
			return "", fmt.Errorf("sabnzbd addurl failed: %s", addResp.Error)
		}
		return "", fmt.Errorf("sabnzbd addurl failed")
	}
	if len(addResp.NZOIDs) == 0 || strings.TrimSpace(addResp.NZOIDs[0]) == "" {
		return "", fmt.Errorf("sabnzbd addurl response missing nzo_id")
	}

	return strings.TrimSpace(addResp.NZOIDs[0]), nil
}

func (c *Client) waitForQueueEntry(ctx context.Context, nzoID string, options DownloadOptions) (QueueSlot, error) {
	timeout := options.DetectTimeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	poll := options.PollInterval
	if poll <= 0 {
		poll = 2 * time.Second
	}

	deadline := time.Now().Add(timeout)
	for {
		slots, err := c.listQueue(ctx)
		if err != nil {
			return QueueSlot{}, err
		}
		for _, slot := range slots {
			if slot.NZOID == nzoID {
				return slot, nil
			}
		}

		if time.Now().After(deadline) {
			return QueueSlot{}, fmt.Errorf("timed out waiting for sabnzbd queue entry: %s", nzoID)
		}

		select {
		case <-ctx.Done():
			return QueueSlot{}, ctx.Err()
		case <-time.After(poll):
		}
	}
}

func (c *Client) waitForCompletion(ctx context.Context, nzoID string, options DownloadOptions) (HistorySlot, error) {
	timeout := options.CompletionTimeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	poll := options.PollInterval
	if poll <= 0 {
		poll = 2 * time.Second
	}

	deadline := time.Now().Add(timeout)
	for {
		slots, err := c.listHistory(ctx)
		if err != nil {
			return HistorySlot{}, err
		}
		for _, slot := range slots {
			if slot.NZOID != nzoID {
				continue
			}
			if completedStatuses[strings.ToLower(slot.Status)] {
				return slot, nil
			}
		}

		if time.Now().After(deadline) {
			return HistorySlot{}, fmt.Errorf("timed out waiting for sabnzbd completion: %s", nzoID)
		}

		select {
		case <-ctx.Done():
			return HistorySlot{}, ctx.Err()
		case <-time.After(poll):
		}
	}
}

func (c *Client) listQueue(ctx context.Context) ([]QueueSlot, error) {
	params := url.Values{}
	params.Set("mode", "queue")
	params.Set("apikey", c.APIKey)
	params.Set("output", "json")

	endpoint := c.BaseURL + "/api?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create sabnzbd queue request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform sabnzbd queue request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sabnzbd queue failed: status %d", resp.StatusCode)
	}

	var queueResp struct {
		Queue struct {
			Slots []QueueSlot `json:"slots"`
		} `json:"queue"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&queueResp); err != nil {
		return nil, fmt.Errorf("decode sabnzbd queue response: %w", err)
	}

	return queueResp.Queue.Slots, nil
}

func (c *Client) listHistory(ctx context.Context) ([]HistorySlot, error) {
	params := url.Values{}
	params.Set("mode", "history")
	params.Set("apikey", c.APIKey)
	params.Set("output", "json")

	endpoint := c.BaseURL + "/api?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create sabnzbd history request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform sabnzbd history request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sabnzbd history failed: status %d", resp.StatusCode)
	}

	var historyResp struct {
		History struct {
			Slots []HistorySlot `json:"slots"`
		} `json:"history"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&historyResp); err != nil {
		return nil, fmt.Errorf("decode sabnzbd history response: %w", err)
	}

	return historyResp.History.Slots, nil
}
