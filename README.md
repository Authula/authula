<p align="center">
  <img src="./project-logo.png" height="150" width="150" alt="Project Logo"/>
</p>

<p align="center">
 A comprehensive, framework-agnostic authentication and authorization library for Go.
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

**GoBetterAuth** is an open-source authentication solution that scales with you. Embed it as a library in your Go app, or run it as a standalone auth server for any language or framework. It provides secure email/password authentication, session management, email verification, password reset, and more, all built with clean architecture.

💭 Why GoBetterAuth?

**GoBetterAuth** offers a robust authentication and authorization solution built using Go, yet flexible enough to integrate with any technology stack. It streamlines the implementation of essential security features through a clean, modular architecture, allowing developers to concentrate on building their applications without the overhead of managing authentication complexities.

---

### Features

- 🔑 **Email & Password** – Secure, production-ready authentication with argon2 password hashing. Includes Email Verification, Password Reset and Change Email flows.
- 🌐 **Social OAuth Providers** – Google, GitHub, Discord and more coming soon.
- 💾 **Multiple Database Support** – SQLite, PostgreSQL, MySQL adapters and more coming soon, with migration scripts included.
- 🗄️ **Secondary Storage** – Supports in-memory/database storage and a custom interface to implement Redis and other key-value stores. Use secondary storage to manage session data, rate limiting counters, and other high-frequency records. This enables offloading intensive data to high-performance storage solutions or RAM for optimal scalability and speed.
- 📦 **Minimal Dependencies** – Standard library first, production-ready, and framework-agnostic.
- 🧩 **Comprehensive Configuration** – Flexible, type-safe config with sensible defaults and environment variable support.
- ⚙️ **Flexible Configuration** – Whether you're embedding as a library or running as a server, GoBetterAuth gives you full control over your authentication logic.
- 🛡️ **Enhanced Security** – CSRF protection.
- ⚡ **Rate Limiting** – Configurable rate limiting with secondary storage.
- 🪝 **Hooks System** – Powerful hooks for deep customisation: Database hooks allow developers to hook into DB events like BeforeCreate and AfterCreate on entities, plus create custom ones in plugins. Event hooks execute after key actions, enabling event-driven workflows. Endpoint hooks allow you to integrate into the request/response lifecycle giving you the ability to modify requests/responses.
- 📨 **Event Bus** – Built-in event bus enables event-driven architecture, allowing you to publish and subscribe to authentication and authorization events for seamless integration with external systems and custom workflows.
- 🔌 **Plugin System** – Extensible plugin architecture for custom business logic and routes. Including flexible plugin interface to implement all kinds of plugins.

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

- [Discord Server](https://discord.gg/nThBksdr2Z)

---
