package api

import (
	"encoding/json"
	"net/http"

	"github.com/cvs-health-source-code/digital-qme-system/job-driver/internal/config"
	"github.com/cvs-health-source-code/digital-qme-system/job-driver/internal/models"
)

// GetConfigHandler handles GET /config
func (h *Handler) GetConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := h.coord.GetConfig()
	job := h.coord.GetJob()

	manifestPath := ""
	if cfg.ManifestPath != nil {
		manifestPath = *cfg.ManifestPath
	}

	measuresManifestPath := ""
	if cfg.MeasuresManifestPath != nil {
		measuresManifestPath = *cfg.MeasuresManifestPath
	}

	response := models.ConfigResponse{
		JobID:                    cfg.JobID,
		BatchSize:                cfg.BatchSize,
		RemainderThreshold:       cfg.RemainderThreshold,
		BasePath:                 cfg.BasePath,
		ManifestPath:             manifestPath,
		MeasuresPath:             cfg.MeasuresPath,
		MeasuresToRun:            cfg.MeasuresToRun,
		MeasuresManifestPath:     measuresManifestPath,
		DistributorType:          cfg.DistributorType,
		CompletionTTL:            cfg.CompletionTTL.String(),
		ConcurrentFileProcessors: cfg.ConcurrentFileProcessors,
		Status:                   string(job.Status),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// PutConfigHandler handles PUT /config
func (h *Handler) PutConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var newCfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Failed to parse JSON request body",
			Details: err.Error(),
		})
		return
	}

	// Check which fields are being updated and validate runtime modifiability
	updatedFields := []string{}
	rejectedFields := []string{}
	
	currentCfg := h.coord.GetConfig()
	job := h.coord.GetJob()
	jobStatus := string(job.Status)

	// Check each field for changes and runtime modifiability
	if newCfg.ConcurrentFileProcessors != currentCfg.ConcurrentFileProcessors {
		if config.IsRuntimeModifiable("concurrent_file_processors", jobStatus) {
			updatedFields = append(updatedFields, "concurrent_file_processors")
		} else {
			rejectedFields = append(rejectedFields, "concurrent_file_processors")
		}
	}

	if !equalStringMaps(newCfg.DistributorConfig, currentCfg.DistributorConfig) {
		if config.IsRuntimeModifiable("distributor_config", jobStatus) {
			updatedFields = append(updatedFields, "distributor_config")
		} else {
			rejectedFields = append(rejectedFields, "distributor_config")
		}
	}

	// Check for attempts to modify non-runtime fields
	if newCfg.BatchSize != currentCfg.BatchSize {
		rejectedFields = append(rejectedFields, "batch_size")
	}
	if newCfg.BasePath != currentCfg.BasePath {
		rejectedFields = append(rejectedFields, "base_path")
	}
	if newCfg.DistributorType != currentCfg.DistributorType {
		rejectedFields = append(rejectedFields, "distributor_type")
	}

	// If there are rejected fields, return error
	if len(rejectedFields) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.UpdateConfigResponse{
			Message:         "Some fields cannot be modified at runtime",
			UpdatedFields:   []string{},
			RejectedFields:  rejectedFields,
			RejectionReason: "Fields are not runtime-modifiable",
		})
		return
	}

	// If no fields to update, return success but note no changes
	if len(updatedFields) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(models.UpdateConfigResponse{
			Message:       "No fields changed",
			UpdatedFields: []string{},
		})
		return
	}

	// Apply updates
	if err := h.coord.UpdateConfig(&newCfg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse{
			Error:   "update_failed",
			Message: "Failed to update configuration",
			Details: err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.UpdateConfigResponse{
		Message:       "Configuration updated successfully",
		UpdatedFields: updatedFields,
	})
}

// Helper function to compare string maps
func equalStringMaps(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
