package main

import (
	"errors"
	"fmt"
	"log"
	"time"
)

// ============================================================================
// 1. Sentinel Errors & Custom Error Types
// ============================================================================

// Sentinel errors for well-known domain failures.
var (
	ErrSentinelNotFound   = errors.New("domain: record not found")
	ErrSentinelConstraint = errors.New("domain: integrity constraint violation")
)

// AppError represents a rich domain error carrying operational metadata.
// Implementing Unwrap() and Is() allows seamless integration with errors.Is/As.
type AppError struct {
	Op        string    // Operation where error occurred (e.g., "UserService.Get")
	Timestamp time.Time // When the error occurred
	Err       error     // Underlying wrapped error
	Code      int       // Enterprise error code
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] code=%d at %s: %v -> caused by: %v", e.Op, e.Code, e.Timestamp.Format(time.RFC3339), e.Err)
	}
	return fmt.Sprintf("[%s] code=%d at %s", e.Op, e.Code, e.Timestamp.Format(time.RFC3339))
}

// Unwrap supports errors.Unwrap and errors.Is/As traversal.
func (e *AppError) Unwrap() error {
	return e.Err
}

// Is supports matching against sentinel errors or similar AppErrors.
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if ok {
		return t.Code == e.Code
	}
	return false
}

// ============================================================================
// 2. Resource Simulation & Defer LIFO Demonstration
// ============================================================================

type MockResource struct {
	Name string
}

func (r *MockResource) Acquire() error {
	log.Printf("[RESOURCE] Acquiring lock/connection for: %s", r.Name)
	return nil
}

func (r *MockResource) Commit() error {
	log.Printf("[RESOURCE] Committing transaction for: %s", r.Name)
	return nil
}

func (r *MockResource) Rollback() error {
	log.Printf("[RESOURCE] Rollback executed for: %s", r.Name)
	return nil
}

func (r *MockResource) Close() {
	log.Printf("[RESOURCE] Releasing resource cleanup for: %s", r.Name)
}

// ============================================================================
// 3. Simulated Business Logic Service
// ============================================================================

func ProcessUserTransaction(userID int) error {
	res := &MockResource{Name: fmt.Sprintf("db-conn-%d", userID)}
	
	if err := res.Acquire(); err != nil {
		return &AppError{
			Op:        "ProcessUserTransaction.Acquire",
			Timestamp: time.Now().UTC(),
			Err:       err,
			Code:      5001,
		}
	}

	// Track transaction success state for LIFO defer cleanup logic
	success := false
	defer func() {
		if !success {
			_ = res.Rollback()
		}
		res.Close()
	}()

	// Simulate business logic failure based on ID
	if userID == 404 {
		// Wrap sentinel error
		return &AppError{
			Op:        "ProcessUserTransaction.Query",
			Timestamp: time.Now().UTC(),
			Err:       fmt.Errorf("user %d: %w", userID, ErrSentinelNotFound),
			Code:      4040,
		}
	}

	if userID == 500 {
		return &AppError{
			Op:        "ProcessUserTransaction.Constraint",
			Timestamp: time.Now().UTC(),
			Err:       ErrSentinelConstraint,
			Code:      4090,
		}
	}

	// Happy path commit
	if err := res.Commit(); err != nil {
		return &AppError{
			Op:        "ProcessUserTransaction.Commit",
			Timestamp: time.Now().UTC(),
			Err:       err,
			Code:      5002,
		}
	}

	success = true
	return nil
}

// ============================================================================
// 4. Main Entrypoint & Error Inspection
// ============================================================================

func main() {
	log.Println("=== Starting Go Error Handling & Flow Control Demo ===")

	testCases := []int{200, 404, 500}

	for _, id := range testCases {
		log.Printf("\n--- Processing Request for User ID: %d ---", id)
		err := ProcessUserTransaction(id)
		if err != nil {
			log.Printf("ERROR DETECTED: %v", err)

			// Inspecting with errors.Is for sentinel errors
			if errors.Is(err, ErrSentinelNotFound) {
				log.Println("-> Inspection match: Caught a NOT FOUND sentinel error. Returning HTTP 404.")
			} else if errors.Is(err, ErrSentinelConstraint) {
				log.Println("-> Inspection match: Caught a CONSTRAINT VIOLATION. Returning HTTP 409.")
			}

			// Inspecting with errors.As for custom AppError metadata
			var appErr *AppError
			if errors.As(err, &appErr) {
				log.Printf("-> Inspection match: Structured AppError caught in operation [%s] with Error Code [%d]", appErr.Op, appErr.Code)
			}
		} else {
			log.Println("-> Transaction completed successfully.")
		}
	}
}