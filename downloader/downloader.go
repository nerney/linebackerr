package downloader

import (
	"context"
	"fmt"
	"linebackerr/prowlarr"
	"linebackerr/qbit"
	"linebackerr/sabnzbd"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ClientType string

const (
	ClientQBit    ClientType = "qbit"
	ClientSABnzbd ClientType = "sabnzbd"
)

type Status string

const (
	StatusQueued    Status = "queued"
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

type SubmitOptions struct {
	QBit    qbit.DownloadOptions
	SABnzbd sabnzbd.DownloadOptions
}

type Job struct {
	ID          string
	Client      ClientType
	Release     prowlarr.NFLRelease
	Name        string
	Progress    float64
	Status      Status
	Error       string
	StartedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
}

func (j Job) IsDone() bool {
	return j.Status == StatusCompleted || j.Status == StatusFailed
}

type Service struct {
	qbitClient *qbit.Client
	sabClient  *sabnzbd.Client

	mu   sync.RWMutex
	jobs map[string]Job
}

func NewService(qbitClient *qbit.Client, sabClient *sabnzbd.Client) *Service {
	return &Service{qbitClient: qbitClient, sabClient: sabClient, jobs: map[string]Job{}}
}

func NewServiceFromEnv() *Service {
	return NewService(qbit.NewClientFromEnv(), sabnzbd.NewClientFromEnv())
}

func (s *Service) Submit(ctx context.Context, client ClientType, releases []prowlarr.NFLRelease, options SubmitOptions) ([]Job, error) {
	switch client {
	case ClientQBit:
		return s.submitQBit(ctx, releases, options.QBit)
	case ClientSABnzbd:
		return s.submitSAB(ctx, releases, options.SABnzbd)
	default:
		return nil, fmt.Errorf("unsupported downloader client: %s", client)
	}
}

func (s *Service) ActiveJobs() []Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		if j.IsDone() {
			continue
		}
		out = append(out, j)
	}
	return out
}

func (s *Service) AllJobs() []Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		out = append(out, j)
	}
	return out
}

func (s *Service) Monitor(ctx context.Context) ([]Job, error) {
	active := s.ActiveJobs()
	if len(active) == 0 {
		return nil, nil
	}

	if err := s.refreshQBit(ctx, active); err != nil {
		return nil, err
	}
	if err := s.refreshSAB(ctx, active); err != nil {
		return nil, err
	}
	return s.ActiveJobs(), nil
}

func (s *Service) submitQBit(ctx context.Context, releases []prowlarr.NFLRelease, options qbit.DownloadOptions) ([]Job, error) {
	if s.qbitClient == nil {
		return nil, fmt.Errorf("qbit client is not configured")
	}
	jobs, err := s.qbitClient.DownloadFromProwlarrReleases(ctx, releases, options)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	stored := make([]Job, 0, len(jobs))
	for _, job := range jobs {
		status := StatusActive
		completedAt := (*time.Time)(nil)
		if qbit.IsTerminalState(job.Torrent.State, job.Torrent.Progress) {
			status = StatusCompleted
			t := now
			completedAt = &t
		}
		entry := Job{
			ID:          job.Torrent.Hash,
			Client:      ClientQBit,
			Release:     job.Release,
			Name:        job.Torrent.Name,
			Progress:    job.Torrent.Progress,
			Status:      status,
			StartedAt:   now,
			UpdatedAt:   now,
			CompletedAt: completedAt,
		}
		s.upsert(entry)
		stored = append(stored, entry)
	}
	return stored, nil
}

