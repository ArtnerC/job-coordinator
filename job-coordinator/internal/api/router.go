package api

import (
	"net/http"

	"github.com/dqme/job-coordinator/internal/coordinator"
)

// SetupRouter creates and configures the HTTP router with all handlers
func SetupRouter(coord *coordinator.Coordinator) *http.ServeMux {
	mux := http.NewServeMux()
	handler := NewHandler(coord)

	// Job control endpoints
	mux.HandleFunc("/job/start", handler.StartJobHandler)
	mux.HandleFunc("/job/pause", handler.PauseJobHandler)
	mux.HandleFunc("/job/cancel", handler.CancelJobHandler)
	mux.HandleFunc("/job/status", handler.StatusHandler)

	// Configuration endpoints
	mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handler.GetConfigHandler(w, r)
		} else if r.Method == http.MethodPut {
			handler.PutConfigHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Health check
	mux.HandleFunc("/health", handler.HealthHandler)

	return mux
}

// StartServer starts the HTTP server on the specified port
func StartServer(addr string, router *http.ServeMux) *http.Server {
	// Wrap router with logging middleware
	handler := LoggingMiddleware(router)
	
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	return server
}
