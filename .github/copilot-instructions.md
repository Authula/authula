### General Guidelines

- Keep things loosely coupled.

- Make sure to implement every feature until completion. Avoid creating mock data for features, implement everything with a real implementation.

---

### Backend

- Develop the code within the root of the directory.

- Only use the `slog` package for logging. Configure it to output logs in a human-readable format.

- Make sure the imports follow the following structure with spaces between each section:
  - Standard library imports
  - Third-party imports
  - Internal imports

- Coding guide:
  - Follow Go's official coding style as outlined in Effective Go and Go Code Review Comments.
  - Use `gofmt` or `goimports` to format code consistently.
  - Use meaningful variable and function names that convey intent.
  - Keep functions small and focused on a single responsibility.
  - Write comments for complex logic, exported functions, and types.
  - Prefer to use `any` instead of `interface{}` for better readability.

- Layer Responsibilities (Clean Architecture):
  - **Presentation Layer** (`handlers/`):
    - Responsible for HTTP request/response handling
    - Validate input using validation tags and custom validators
    - Call use cases with validated data
    - Transform use case results into HTTP responses
    - Handle error responses with appropriate status codes
    - Should NOT contain business logic
  - **Use Case Layer** (`auth/feature-name/usecase.go`):
    - Orchestrates the workflow for a specific feature
    - Coordinates between multiple services to achieve a business goal
    - Handles feature-specific validation and decision making
    - Translates between domain models and DTOs
    - Should NOT know about HTTP, should be framework-agnostic
  - **Service Layer** (`services/`):
    - Implements reusable business logic
    - Can be used by multiple use cases
    - Handles cross-cutting concerns like password hashing, email sending, rate limiting
    - Interacts with storage/database for data persistence
    - Should be focused on single responsibility
  - **Models** (`models/`):
    - Define domain models (User, Session, Account, etc.)
    - Define DTOs for request/response serialization
    - Include validation tags on DTOs for input validation
    - Include JSON tags on all serializable structs

- Project Layout (Go Clean Architecture):
  - The project is organized into clear layers following clean architecture principles. Each feature is self-contained with its own use cases, services, and handlers.

  **High-Level Directory Overview:**
  - `cmd/`: Application entrypoint
  - `config/`: Top-level configuration management
  - `internal/`: Core business logic organized by feature/domain:
    - `auth/`: Authentication features with modular subfolders (sign-up, sign-in, reset-password, etc.)
    - `handlers/`: HTTP request handlers (one per feature)
    - `services/`: Domain services with reusable business logic
    - `middleware/`: HTTP middleware (auth, rate limiting, hooks)
    - `config/`: Configuration manager implementations
    - `util/`: Utility functions and helpers
    - `events/`: Event system and webhooks
    - `plugins/`: Plugin registry and management
    - `constants/`: Application constants and errors
    - `common/`: Shared utilities for handlers
    - `admin/`: Admin-specific routes and handlers
  - `models/`: Shared data models and DTOs
  - `migrations/`: Database migrations (MySQL, PostgreSQL, SQLite)
  - `providers/`: OAuth2 provider implementations
  - `storage/`: Storage implementations (memory and database-based)
  - `events/`: Event bus and pub/sub implementations

  **Architecture Flow:**

  ```
  HTTP Request
    ↓
  Middleware (auth, rate_limit, hooks)
    ↓
  Handler (validation, error handling)
    ↓
  Use Case (orchestrates workflow)
    ↓
  Services (business logic)
    ↓
  Storage/Database
  ```

  **Design Patterns:**
  - Use dependency injection to provide service implementations to handlers and use cases
  - Keep domain logic free of framework-specific code
  - Each feature has its own use case that orchestrates the workflow
  - Services contain reusable business logic used by multiple use cases
  - Handlers are thin and delegate complex logic to use cases
  - Always add JSON tags to structs that will be serialized (especially request/response DTOs)

- Modular Design & Feature Implementation:
  - Follow S.O.L.I.D principles and make sure that every feature and domain is separated into its own folder and relevant files.
  - Make sure each code file is not very large and if needed separate it into multiple different files to make the code more readable, maintainable and testable.
  - Always create interfaces to abstract away implementations and make the code loosely coupled.

  **How to Implement a New Feature:**
  1. **Create Use Case** (`internal/auth/feature-name/usecase.go`):
     - Define use case interface/struct that orchestrates the feature workflow
     - Handle feature-specific validation and business logic orchestration
     - Call multiple services as needed to complete the workflow
     - Return errors with context
  2. **Create Service** (`internal/auth/feature-name/service.go` or `internal/services/`):
     - Implement reusable business logic
     - Can be used by multiple use cases
     - Interact with repositories for data access
     - Keep services independent and focused on a single responsibility
  3. **Create Handler** (`internal/handlers/feature_name.go`):
     - Parse and validate HTTP request
     - Call the use case
     - Format HTTP response with appropriate status codes
     - Return error responses in consistent format
  4. **Add Routes** (`internal/handlers/routes.go`):
     - Register the handler with appropriate HTTP method and path
     - Apply necessary middleware (auth, rate_limit, etc.)
  5. **Create Models/DTOs** (in `models/dto.go`):
     - Define request structs with validation tags
     - Define response structs with JSON tags
     - Keep DTOs separate from domain models
  6. **Write Tests**:
     - Unit tests for services and use cases
     - Mock dependencies using interfaces
     - Integration tests for the full flow
     - Test edge cases and error scenarios

