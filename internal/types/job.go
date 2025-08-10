package types

type JobService interface {
	GetJobPort(jobID string) (int, bool)
	CreateJob(name string, targetURL string, isScript bool) (*CreateJobOutput, error)
	PurgeJob(jobID string) error
	Close() error
}

type CreateJobOutput struct {
	URL string
}
