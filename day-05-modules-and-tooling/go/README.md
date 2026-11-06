# Go Modularity: The Architect's Deep Dive

Go's approach to modularity is intentionally simple but provides powerful primitives for building large-scale, maintainable systems.

## 1. Encapsulation via Capitalization
In Go, there is no `public` or `private` keyword. Instead, visibility is determined by the **first letter of the identifier**.
- **Exported (`Name`)**: Visible to any package that imports this package.
- **Unexported (`name`)**: Visible only within the package where it is defined.

**Senior Tip:** When designing a package, ask: "What is the minimum surface area required for the consumer to use this?" If a struct field doesn't *need* to be public, keep it unexported and provide a getter/setter if necessary. This protects your internal invariants.

## 2. The `internal/` Directory: Compiler-Enforced Privacy
The `internal/` directory is a unique feature of Go. Any package within a directory named `internal` can only be imported by packages that share the same parent directory.

```text
my-project/
├── go.mod
├── pkg/
│   └── domain/      <-- Publicly available
├── internal/
│   └── service/     <-- ONLY importable by code within my-project/
└── cmd/
    └── main/        <-- Entry point
```

This is not a convention; it is enforced by the Go compiler. It is the most effective way to prevent "Leaky Abstractions" in large monorepos.

## 3. Go Modules and Integrity
- **`go.mod`**: Defines the module path and the minimal version of requirements.
- **`go.sum`**: Contains the expected cryptographic checksums of the content of specific module versions. 
- **The Goal**: Reproducible builds. A `go.sum` file ensures that the code you tested in CI is *exactly* the same code that runs in production.

## 4. Multi-Module Workspaces (`go.work`)
For complex development where you are working on multiple modules simultaneously (e.g., a microservice and a shared library), `go.work` allows you to manage multiple modules without needing to use `replace` directives in `go.mod`, which can be brittle.

## How to Run
To execute the demonstration, run the following command from the `day-05-modules-and-tooling/go` directory:

```bash
go run ./cmd/main/main.go
```

---

## Summary Table for Senior Engineers

| Concept | Mechanism | Level | Purpose |
| :--- | :--- | :--- | :--- |
| **Visibility** | Capitalization | Identifier | Control API surface area |
| **Privacy** | `internal/` directory | Package | Enforce strict boundaries/encapsulation |
| **Versioning** | `go.mod` | Module | Deterministic dependency management |
| **Integrity** | `go.sum` | File | Verify dependency authenticity |
