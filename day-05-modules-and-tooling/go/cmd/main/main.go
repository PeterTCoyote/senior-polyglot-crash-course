package main

import (
	"fmt"
	"log"

	// Simulating imports from our local layout
	"senior-polyglot-crash-course/day-05-modules-and-tooling/go/internal/service"
	"senior-polyglot-crash-course/day-05-modules-and-tooling/go/pkg/domain"
)

/*
   SENIOR ARCHITECT NOTE:
   This main.go demonstrates a standard "Clean Architecture" layout in Go.
   We use 'pkg/' for code that is safe for external consumption and
   'internal/' for code that is private to this specific module.

   The project structure we are simulating:
   ├── cmd/main/main.go (this file)
   ├── internal/service/engine.go
   └── pkg/domain/models.go
*/

func main() {
	fmt.Println("--- Day 5: Go Modularity & Encapsulation Demo ---")

	// 1. Interacting with 'pkg/domain' (Public)
	// We can create a User because the 'User' struct and its 'ID' field are exported.
	user := domain.User{
		ID:   "user-123",
		Name: "Alice Smith",
	}
	fmt.Printf("[Domain] Created Public Model: %+v\n", user)

	// 2. Interacting with 'internal/service' (Private)
	// The 'Service' package is only importable by code within this module.
	// It contains sensitive business logic we don't want other modules to touch.
	engine := service.NewEngine()
	err := engine.Process(user)
	if err != nil {
		log.Fatalf("Failed to process user: %v", err)
	}

	fmt.Println("[Main] Execution completed successfully.")
}
