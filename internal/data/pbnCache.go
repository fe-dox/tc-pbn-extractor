package data

type JobStatus int

const (
	JobNotFound JobStatus = iota
	JobProcessing
	JobDone
)

type ResultsCache interface {
	Get(key string) (JobStatus, Result, error)
	GetStatus(key string) (JobStatus, error)
	SaveResult(key string, value Result) error
	SetStatusProcessing(key string) error
}
