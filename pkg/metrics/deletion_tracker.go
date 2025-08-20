package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"knative.dev/pkg/logging"
)

// DeletionTracker tracks deleted resources to prevent double-counting
type DeletionTracker struct {
	mu            sync.RWMutex
	deletedItems  map[string]time.Time
	cleanupTicker *time.Ticker
	stopCh        chan struct{}
}

// deletionKey creates a unique key for a deleted resource
func deletionKey(resourceType, namespace, name string) string {
	return fmt.Sprintf("%s/%s/%s", resourceType, namespace, name)
}

// NewDeletionTracker creates a new deletion tracker
func NewDeletionTracker() *DeletionTracker {
	dt := &DeletionTracker{
		deletedItems:  make(map[string]time.Time),
		cleanupTicker: time.NewTicker(5 * time.Minute),
		stopCh:        make(chan struct{}),
	}

	// Start background cleanup
	go dt.cleanupLoop()

	return dt
}

// RecordDeletion records a deletion and returns true if this is the first time
// this resource is being marked as deleted (i.e., should count in metrics)
func (dt *DeletionTracker) RecordDeletion(ctx context.Context, resourceType, namespace, name string) bool {
	logger := logging.FromContext(ctx)
	key := deletionKey(resourceType, namespace, name)

	dt.mu.Lock()
	defer dt.mu.Unlock()

	// Check if we've already recorded this deletion recently
	if lastDeleted, exists := dt.deletedItems[key]; exists {
		// If deleted within the last minute, consider it a duplicate
		if time.Since(lastDeleted) < time.Minute {
			logger.Debugw("Duplicate deletion detected, skipping metrics",
				"key", key, "lastDeleted", lastDeleted)
			return false
		}
	}

	// Record the deletion
	dt.deletedItems[key] = time.Now()
	logger.Debugw("Recorded new deletion", "key", key)
	return true
}

// cleanupLoop removes old entries from the tracking map
func (dt *DeletionTracker) cleanupLoop() {
	for {
		select {
		case <-dt.cleanupTicker.C:
			dt.cleanup()
		case <-dt.stopCh:
			return
		}
	}
}

// cleanup removes entries older than 10 minutes
func (dt *DeletionTracker) cleanup() {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute)
	for key, deletedTime := range dt.deletedItems {
		if deletedTime.Before(cutoff) {
			delete(dt.deletedItems, key)
		}
	}
}

// Stop stops the cleanup goroutine
func (dt *DeletionTracker) Stop() {
	close(dt.stopCh)
	dt.cleanupTicker.Stop()
}

// Global deletion tracker instance
var globalDeletionTracker *DeletionTracker
var trackerOnce sync.Once

// GetDeletionTracker returns the global deletion tracker instance
func GetDeletionTracker() *DeletionTracker {
	trackerOnce.Do(func() {
		globalDeletionTracker = NewDeletionTracker()
	})
	return globalDeletionTracker
}
