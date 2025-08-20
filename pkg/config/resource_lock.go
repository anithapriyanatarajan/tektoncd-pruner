package config

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/logging"
)

const (
	// PrunerLockAnnotation is used to mark resources being processed
	PrunerLockAnnotation = "tekton-pruner.io/processing-lock"
	// LockTimeout defines how long a lock is valid
	LockTimeout = 5 * time.Minute
)

// ResourceLocker provides distributed locking for resources being processed
type ResourceLocker struct {
	instanceID string
}

// NewResourceLocker creates a new resource locker with a unique instance ID
func NewResourceLocker(instanceID string) *ResourceLocker {
	return &ResourceLocker{
		instanceID: instanceID,
	}
}

// ResourcePatchFuncs defines minimal interface needed for resource locking
type ResourcePatchFuncs interface {
	Get(ctx context.Context, namespace, name string) (metav1.Object, error)
	Patch(ctx context.Context, namespace, name string, patchBytes []byte) error
}

// TryLock attempts to acquire a lock on the resource
func (rl *ResourceLocker) TryLock(ctx context.Context, resource metav1.Object, funcs ResourcePatchFuncs) (bool, error) {
	logger := logging.FromContext(ctx)

	// Check if resource already has a lock
	annotations := resource.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	if lockValue, exists := annotations[PrunerLockAnnotation]; exists {
		// Parse lock timestamp and owner
		if rl.isLockValid(lockValue) && !rl.isOwnedByMe(lockValue) {
			logger.Debugw("Resource is locked by another instance",
				"namespace", resource.GetNamespace(),
				"name", resource.GetName(),
				"lock", lockValue)
			return false, nil
		}
	}

	// Try to acquire lock
	lockValue := fmt.Sprintf("%s:%d", rl.instanceID, time.Now().Unix())
	annotations[PrunerLockAnnotation] = lockValue

	patchData := fmt.Sprintf(`{"metadata":{"annotations":{"%s":"%s"}}}`,
		PrunerLockAnnotation, lockValue)

	err := funcs.Patch(ctx, resource.GetNamespace(), resource.GetName(), []byte(patchData))
	if err != nil {
		logger.Warnw("Failed to acquire lock",
			"namespace", resource.GetNamespace(),
			"name", resource.GetName(),
			"error", err)
		return false, err
	}

	// Verify we got the lock by re-reading the resource
	updatedResource, err := funcs.Get(ctx, resource.GetNamespace(), resource.GetName())
	if err != nil {
		return false, err
	}

	updatedAnnotations := updatedResource.GetAnnotations()
	if updatedAnnotations[PrunerLockAnnotation] == lockValue {
		logger.Debugw("Successfully acquired lock",
			"namespace", resource.GetNamespace(),
			"name", resource.GetName())
		return true, nil
	}

	logger.Debugw("Failed to acquire lock - lost race",
		"namespace", resource.GetNamespace(),
		"name", resource.GetName())
	return false, nil
}

// ReleaseLock removes the lock from the resource
func (rl *ResourceLocker) ReleaseLock(ctx context.Context, resource metav1.Object, funcs ResourcePatchFuncs) error {
	annotations := resource.GetAnnotations()
	if annotations == nil {
		return nil
	}

	if lockValue, exists := annotations[PrunerLockAnnotation]; exists {
		if rl.isOwnedByMe(lockValue) {
			patchData := fmt.Sprintf(`{"metadata":{"annotations":{"%s":null}}}`,
				PrunerLockAnnotation)
			return funcs.Patch(ctx, resource.GetNamespace(), resource.GetName(), []byte(patchData))
		}
	}

	return nil
}

// isLockValid checks if the lock is still within the timeout window
func (rl *ResourceLocker) isLockValid(lockValue string) bool {
	parts := parseLocationValue(lockValue)
	if len(parts) != 2 {
		return false
	}

	var timestamp int64
	if _, err := fmt.Sscanf(parts[1], "%d", &timestamp); err != nil {
		return false
	}

	lockTime := time.Unix(timestamp, 0)
	return time.Since(lockTime) < LockTimeout
}

// isOwnedByMe checks if the lock belongs to this instance
func (rl *ResourceLocker) isOwnedByMe(lockValue string) bool {
	parts := parseLocationValue(lockValue)
	if len(parts) != 2 {
		return false
	}

	return parts[0] == rl.instanceID
}

// parseLocationValue splits the lock value into instanceID and timestamp
func parseLocationValue(lockValue string) []string {
	// Split by ":" to get instanceID and timestamp
	var parts []string
	for i, part := range []rune(lockValue) {
		if part == ':' {
			parts = []string{lockValue[:i], lockValue[i+1:]}
			break
		}
	}
	if len(parts) == 0 {
		parts = []string{lockValue}
	}
	return parts
}
