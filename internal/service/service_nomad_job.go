package service

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/alexisvisco/koyebtests/internal/types"
	"github.com/google/uuid"
	"github.com/hashicorp/nomad/api"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type NomadJobService struct {
	client *api.Client
	host   string
	logger *slog.Logger

	rwMutex     sync.RWMutex
	jobIdToPort map[string]int
}

func NewNomadJobService(host string, client *api.Client) *NomadJobService {
	return &NomadJobService{
		client:      client,
		logger:      slog.With("component", "nomad"),
		host:        host,
		jobIdToPort: make(map[string]int),
	}
}

func (s *NomadJobService) GetJobPort(jobID string) (int, bool) {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()

	port, exists := s.jobIdToPort[jobID]
	return port, exists
}

func (s *NomadJobService) CreateJob(name string, targetURL string, isScript bool) (*types.CreateJobOutput, error) {
	jobID := fmt.Sprintf(slugify(name) + "%s", uuid.New().String())

	job := s.createNomadJobSpec(jobID, targetURL, isScript)

	_, err := s.submitJob(job)
	if err != nil {
		return nil, fmt.Errorf("failed to submit job: %w", err)
	}

	_, port, err := s.waitForServiceURL(jobID)
	if err != nil {
		_ = s.PurgeJob(jobID)
		return nil, fmt.Errorf("job submitted but failed to get service URL: %w", err)
	}

	s.logger.Info("Job created successfully", "job_id", jobID, "port", port)

	s.rwMutex.Lock()
	s.jobIdToPort[jobID] = port
	s.rwMutex.Unlock()

	return &types.CreateJobOutput{
		URL: fmt.Sprintf("http://%s.%s", jobID, s.host),
	}, nil
}

func (s *NomadJobService) createNomadJobSpec(jobID, targetURL string, isScript bool) *api.Job {
	job := api.NewServiceJob(jobID, jobID, "global", 1)
	job.Datacenters = []string{"dc1"}

	group := api.NewTaskGroup("web", 1)

	task := api.NewTask("koyeb-nginx", "docker")
	task.Config = map[string]interface{}{
		"image": "alexisvisco/koyeb-nginx",
		"port_map": []map[string]int{
			{"http": 80},
		},
	}

	task.Env = map[string]string{
		"URL":       targetURL,
		"IS_SCRIPT": strconv.FormatBool(isScript),
	}

	task.Resources = &api.Resources{
		CPU:      toPtr[int](100), // 100 MHz
		MemoryMB: toPtr[int](128), // 128 MB
		Networks: []*api.NetworkResource{
			{
				MBits: toPtr[int](10), // 10 Mbits
				DynamicPorts: []api.Port{
					{Label: "http"},
				},
			},
		},
	}

	group.AddTask(task)
	job.AddTaskGroup(group)

	return job
}

func (s *NomadJobService) submitJob(job *api.Job) (*api.JobRegisterResponse, error) {
	jobs := s.client.Jobs()

	resp, _, err := jobs.Register(job, nil)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *NomadJobService) waitForServiceURL(jobID string) (string, int, error) {
	for i := 0; i < 30; i++ {
		netIp, port, err := s.getServiceURL(jobID)
		if err == nil {
			return netIp, port, nil
		}

		time.Sleep(250 * time.Millisecond)
	}

	return "", 0, fmt.Errorf("failed to get service URL for job %s after retries", jobID)
}

func (s *NomadJobService) getServiceURL(jobID string) (string, int, error) {
	jobs := s.client.Jobs()

	allocs, _, err := jobs.Allocations(jobID, false, nil)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get allocations for job %s: %w", jobID, err)
	}

	for _, alloc := range allocs {
		if alloc.ClientStatus == "running" {
			// Get allocation details
			allocsAPI := s.client.Allocations()
			allocDetail, _, err := allocsAPI.Info(alloc.ID, nil)
			if err != nil {
				continue
			}

			// Extract the reserved port
			if allocDetail.Resources != nil && allocDetail.Resources.Networks != nil {
				for _, network := range allocDetail.Resources.Networks {
					for _, port := range network.DynamicPorts {
						if port.Label == "http" {
							return network.IP, port.Value, nil
						}
					}
				}
			}
		}
	}

	return "", 0, fmt.Errorf("no running allocation found for job %s", jobID)
}

func (s *NomadJobService) PurgeJob(jobID string) error {
	jobs := s.client.Jobs()

	// Stop and purge the job
	_, _, err := jobs.Deregister(jobID, true, nil)
	if err != nil {
		return fmt.Errorf("failed to deregister job %s: %w", jobID, err)
	}

	// Clean up the internal port mapping
	s.rwMutex.Lock()
	delete(s.jobIdToPort, jobID)
	s.rwMutex.Unlock()

	s.logger.Info("jobs purged", "job_id", jobID)

	return nil
}

func (s *NomadJobService) Close() error {
	s.rwMutex.RLock()
	jobIDs := make([]string, 0, len(s.jobIdToPort))
	for j := range s.jobIdToPort {
		jobIDs = append(jobIDs, j)
	}
	s.rwMutex.RUnlock()

	var errs error
	for _, jid := range jobIDs {
		if err := s.PurgeJob(jid); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

func toPtr[T any](v T) *T {
	return &v
}

// Slugify converts a string to a URL-friendly slug
func slugify(s string) string {
	s = strings.ToLower(s)

	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	s, _, _ = transform.String(t, s)

	reg := regexp.MustCompile(`[^a-z0-9]+`)
	s = reg.ReplaceAllString(s, "-")

	s = strings.Trim(s, "-")

	reg = regexp.MustCompile(`-+`)
	s = reg.ReplaceAllString(s, "-")

	return s
}
