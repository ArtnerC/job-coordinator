package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dqme/job-coordinator/internal/api"
	"github.com/dqme/job-coordinator/internal/config"
	"github.com/dqme/job-coordinator/internal/coordinator"
	"github.com/dqme/job-coordinator/internal/distributor"
	"github.com/dqme/job-coordinator/internal/models"
)

func main() {
	log.Println("Starting Job Coordinator...")

	// Load configuration from CLI args and environment
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded: job_id=%s, batch_size=%d, distributor_type=%s",
		cfg.JobID, cfg.BatchSize, cfg.DistributorType)

	// Create distributor
	dist, err := distributor.NewDistributor(context.Background(), *cfg)
	if err != nil {
		log.Fatalf("Failed to create distributor: %v", err)
	}
	log.Printf("Distributor created: type=%s", cfg.DistributorType)

	// Create job
	job := &models.Job{
		ID:             cfg.JobID,
		Status:         models.JobStatusPending,
		TotalWorkUnits: 0,
		ProcessedCount: 0,
		ErrorCount:     0,
		Errors:         []string{},
		CompletionTTL:  cfg.CompletionTTL,
	}

	// Create coordinator
	coord := coordinator.NewCoordinator(job, cfg, dist)
	log.Printf("Coordinator initialized for job: %s", job.ID)

	// Setup API router and server
	router := api.SetupRouter(coord)
	addr := fmt.Sprintf(":%d", cfg.APIPort)
	server := api.StartServer(addr, router)
	
	// Start API server in background
	go func() {
		log.Printf("Starting API server on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("API server failed: %v", err)
		}
	}()

	// Auto-start job if configured
	if cfg.AutoStart {
		log.Println("Auto-starting job...")
		if err := coord.Start(); err != nil {
			log.Fatalf("Failed to start job: %v", err)
		}
	} else {
		log.Println("Job created in pending state. Use POST /job/start to begin processing.")
	}

	// Wait for shutdown signal
	shutdownReason := <-coordinator.WaitForShutdownSignal(cfg.CompletionTTL, coord.ShutdownChan())
	log.Printf("Shutdown initiated: reason=%s", shutdownReason)

	// Graceful shutdown
	shutdownCtx := context.Background()
	if err := coordinator.GracefulShutdown(shutdownCtx, coord, 5*time.Second); err != nil {
		log.Printf("Graceful shutdown completed with errors: %v", err)
		os.Exit(1)
	}

	// Shutdown API server
	shutdownAPICtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownAPICtx); err != nil {
		log.Printf("API server shutdown error: %v", err)
	}

	log.Println("Job Coordinator shutdown complete")
}
