package service

import (
	"fmt"

	"senior-polyglot-crash-course/day-05-modules-and-tooling/go/pkg/domain"
)

// Engine represents a business logic processor.
// It is unexported in terms of its intended use: it should only be used by the application entry point.
type Engine struct {
	// internalState is unexported, ensuring it cannot be tampered with from outside.
	internalState string
}

// NewEngine is an exported constructor for the unexported Engine struct.
// This is a common Go pattern: provide a public constructor for a type that might
// have private state or logic that shouldn't be manually instantiated.
func NewEngine() *Engine {
	return &Engine{
		internalState: "Initialized",
	}
}

// Process demonstrates business logic that operates on the domain model.
func (e *Engine) Process(u domain.User) error {
	fmt.Printf("[Service] Processing user %s (ID: %s) with state: %s\n", u.Name, u.ID, e.internalState)
	
	// In a real system, this might involve database calls or external APIs.
	e.internalState = "Processed"
	return nil
}

/*
   ARCHITECTURAL NOTE:
   Because this code resides in the 'internal/' directory, 
   no other module in the entire Go ecosystem can import this package.
   It is truly 'private' to this project.
*/
