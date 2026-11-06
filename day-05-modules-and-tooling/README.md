# Day 5: Modularity, Package Management, and Tooling

## The Senior Engineer's Mental Model: Dependency & Encapsulation

As a senior engineer, you move beyond "how to import a file" to "how to design a system that survives scale, team growth, and dependency hell." This day focuses on the architectural implications of how languages handle modules and packages.

### Comparative Architecture Matrix

| Feature | Java (Maven/Gradle) | Go (Modules) | Python (PEP 517/621) |
| :--- | :--- | :--- | :--- |
| **Unit of Distribution** | JAR (Java Archive) | Module (as a repository/version) | Wheel / Source Distribution |
| **Encapsulation Mechanism** | `public`, `private`, `protected` (Class-level) | **Exported vs Unexported** (Package-level via capitalization) | Module-level via naming conventions (and `__all__`) |
| **Strict Privacy** | Package-private (default) | `internal/` directory (Compiler-enforced) | Convention-based (leading `_`) |
| **Dependency Resolution** | Centralized (Maven Central) | Decentralized (Direct Git URLs/Proxies) | Centralized (PyPI) |
| **Environment Isolation** | Classpath / OS Environment Variables | Single Binary / `go.mod` | Virtual Environments (`venv`, `Poetry`, `Conda`) |
| **Namespace Collision** | Package naming (e.g., `com.google...`) | Module path (e.g., `github.com/user/repo`) | Module name in `sys.path` |

---

### Key Architectural Patterns

#### 1. The "Internal" Pattern (Go Style)
In Go, the `internal/` directory is a powerful tool for library authors. Any package located under a directory named `internal/` can only be imported by code rooted at the same parent directory. This is **compiler-enforced**, meaning you cannot "accidentally" expose implementation details to consumers.
*   **Senior Insight:** Use `internal/` to prevent "leaky abstractions" where consumers start depending on your private implementation details, making future refactoring impossible.

#### 2. The "Interface/Implementation" Split (Python/Java Style)
Python and Java rely heavily on defining clear interfaces. In Python, this is often achieved via `__init__.py` to expose only a subset of a package's functionality.
*   **Senior Insight:** Use `__all__` in `__init__.py` to define the public API of your package. This provides clarity to other developers and helps prevent circular dependencies by controlling the entry points.

#### 3. Dependency Isolation (The Environment Problem)
*   **Java:** Usually relies on the runtime classpath. A common trap is "Jar Hell," where conflicting versions of the same library exist on the classpath.
*   **Go:** Favors a "one version per module" approach. `go.mod` and `go.sum` ensure that every developer and CI/CD runner uses the exact same checksummed dependency.
*   **Python:** Relies on `sys.path`. Without strict isolation (like `venv` or `Poetry`), global package installations can break entire operating systems or other projects.
*   **Senior Insight:** Never assume the environment. Your code should be "environment-aware" or, preferably, "environment-agnostic" through rigorous dependency pinning.

---

## Learning Objectives
- Understand the difference between package-level and class-level encapsulation.
- Master the Go `internal/` pattern for enforcing boundaries.
- Navigate the Python import system to avoid circular dependency deadlocks.
- Implement clean architectural layering (Domain vs. Service vs. Infrastructure).
