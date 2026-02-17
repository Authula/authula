<p align="center">
  <img src="./project-logo.png" height="150" width="150" alt="GoBetterAuth Logo"/>
</p>

<div align="center">
  <a href="https://www.npmjs.com/package/go-better-auth" target="_parent">
    <img src="https://img.shields.io/npm/dm/go-better-auth.svg" alt="Node.js SDK downloads" />
  </a>
  <a href="https://github.com/GoBetterAuth/go-better-auth/stargazers" target="_parent">
    <img src="https://img.shields.io/github/stars/GoBetterAuth/go-better-auth.svg?style=social&label=Star" alt="GitHub stars" />
  </a>
  <a href="https://goreportcard.com/report/github.com/GoBetterAuth/go-better-auth" target="_parent">
    <img src="https://goreportcard.com/badge/github.com/GoBetterAuth/go-better-auth" alt="Go Report Card" />
  </a>
  <a href="https://pkg.go.dev/github.com/GoBetterAuth/go-better-auth/v2" target="_parent">
    <img src="https://pkg.go.dev/badge/github.com/GoBetterAuth/go-better-auth/v2.svg" alt="Go Reference" />
  </a>
  <a href="https://github.com/GoBetterAuth/go-better-auth/blob/main/LICENSE" target="_parent">
    <img src="https://img.shields.io/github/license/GoBetterAuth/go-better-auth.svg" alt="License" />
  </a>
</div>

<div align="center">

### [Become a Sponsor!](https://buy.polar.sh/polar_cl_Q8rpefucf3fmnRTTeIvPCiE6ZvfMKclwxlyOz283ZC7)

</div>

---

### Introduction

✨ Overview

**GoBetterAuth** is an open-source authentication solution that scales with you. Embed it as a library in your Go app, or run it as a standalone auth server with any tech stack. It simplifies adding robust authentication to backend services, empowering developers to build secure applications faster.

All functionality is delivered through a powerful plugin system, allowing you to compose exactly the authentication stack you need — no more, no less, all built with clean architecture. **GoBetterAuth** is flexible enough to integrate with any technology stack. It streamlines the implementation of essential security features through a clean, modular architecture, allowing developers to concentrate on building their applications without the overhead of managing authentication complexities.

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

### Features

GoBetterAuth comes with a variety of plugins that provide essential authentication features out of the box:

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

### Hooks System

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

### Deployment Modes

`Embedded Mode (Go Library)`

Embed GoBetterAuth directly into your Go application:

```go
import (
  gobetterauth "github.com/GoBetterAuth/go-better-auth/v2"
  gobetterauthconfig "github.com/GoBetterAuth/go-better-auth/v2/config"
  gobetterauthmodels "github.com/GoBetterAuth/go-better-auth/v2/models"
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

`Standalone Mode`

Run GoBetterAuth as a standalone authentication server via Docker:

```bash
docker run -itd -p 8080:8080 \
  -v $(pwd)/config.toml:/home/appuser/config.toml \
  -e GO_BETTER_AUTH_BASE_URL=http://localhost:8080 \
  -e GO_BETTER_AUTH_SECRET=my-app-secret \
  -e GO_BETTER_AUTH_DATABASE_URL=<your_connection_string> \
  # other env vars depending on plugins used...
  ghcr.io/gobetterauth/go-better-auth:latest
```

Use it from any language or framework over HTTP.

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

### Docs

For more info and a full guide on how to use this library, check out the [Docs](https://go-better-auth.vercel.app/docs).

---

### SDKs

We provide the following SDKs to facilitate easy integration with GoBetterAuth:

- [Node.js SDK](https://github.com/GoBetterAuth/go-better-auth-node-sdk)

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

### 💎 Support Development

If you'd like to support the ongoing development of this project, consider subscribing on Polar, it means a lot to me!

[![Subscribe on Polar](https://img.shields.io/badge/Subscribe-on%20Polar-00d1ff?style=for-the-badge&logo=polar&logoColor=white)](https://buy.polar.sh/polar_cl_Q8rpefucf3fmnRTTeIvPCiE6ZvfMKclwxlyOz283ZC7)

---

### 💖 Our Sponsors

#### 🏢 Corporate Sponsors

#### 🥇 Gold Sponsors

#### 🥈 Silver Sponsors

#### 🥉 Bronze Sponsors

---