func (s *Service) submitSAB(ctx context.Context, releases []prowlarr.NFLRelease, options sabnzbd.DownloadOptions) ([]Job, error) {
	if s.sabClient == nil {
		return nil, fmt.Errorf("sabnzbd client is not configured")
	}
	jobs, err := s.sabClient.DownloadFromProwlarrReleases(ctx, releases, options)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	stored := make([]Job, 0, len(jobs))
	for _, job := range jobs {
		status := StatusActive
		errMsg := ""
		completedAt := (*time.Time)(nil)
		if job.History.NZOID != "" && sabnzbd.IsTerminalStatus(job.History.Status) {
			t := now
			completedAt = &t
			if sabnzbd.IsSuccessfulStatus(job.History.Status) {
				status = StatusCompleted
			} else {
				status = StatusFailed
				errMsg = strings.TrimSpace(job.History.FailMessage)
			}
		}
		entry := Job{
			ID:          job.NZOID,
			Client:      ClientSABnzbd,
			Release:     job.Release,
			Name:        choose(job.History.Name, job.Queue.Filename, job.Release.Raw.Title),
			Progress:    parsePercent(job.Queue.Percentage),
			Status:      status,
			Error:       errMsg,
			StartedAt:   now,
			UpdatedAt:   now,
			CompletedAt: completedAt,
		}
		s.upsert(entry)
		stored = append(stored, entry)
	}
	return stored, nil
}

func (s *Service) refreshQBit(ctx context.Context, active []Job) error {
	if s.qbitClient == nil {
		return nil
	}
	need := false
	for _, j := range active {
		if j.Client == ClientQBit {
			need = true
			break
		}
	}
	if !need {
		return nil
	}

	torrents, err := s.qbitClient.ListTorrents(ctx)
	if err != nil {
		return err
	}
	byHash := make(map[string]qbit.TorrentInfo, len(torrents))
	for _, t := range torrents {
		byHash[t.Hash] = t
	}

	now := time.Now()
	for _, j := range active {
		if j.Client != ClientQBit {
			continue
		}
		torrent, ok := byHash[j.ID]
		if !ok {
			continue
		}
		j.Name = choose(torrent.Name, j.Name)
		j.Progress = torrent.Progress
		j.UpdatedAt = now
		if qbit.IsTerminalState(torrent.State, torrent.Progress) {
			j.Status = StatusCompleted
			if j.CompletedAt == nil {
				t := now
				j.CompletedAt = &t
			}
		}
		s.upsert(j)
	}
	return nil
}

func (s *Service) refreshSAB(ctx context.Context, active []Job) error {
	if s.sabClient == nil {
		return nil
	}
	need := false
	for _, j := range active {
		if j.Client == ClientSABnzbd {
			need = true
			break
		}
	}
	if !need {
		return nil
	}

	queue, err := s.sabClient.ListQueue(ctx)
	if err != nil {
		return err
	}
	history, err := s.sabClient.ListHistory(ctx)
	if err != nil {
		return err
	}
	queueByID := map[string]sabnzbd.QueueSlot{}
	for _, slot := range queue {
		queueByID[slot.NZOID] = slot
	}
	historyByID := map[string]sabnzbd.HistorySlot{}
	for _, slot := range history {
		historyByID[slot.NZOID] = slot
	}

	now := time.Now()
	for _, j := range active {
		if j.Client != ClientSABnzbd {
			continue
		}
		if q, ok := queueByID[j.ID]; ok {
			j.Name = choose(q.Filename, j.Name)
			j.Progress = parsePercent(q.Percentage)
			j.Status = StatusActive
			j.UpdatedAt = now
		}
		if h, ok := historyByID[j.ID]; ok && sabnzbd.IsTerminalStatus(h.Status) {
			j.Name = choose(h.Name, j.Name)
			j.UpdatedAt = now
			t := now
			j.CompletedAt = &t
			if sabnzbd.IsSuccessfulStatus(h.Status) {
				j.Status = StatusCompleted
				j.Progress = 100
				j.Error = ""
			} else {
				j.Status = StatusFailed
				j.Error = strings.TrimSpace(h.FailMessage)
			}
		}
		s.upsert(j)
	}
	return nil
}

func (s *Service) upsert(job Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[key(job.Client, job.ID)] = job
}

func key(client ClientType, id string) string {
	return string(client) + ":" + id
}

func parsePercent(raw string) float64 {
	raw = strings.TrimSpace(strings.TrimSuffix(raw, "%"))
	if raw == "" {
		return 0
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return v
}

func choose(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
