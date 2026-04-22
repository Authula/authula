<p align="center">
  <img src="./project-logo.png" height="100" width="250" alt="Authula Logo"/>
</p>

<div align="center">
  <a href="https://github.com/Authula/authula/stargazers" target="_parent">
    <img src="https://img.shields.io/github/stars/Authula/authula.svg?style=social&label=Star" alt="GitHub stars" />
  </a>
  <a href="https://goreportcard.com/report/github.com/Authula/authula" target="_parent">
    <img src="https://goreportcard.com/badge/github.com/Authula/authula" alt="Go Report Card" />
  </a>
  <a href="https://pkg.go.dev/github.com/Authula/authula" target="_parent">
    <img src="https://pkg.go.dev/badge/github.com/Authula/authula.svg" alt="Go Reference" />
  </a>
  <a href="https://github.com/Authula/authula/blob/main/LICENSE" target="_parent">
    <img src="https://img.shields.io/github/license/Authula/authula.svg" alt="License" />
  </a>
</div>

<div align="center">

### [Become a Sponsor!](https://buy.polar.sh/polar_cl_Q8rpefucf3fmnRTTeIvPCiE6ZvfMKclwxlyOz283ZC7)

</div>

---

### Overview

**Authula** is an open-source authentication solution that scales with you. Embed it as a library in your Go app, or run it as a standalone auth server with any tech stack. It simplifies adding robust authentication to backend services, empowering developers to build secure applications faster.

All functionality is delivered through a powerful plugin system, allowing you to compose exactly the authentication stack you need — no more, no less, all built with clean architecture. **Authula** is flexible enough to integrate with any technology stack. It streamlines the implementation of essential security features through a clean, modular architecture, allowing developers to concentrate on building their applications without the overhead of managing authentication complexities.

---

### Who is it for?

Authula is ideal for:

- Startups that want full control over their authentication stack
- Teams building microservices or multi-backend systems
- Companies with self-hosting or compliance requirements
- Go developers who want first-class embedded auth
- Anyone who wants modern auth without SaaS lock-in and full extensibility

---

### 🧩 Plugins & Capabilities

Authula is architected around a powerful plugin and capability system.

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

Authula comes with a variety of plugins that provide essential authentication features out of the box:

- 📧 Email & Password: Email-based flows for registration, login, email verification, password reset & email change.
- 🌐 OAuth providers
- 🔐 TOTP: Authenticator app support, backup codes, trusted devices for two-factor authentication
- 💾 Multiple database backends (SQLite, PostgreSQL, MySQL)
- 🗄️ Secondary storage (In-memory, DB, Redis)
- ⚡ Rate limiting
- 🛡️ CSRF protection
- 🪝 Hooks system
- 📨 Event bus
- 🧩 Custom routes and logic

---

### Hooks System

Authula includes a powerful, lifecycle-based hooks system that lets you intercept and customize request handling at every stage of the HTTP request/response cycle.

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

### Quick Start

[Get Started with Authula](https://authula.vercel.app/docs/get-started/installation)

---

### Docs

For more info and a full guide on how to use this library, check out the [Docs](https://authula.vercel.app/docs).

---

### SDKs

We provide the following SDKs to facilitate easy integration with Authula:

- [Node.js SDK](https://github.com/Authula/authula-node-sdk)

---

### Contributing

Your contributions are welcome! Here's how you can get involved:

- If you find a bug, please [submit an issue](https://github.com/Authula/authula/issues).
- Set up your development environment by following our [Contribution Guide](./.github/CONTRIBUTING.md).
- Contribute code by making a [pull request](https://github.com/Authula/authula/) to enhance features, improve user experience, or fix issues.

[![Star History Chart](https://api.star-history.com/svg?repos=Authula/authula&type=date&legend=top-left)](https://www.star-history.com/#Authula/authula&type=date&legend=top-left)

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

<a href="https://github.com/libanj"><img src="https://wsrv.nl/?url=github.com/libanj.png?w=64&h=64&mask=circle" width="32" height="32"></a>

---
