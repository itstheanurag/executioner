package queue

import (
	"context"

	"github.com/itstheanurag/executioner/internal/executor"
	"github.com/itstheanurag/executioner/internal/metrics"
)

type Job struct {
	ID      string
	Options executor.ExecuteOptions
	Result  chan *executor.ExecutionResult
	Err     chan error
	Ctx     context.Context
}

type Manager struct {
	jobQueue chan *Job
}

func NewManager(capacity int) *Manager {
	return &Manager{
		jobQueue: make(chan *Job, capacity),
	}
}

func (m *Manager) Submit(job *Job) {
	m.jobQueue <- job
	metrics.QueueDepth.Set(float64(len(m.jobQueue)))
}

func (m *Manager) NextJob() <-chan *Job {
	return m.jobQueue
}

func (m *Manager) UpdateQueueMetric() {
	metrics.QueueDepth.Set(float64(len(m.jobQueue)))
}
