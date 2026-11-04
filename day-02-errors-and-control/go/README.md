# Go Error Handling & Resource Management

## Architectural Deep Dive

Go rejects traditional exception models in favor of treating errors as ordinary values. This design choice aligns with Go's philosophy of transparency and explicit control flow.

### 1. Sentinel Errors vs. Error Types
* **Sentinel Errors:** Package-level immutable error values (e.g., `io.EOF`, `sql.ErrNoRows`). Used when the caller needs to check for a specific, known failure condition using `errors.Is`.
* **Custom Error Types:** Structs implementing the `error` interface. Used when errors carry rich structural context (e.g., HTTP status codes, retry metadata, validation field names) inspected via `errors.As`.

### 2. Error Wrapping & Inspection (`%w`)
Go 1.13+ introduced error wrapping via `fmt.Errorf("... %w", err)`. 
* **`errors.Is(err, target)`:** Traverses the error chain to determine if any wrapped error matches a sentinel.
* **`errors.As(err, &targetStruct)`:** Traverses the error chain and type-asserts the first error matching the target pointer type.

### 3. Defer Mechanics & LIFO Resource Safety
`defer` pushes function calls onto a goroutine-local LIFO (Last-In, First-Out) stack executed when the surrounding function returns.
* **Argument Evaluation:** Arguments to deferred functions are evaluated immediately when the `defer` statement is reached, *not* when the function executes.
* **Named Return Values:** Deferred closures can inspect and modify named return values, making them ideal for transaction rollback or error enrichment patterns.