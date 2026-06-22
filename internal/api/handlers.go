package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/itstheanurag/executioner/internal/executor"
	"github.com/itstheanurag/executioner/internal/languages"
	"github.com/itstheanurag/executioner/internal/queue"
)

type ExecutionRequest struct {
	Language    string `json:"language"`
	SourceCode  string `json:"source_code"`
	Stdin       string `json:"stdin"`
	TimeLimit   int    `json:"time_limit"`   // in seconds
	MemoryLimit int    `json:"memory_limit"` // in MB
}

type LanguageResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Handler struct {
	queueManager *queue.Manager
	registry     *languages.Registry
}

func NewHandler(manager *queue.Manager, registry *languages.Registry) *Handler {
	return &Handler{
		queueManager: manager,
		registry:     registry,
	}
}

func (h *Handler) ListLanguages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	langs := h.registry.List()
	resp := make([]LanguageResponse, 0, len(langs))
	for _, l := range langs {
		resp = append(resp, LanguageResponse{ID: l.ID, Name: l.Name})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Default limits
	if req.TimeLimit == 0 {
		req.TimeLimit = 2
	}
	if req.MemoryLimit == 0 {
		req.MemoryLimit = 256
	}

	jobID := "job-" + time.Now().Format("150405.000000")
	resultChan := make(chan *executor.ExecutionResult, 1)
	errChan := make(chan error, 1)

	// Create context with timeout for the job
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.TimeLimit+1)*time.Second)
	defer cancel()

	job := &queue.Job{
		ID: jobID,
		Options: executor.ExecuteOptions{
			LanguageID:    req.Language,
			SourceCode:    req.SourceCode,
			Stdin:         req.Stdin,
			TimeLimitMs:   req.TimeLimit * 1000,
			MemoryLimitKb: req.MemoryLimit * 1024,
		},
		Result: resultChan,
		Err:    errChan,
		Ctx:    ctx,
	}

	h.queueManager.Submit(job)

	select {
	case res := <-resultChan:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	case err := <-errChan:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	case <-ctx.Done():
		http.Error(w, "Execution timed out", http.StatusGatewayTimeout)
	}
}
