package worker

import (
	"context"
	"time"

	"github.com/itstheanurag/executioner/internal/executor"
	"github.com/itstheanurag/executioner/internal/metrics"
	"github.com/itstheanurag/executioner/internal/queue"
	"github.com/rs/zerolog"
)

type Worker struct {
	id       int
	executor *executor.Executor
	manager  *queue.Manager
	logger   *zerolog.Logger
}

func NewWorker(id int, exec *executor.Executor, manager *queue.Manager, logger *zerolog.Logger) *Worker {
	return &Worker{
		id:       id,
		executor: exec,
		manager:  manager,
		logger:   logger,
	}
}

func (w *Worker) Start(ctx context.Context) {
	w.logger.Info().Int("worker_id", w.id).Msg("worker started")
	for {
		select {
		case job := <-w.manager.NextJob():
			metrics.ActiveWorkers.Inc()
			w.processJob(job)
			metrics.ActiveWorkers.Dec()
		case <-ctx.Done():
			w.logger.Info().Int("worker_id", w.id).Msg("worker stopping")
			return
		}
	}
}

func (w *Worker) processJob(job *queue.Job) {
	w.logger.Info().Int("worker_id", w.id).Str("job_id", job.ID).Msg("processing job")

	startTime := time.Now()
	result, err := w.executor.Execute(job.Ctx, job.Options)
	duration := time.Since(startTime).Milliseconds()

	// Record metrics
	status := "success"
	if err != nil {
		status = "error"
		job.Err <- err
		metrics.ExecutionsTotal.WithLabelValues(job.Options.LanguageID, status).Inc()
		return
	}

	if result.Status != "success" {
		status = result.Status
	}

	metrics.ExecutionsTotal.WithLabelValues(job.Options.LanguageID, status).Inc()
	metrics.ExecutionDuration.WithLabelValues(job.Options.LanguageID, "total").Observe(float64(duration))

	if result.MemoryKb > 0 {
		metrics.MemoryUsage.WithLabelValues(job.Options.LanguageID).Observe(float64(result.MemoryKb))
	}

	job.Result <- result
}
