# Contribution Guide

### Code of Conduct

This project is committed to fostering a welcoming and inclusive community. As a contributor,
you agree to uphold the principles outlined in the [Code of Conduct](./CODE_OF_CONDUCT.md). If
you have concerns or encounter any unacceptable behavior, please reach out to authula.tech@gmail.com.

---

### I Want To Contribute

> ### Legal Notice
>
> By contributing to this project, you agree that you are the original author of the contributed material and that you have the necessary rights to contribute it, and that the contributed material may be distributed under the project's license.

### Submit issues

### Reporting bugs

We rely on bug reports to enhance this project for all users. To assist us, we have a bug reporting template specifying the necessary details. Ensure you check our [existing bug reports](https://github.com/Authula/authula/issues?q=is%3Aissue+is%3Aopen+label%3Abug) prior to submitting a new one to avoid duplicates.

### Reporting security issues

Avoid creating a public GitHub issue for security concerns. If you discover a security vulnerability, contact us directly via email at authula.tech@gmail.com rather than opening an issue.

### Requesting new features

To request new features, please create an issue on this project.
To ensure that we can understand the problem you are looking to solve, please be as detailed as possible.
To see what other people have already suggested, you can look [here](https://github.com/Authula/authula/issues?q=is%3Aissue+is%3Aopen+label%3Aenhancement).
Please be aware that duplicate issues might already exist. If you are creating a new issue, please check existing open, or recently closed. Having a single vote for an issue is far easier for us to prioritise.

---

### Begin Contributing

#### Requirements

To start contributing:

- [Fork](https://docs.github.com/en/github/getting-started-with-github/fork-a-repo) the repository
- Clone the fork on your workstation:

  ```bash
  $ git clone git@github.com:{YOUR_USERNAME}/authula.git

  $ cd authula
  ```

Choose one of the following development setups:

1. `Devcontainers`:

Once you have this repo cloned to your local system, you will need to install the VSCode extension [Remote Development](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.vscode-remote-extensionpack).

Then run the following command from the command palette:
`Dev Containers: Open Folder in Container...`

This will automatically select the workspace folder. But if you need to find the project manually then it is located at `/workspaces/authula`. You can then proceed to the development section below.

2. `Without devcontainers`:

- Make sure to install [Go](https://go.dev/doc/install) and set it up as shown in their docs.

#### Development:

1. **Set Up the Environment**

- Once you have your environment set up and you are within the project, run the setup target. It is the single place that installs everything you need:

  ```bash
  $ make setup
  ```

  This does three things:

  - `make install` – downloads Go module dependencies (`go mod download && go mod tidy`).
  - `make tools` – installs `golangci-lint` and `air` (hot reloading) into the project's `./bin/` directory, which is where the other make targets expect them.
  - `make hooks` – points `core.hooksPath` at the repo's `.githooks/` directory so the pre-commit hook runs for you.

  Each of these can also be run on its own if you only need to redo one step.

- Then as a test run `make build` to ensure the project builds successfully, this could take a few seconds to a minute.

- Make a copy of the `config.example.toml` and rename it to `config.toml`. Replace any of the necessary properties.

- Then you can run the dev server:

  ```bash
  $ make run

  # or, with hot reloading
  $ make dev
  ```

- About the pre-commit hook: it calls `scripts/pre-commit-checks.sh`, which runs `make format`, `make vet`, `make lint`, `make build` and `make test` in that order and rejects the commit if any step fails. If `make format` changes a staged file, the commit is stopped so you can review and re-stage it. The hook only runs when Go files, `go.mod` or `go.sum` are staged. Use `git commit --no-verify` to skip it for a WIP commit, and `make hooks-uninstall` to remove it.

2. **Project Structure**

Authula can be used in two ways, and the layout reflects that:

- **As a library.** The module root is the public Go API. Packages such as `config/`, `models/`, `events/` and `middleware/` are what an application imports when it embeds Authula.
- **As a standalone server.** `cmd/` holds the entry points (the auth server, the migration CLI, the OpenAPI exporter). These are thin wrappers that wire the library together and run it.

Inside the module, the code is split into three main areas:

- **`core/`** – the base behaviour every setup gets: users, sessions, error types, the event system, security helpers and the core HTTP routes.
- **`plugins/`** – every authentication feature lives in its own plugin folder (email/password, TOTP, OAuth2, rate limiting, organizations and so on). A plugin is self-contained: it registers its own routes, services, migrations and hooks, and declares the capabilities it provides. Most new features are added as a plugin, or as an extension to an existing one.
- **`internal/`** – plumbing that is not part of the public API, such as bootstrapping, the router, the plugin registry and the migration manager. Anything here can change without a versioned release.

Both `core/` and each plugin follow the same layered structure, from the outside in:

- **handlers** – parse the HTTP request, call a use case, write the response. No business logic.
- **usecases** – orchestrate a single application workflow (for example "register a user") by combining services.
- **services** – reusable business logic behind an interface, so implementations can be swapped and mocked.
- **repositories** – data access. All database work goes through a repository interface; nothing above this layer talks to the database directly.
- **types / constants** – request and response shapes, errors and shared values used by the layers above.

Dependencies always point inwards (handlers → use cases → services → repositories) and are passed in through constructors, never reached for globally.

Supporting folders around these hold cross-cutting pieces: database and storage `adapters/`, versioned `migrations/`, OpenAPI generation, and `scripts/` used by the Makefile and git hooks. Tests live next to the code they cover.

The `.agents/skills/` folder contains short playbooks for each of these layers (handlers, use cases, services, repositories, plugins, testing). Read the matching one before working on that part of the code – they document the conventions the project expects.

3. **Testing**

- Run unit and integration tests:

  ```bash
  # Run all tests
  make test

  # Run specific tests
  go test -v ./path/to/package -function TestName
  ```

4. **Making Changes**

- Follow the project’s folder structure.
- Write tests for new features.
- Ensure all code passes tests before submitting a PR.

5. **Submitting a PR**

- Push your branch and open a pull request.
- Fill out the PR and link related issues.

#### Extras

`OpenAPI Export`:

```bash
# Export JSON spec (default output: openapi.json)
$ make openapi-export

# Export with custom options
$ make openapi-export ARGS="--output ./openapi.json --format json --openapi-version 3.1.0"

# Export YAML
$ make openapi-export ARGS="--format yaml"

# Make sure to format the openapi.json file using jq to keep it consistent with how we prettify the file to prevent merge conflicts too. Run the following command:
$ jq "." openapi.json > temp.json && mv temp.json openapi.json
```

---

### AI Agents

We welcome contributions from AI agents, but please ensure that any code generated by an AI agent is reviewed and tested by a human before submission. This helps maintain the quality and integrity of the codebase.

You can run the following script from the root of the project to symlink all the Agent Skills to some of the most well-known agent folders:

```bash
$ bash ./scripts/agent-skills-symlinker.sh
```

If you're using a different agent and the directory is not included in this script, you can raise a PR so we can add support for it.

---
