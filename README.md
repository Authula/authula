<p align="center">
  <img src="./project-logo.png" height="150" width="150" alt="Project Logo"/>
</p>

---

### Table of Contents

1.  [Introduction](#introduction)
1.  [Features](#features)
1.  [Docs](#docs)
1.  [Contributing](#contributing)

---

### Introduction

✨ Overview

**GoBetterAuth** is an open-source authentication solution that scales with you. Embed it as a library in your Go app, or run it as a standalone auth server for any language or framework. All functionality is delivered through a powerful plugin system, allowing you to compose exactly the authentication stack you need — no more, no less, all built with clean architecture. **GoBetterAuth** is flexible enough to integrate with any technology stack. It streamlines the implementation of essential security features through a clean, modular architecture, allowing developers to concentrate on building their applications without the overhead of managing authentication complexities.

---

### 🎯 Who is it for?

GoBetterAuth is ideal for:

- Startups that want full control over their authentication stack
- Teams building microservices or multi-backend systems
- Companies with self-hosting or compliance requirements
- Go developers who want first-class embedded auth
- Anyone who wants modern auth without SaaS lock-in

---

🧩 Plugins & Capabilities

GoBetterAuth is architected around a powerful plugin and capability system.

**Plugins** are modular packages that encapsulate related authentication features.  
**Capabilities** represent individual, fine-grained functionalities exposed by these plugins.

Each plugin can offer multiple capabilities, and every route in your application explicitly declares which capabilities it leverages. This approach ensures that authentication logic is:

- **Explicit** – No hidden behaviors; every capability is clearly declared.
- **Composable** – Mix and match only the features you need.
- **Auditable** – Easily track which routes use which authentication features.
- **Understandable** – The authentication flow is transparent and easy to reason about.

This design empowers you to build secure, maintainable, and highly customizable authentication flows tailored to your application's needs.

---

### Features via Plugins

- 📧 Email & Password: Authentication, Email Verification & Password Reset
- 🌐 OAuth providers
- 💾 Multiple database backends
- 🗄️ Secondary storage (Redis, memory, DB)
- ⚡ Rate limiting
- 🛡️ CSRF protection
- 🪝 Hooks system
- 📨 Event bus
- 🧩 Custom routes and logic

---

### 🪝 Hooks System

GoBetterAuth includes a powerful, lifecycle-based hooks system that lets you intercept and customize request handling at every stage of the HTTP pipeline.

Hooks allow you to implement:

- custom authentication logic
- request validation
- logging & tracing
- metrics
- access control
- A/B testing
- feature flags
- audit trails
- custom headers
- dynamic routing

All without modifying core code.

Build your own plugins for:

- business logic
- custom routes
- custom auth flows
- external integrations
- internal tooling

---

### ⚙️ Deployment Modes

### Embedded Mode (Go Library)

Embed GoBetterAuth directly into your Go application:

```go
import (
  gobetterauth "github.com/GoBetterAuth/go-better-auth"
  gobetterauthconfig "github.com/GoBetterAuth/go-better-auth/config"
  gobetterauthmodels "github.com/GoBetterAuth/go-better-auth/models"
)

config := gobetterauthconfig.NewConfig(
  gobetterauthconfig.WithAppName("GoBetterAuthPlayground"),
  gobetterauthconfig.WithBasePath("/api/auth"),
  gobetterauthconfig.WithDatabase(gobetterauthmodels.DatabaseConfig{
    Provider: "postgres",
    URL:      os.Getenv(gobetterauthenv.EnvDatabaseURL),
  }),
  // other config options...
)

auth := gobetterauth.New(gobetterauth.AuthConfig{
  Config:  config,
  Plugins: []gobetterauthmodels.Plugin{
    emailpasswordplugin.New(...),
    // other plugins...
  },
})

http.ListenAndServe(":8080", auth.Handler())
```

You get:

- zero network overhead
- full type safety
- native integration
- maximum performance

---

### Server Mode (Standalone Auth Server)

Run GoBetterAuth as a standalone authentication server via Docker:

```bash
docker run -itd -p 8080:8080 \
  -v $(pwd)/config.toml:/home/appuser/config.toml \
  -e GO_BETTER_AUTH_ADMIN_API_KEY=my-admin-api-key \
  -e GO_BETTER_AUTH_BASE_URL=http://localhost:8080 \
  -e GO_BETTER_AUTH_SECRET=my-app-secret \
  -e GO_BETTER_AUTH_DATABASE_URL=<your_connection_string> \
  # other env vars depending on plugins used...
  ghcr.io/gobetterauth/go-better-auth:latest
```

Use it from any language or framework over HTTP.

Perfect for:

- microservices
- polyglot stacks

---

### 🧠 Design Principles

- Plugin-first architecture
- Clean architecture
- Minimal dependencies
- Standard library first
- Secure by default
- Framework agnostic
- Self-hosted
- Extensible

---

### 🚀 Roadmap

- Many more plugins including Admin, Organizations, RBAC, MFA...
- And more to be announced!

---

### 📜 License

Apache 2.0 License

---

### Docs

For more info and a full guide on how to use this library, check out the [Docs](https://go-better-auth.vercel.app/docs).

---

### Contributing

Your contributions are welcome! Here's how you can get involved:

- If you find a bug, please [submit an issue](https://github.com/GoBetterAuth/go-better-auth/issues).
- Set up your development environment by following our [Contribution Guide](./.github/CONTRIBUTING.md).
- Contribute code by making a [pull request](https://github.com/GoBetterAuth/go-better-auth/) to enhance features, improve user experience, or fix issues.

---

### Support & Community

Join our growing community for support, discussions, and updates:

- [Discord Server](https://discord.gg/nThBksdr2Z)

---
