# GoBetterAuth Project Guidelines

### Code Style Guide

- Always write clean code that is easy to read and maintain.
- Follow consistent naming conventions for variables, functions, and classes.
- Include comments where necessary to explain complex logic.
- Ensure proper error handling throughout the codebase.
- Follow best practices and concepts from SOLID principles for better software design.
- Use Go modules for dependency management.
- Adhere to idiomatic Go practices as outlined in the official Go documentation.
- Use `gofmt` to format your code before committing.
- Avoid global variables unless absolutely necessary.
- Use interfaces to define behavior and promote decoupling. Never code to implementations unless you're writing api services which are structs that handle business logic and import repositories for database access.
- For other services, define interfaces in the `interfaces.go` file within the `services` package and implement them in separate files just like the password service is an interface which has an argon2 implementation. So now it can easily be swapped out for another implementation if needed without changing the rest of the code that depends on it.
- API routes, handlers and services should be organized within their respective plugins to maintain modularity. Take a look at the email-password plugin as a guide to implement all other plugins.

# Testing Guidelines

- Write unit tests for as many components as possible to ensure reliability such as repositories, services and handlers as well as plugins.
- Use descriptive names for test cases to clearly indicate their purpose.
- Maintain a high level of code coverage to catch potential issues early.
- A good way to ensure high quality code is to write unit tests first to outline the feature and behaviour you want before implementing it. Then implement the code to make the tests pass. Finally, refactor the code while ensuring all tests still pass.
- Use table-driven tests to cover multiple scenarios in a concise manner.
- Mock external dependencies to isolate the unit being tested.
- Regularly run tests during development to catch issues early.
- Run `make build` to ensure the project builds successfully after changes.
- Then run `make test` to run all tests in the project.

### Documentation Guidelines

- Keep documentation up to date with code changes.
- Use clear and concise language in documentation.
- Include examples where applicable to illustrate usage.
- Document public APIs, functions, and complex logic thoroughly.
- Use markdown format for README files and other documentation.
- Maintain a changelog to track significant changes and updates in the project.
- Use inline code comments to explain non-obvious parts of the code.
- When updating a feature, ensure that any related documentation is also updated to reflect the changes.
- Create all docs in markdown format and within a top level docs/ directory.
