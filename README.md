<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="apps/desktop/src/renderer/src/assets/logos/mildstack-logo-full-white.png">
    <source media="(prefers-color-scheme: light)" srcset="apps/desktop/src/renderer/src/assets/logos/mildstack-logo-full-black.png">
    <img alt="MildStack Logo" src="apps/desktop/src/renderer/src/assets/logos/mildstack-logo-full-black.png" width="400">
  </picture>
</p>

# MildStack

> A lightweight, local-first AWS emulator built for developers. The best localstack alternative.

MildStack is an open-source project that helps you run and test AWS-like services locally with a focus on speed, simplicity, and low resource usage.

It is designed to be a practical alternative for local cloud development, without unnecessary overhead.

## Why MildStack?

Working with AWS-based applications locally can be slow, heavy, or fragmented.

MildStack aims to make that experience better by being:

- lightweight
- fast
- developer-friendly
- open-source
- easy to extend
- suitable for local development workflows

## What it is

MildStack is being built as a small ecosystem around a core emulator.

The project currently includes:

- a Go-based core for the emulator and API
- a CLI for local control and startup
- a desktop app for a more visual experience and resource browsing
- a web presence for documentation and project info

## Project goals

MildStack is intended to be:

- a local AWS-like emulator
- simple to run and use
- performant and memory-efficient
- modular by design
- consistent across services
- easy to evolve over time

## Tech stack

The project is centered around:

- **Go** for the core runtime, API, and CLI
- **Gin** for the HTTP API
- **Charm** for the terminal UI and CLI experience
- **Electron** for the desktop app
- **React** for the website and docs

## Design principles

The core of the project follows:

- **Domain-Driven Design**
- **Clean Architecture**
- **SOLID principles**

The goal is to keep the emulator core independent from frameworks and easy to maintain as new services are added.

## Philosophy

MildStack is built around a few simple ideas:

- local-first
- developer-first
- performance-oriented
- minimal overhead
- clear architecture
- open-source friendly

## Current status

MildStack is still in early development.

This README is intentionally lightweight and provisional until the project has proper installation docs, usage guides, and service-specific documentation.

## Contributing

Contributions are welcome.

If you want to help, you can:

- suggest services to emulate first
- review architecture decisions
- improve documentation
- test the project locally
- help build core features

## License

MIT.
