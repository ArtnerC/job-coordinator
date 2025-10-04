package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/dqme/job-coordinator/internal/coordinator"
	"github.com/dqme/job-coordinator/internal/models"
)

// Handler holds the coordinator reference for API handlers
type Handler struct {
	coord *coordinator.Coordinator
}

// NewHandler creates a new API handler
func NewHandler(coord *coordinator.Coordinator) *Handler {
	return &Handler{coord: coord}
}

// StartJobHandler handles POST /job/start
func (h *Handler) StartJobHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := h.coord.Start()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	job := h.coord.GetJob()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id":  job.ID,
		"status":  string(job.Status),
		"message": "Job started successfully",
	})
}

// PauseJobHandler handles PUT /job/pause
func (h *Handler) PauseJobHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := h.coord.Pause()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	job := h.coord.GetJob()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id":  job.ID,
		"status":  string(job.Status),
		"message": "Job paused successfully",
	})
}

// CancelJobHandler handles PUT /job/cancel
func (h *Handler) CancelJobHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := h.coord.Cancel()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	job := h.coord.GetJob()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id":  job.ID,
		"status":  string(job.Status),
		"message": "Job cancelled successfully",
	})
}

// StatusHandler handles GET /job/status
func (h *Handler) StatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	job := h.coord.GetJob()
	
	// Calculate completion percentage
	completionPercent := 0.0
	if job.TotalWorkUnits > 0 {
		completionPercent = float64(job.ProcessedCount) / float64(job.TotalWorkUnits) * 100
	}

	response := models.JobStatusResponse{
		JobID:              job.ID,
		Status:             string(job.Status),
		TotalWorkUnits:     job.TotalWorkUnits,
		ProcessedWorkUnits: job.ProcessedCount,
		ErrorCount:         job.ErrorCount,
		CompletionPercent:  completionPercent,
		StartTime:          job.StartTime,
		EndTime:            job.EndTime,
		Errors:             job.Errors,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HealthHandler handles GET /health
func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	job := h.coord.GetJob()
	
	response := map[string]interface{}{
		"status":       "ok",
		"job_id":       job.ID,
		"job_status":   string(job.Status),
		"is_processing": job.Status == models.JobStatusRunning,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// LoggingMiddleware logs each HTTP request
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
