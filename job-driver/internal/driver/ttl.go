package coordinator

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// TTLManager handles TTL-based automatic shutdown after job completion
type TTLManager struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	ttl        time.Duration
}

// NewTTLManager creates a new TTL manager with the specified duration
func NewTTLManager(ttl time.Duration) *TTLManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &TTLManager{
		ctx:        ctx,
		cancelFunc: cancel,
		ttl:        ttl,
	}
}

// StartTTL starts the TTL countdown. Returns a channel that signals when TTL expires.
func (tm *TTLManager) StartTTL() <-chan struct{} {
	doneChan := make(chan struct{})
	
	go func() {
		select {
		case <-time.After(tm.ttl):
			log.Printf("TTL expired after %v, initiating shutdown", tm.ttl)
			close(doneChan)
		case <-tm.ctx.Done():
			log.Println("TTL cancelled before expiry")
			close(doneChan)
		}
	}()
	
	return doneChan
}

// Cancel cancels the TTL timer
func (tm *TTLManager) Cancel() {
	tm.cancelFunc()
}

// WaitForShutdownSignal waits for either:
// - TTL expiry after job completion
// - OS signal (SIGTERM/SIGINT)
// Returns a channel that signals when shutdown should commence
func WaitForShutdownSignal(ttl time.Duration, jobCompletedChan <-chan struct{}) <-chan string {
	shutdownChan := make(chan string, 1)
	
	// Listen for OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	
	go func() {
		select {
		case <-jobCompletedChan:
			// Job completed - wait for TTL
			log.Printf("Job completed, starting TTL countdown (%v)", ttl)
			ttlManager := NewTTLManager(ttl)
			ttlExpiredChan := ttlManager.StartTTL()
			
			select {
			case <-ttlExpiredChan:
				shutdownChan <- "ttl_expired"
			case sig := <-sigChan:
				log.Printf("Received signal %v during TTL countdown", sig)
				ttlManager.Cancel()
				shutdownChan <- "signal"
			}
			
		case sig := <-sigChan:
			// Signal received before job completion
			log.Printf("Received signal %v", sig)
			shutdownChan <- "signal"
		}
	}()
	
	return shutdownChan
}

// GracefulShutdown performs graceful shutdown of the coordinator
func GracefulShutdown(ctx context.Context, coord *Coordinator, shutdownTimeout time.Duration) error {
	log.Println("Starting graceful shutdown...")
	
	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()
	
	// Create error channel
	errChan := make(chan error, 1)
	
	// Perform shutdown in goroutine
	go func() {
		// Close coordinator (includes distributor)
		if err := coord.Close(); err != nil {
			log.Printf("Error closing coordinator: %v", err)
			errChan <- err
			return
		}
		
		log.Println("Coordinator closed successfully")
		errChan <- nil
	}()
	
	// Wait for shutdown or timeout
	select {
	case err := <-errChan:
		if err != nil {
			log.Printf("Graceful shutdown completed with errors: %v", err)
			return err
		}
		log.Println("Graceful shutdown completed successfully")
		return nil
		
	case <-shutdownCtx.Done():
		log.Printf("Graceful shutdown timed out after %v", shutdownTimeout)
		return shutdownCtx.Err()
	}
}