- Error handling:
  - Return Errors Explicitly: Go's idiomatic way is to return errors as the last return value.
  - Wrap Errors: Use fmt.Errorf with %w to wrap errors, preserving the original error context. This allows for programmatic inspection using errors.Is and errors.As.

- Logging:
  - Structured Logging: Use structured loggers such as slog to output logs in a machine-readable format (JSON).
  - Contextual Logging: Pass a logger through the request context or as a dependency to functions, enriching logs with request-specific information (e.g., request ID, user ID).
  - Log Levels: Use appropriate log levels (DEBUG, INFO, WARN, ERROR, FATAL) for different severities.

- Configuration Management:
  - Manage application settings effectively.
  - Config Variables: Always provide properties within the main Base Config struct for all configuration variables.
  - Strict Validation: Validate configuration values at startup to catch errors early.

- Database Interactions
  - Efficient and safe database access.
  - Repository Pattern: Encapsulate database operations within a repository layer. This separates business logic from data access details and makes it easier to swap databases or ORMs.
  - Use `gorm` ORM, but avoid using it directly in handlers. Instead, create a repository layer that abstracts database operations.
  - Connection Pooling: Configure database connection pooling correctly to manage connections efficiently and prevent resource exhaustion.
  - Context for Database Operations: Always pass context. Context to database operations for timeout and cancellation.
  - Transactions: Use database transactions for operations that require atomicity (all or nothing).

- Middleware:
  - Leverage the base net/http package for middleware.

- Validation:
  - Ensure incoming data is valid.
  - Request Body Validation: Validate incoming JSON request bodies. Use libraries like `go-playground/validator/v10` for declarative validation.
  - Business Logic Validation: Perform additional validation within the service layer that depends on business rules or database lookups. Add the `validate:"required"` tag to struct fields to enforce required fields as well as other features that the validator provides.

- Testing:
  - Write comprehensive tests.
  - Unit Tests: Test individual functions and components in isolation. Mock external dependencies (database, external APIs).
  - Integration Tests: Test interactions between different components (e.g., handler -> usecase -> repository -> database). Use a test database or Docker Compose for dependencies.
  - End-to-End Tests: Test the entire application flow from the client perspective.
  - Table-Driven Tests: Use table-driven tests for multiple test cases, especially for handlers and validation.
  - Make sure that the tests reflect production too so that the tests are relevant and meaningful.
  - Always run `make test | grep --text "FAIL"` to run all tests and to detect any failing tests to fix.

- Security:
  - Implement security best practices.
  - Input Sanitization: Sanitize all user inputs to prevent injection attacks (SQL injection, XSS).
  - Authentication & Authorization:
    - Use secure authentication mechanisms (e.g., JWT, OAuth2).
    - Implement robust authorization checks (role-based access control, attribute-based access control).
  - Rate Limiting: Protect the library from abuse and DDoS attacks using rate limiting.
  - Mocking: Use Go's interfaces to enable easy mocking of dependencies for unit testing. Libraries like stretchr/testify/mock can be helpful.
  - DO NOT store plain text passwords/tokens in the DB. Always hash passwords using a strong hashing algorithm like bcrypt before storing them.

- Dependency Management:
  - Manage external libraries and modules.
  - Go Modules: Use Go Modules for dependency management.
  - Pin Dependencies: Pin specific versions of dependencies to ensure reproducible builds.
  - Vendoring (Optional): Consider vendoring dependencies for strict control over builds, especially in regulated environments.
  - Minimize Dependencies: Avoid unnecessary dependencies to reduce complexity and attack surface.

- Concurrency:
  - Go's concurrency features are powerful but require careful handling.
  - Goroutines & Channels: Use goroutines for concurrent execution and channels for safe communication between goroutines.
  - Context for Cancellation: Always pass context.Context to goroutines that perform long-running operations, allowing for graceful cancellation.
  - Avoid Race Conditions: Use sync.Mutex, sync.RWMutex, or channels to prevent race conditions when accessing shared resources.
  - Worker Pools: For CPU-bound or I/O-bound tasks, consider implementing worker pools to limit concurrent operations and manage resources.

- Observability
  - Provide support for observability via OpenTelemetry.

- Documentation & API Design
  - Clear and consistent API design.
  - RESTful Principles: Adhere to RESTful principles for API design (resources, HTTP methods, status codes).
  - Clear Endpoints: Design clear, predictable, and versioned API endpoints.
  - API Documentation: Document your API using OpenAPI/Swagger. Tools like swag can generate this from Go code annotations.
  - Consistent Response Formats: Use consistent JSON response formats for success and error messages.

By following these best practices, you can build the library in a robust, scalable, and easy to maintain manner.

---
