<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="apps/desktop/src/renderer/src/assets/logos/mildstack-logo-full-white.png">
    <source media="(prefers-color-scheme: light)" srcset="apps/desktop/src/renderer/src/assets/logos/mildstack-logo-full-black.png">
    <img alt="MildStack Logo" src="apps/desktop/src/renderer/src/assets/logos/mildstack-logo-full-black.png" width="450">
  </picture>
</p>

<p align="center">
  <strong>The Lightweight, Drop-in Replacement for LocalStack.</strong><br />
  Fast, Open Source, and Developer-First.
</p>

<p align="center">
  <a href="https://mildstack.dev">Website</a> •
  <a href="#key-features">Key Features</a> •
  <a href="#supported-services">Supported Services</a> •
  <a href="https://discord.gg/your-invite">Community</a>
</p>

---

## ⚡️ What is MildStack?

MildStack is a high-performance, local-first AWS emulator designed to streamline your cloud development workflow. Unlike heavy alternatives that require Docker and significant system resources, MildStack is built in **Go** for maximum efficiency and speed.

Stop waiting for containers to spin up. Start building instantly with a local cloud that feels "mild" on your CPU but "spicy" on productivity.

## ✨ Why MildStack?

- **🚀 Instant-On**: No Docker required. MildStack runs as a native binary, starting in milliseconds.
- **🖥️ Desktop App**: A beautiful, intuitive UI to browse S3 buckets, query DynamoDB tables, and monitor SQS queues without leaving your IDE.
- **🍃 Ultra-Lightweight**: Minimal RAM and CPU footprint. Keep your machine cool while simulating complex cloud architectures.
- **🔌 Drop-in Compatibility**: Works seamlessly with official AWS SDKs and CLI. Just change your endpoint URL.
- **📡 Offline-First**: Build and test your cloud applications on a plane, a train, or anywhere without an internet connection.
- **💰 100% Free**: No "Pro" tiers for basic features. Everything you need for local development, open-source and free.

## 🛠 Supported Services

MildStack is rapidly evolving. We currently provide robust support for core AWS services:

| Service | Status | Features |
| :--- | :--- | :--- |
| **S3** | ✅ Active | Bucket management, Multipart uploads, Metadata support |
| **DynamoDB** | ✅ Active | Tables, GSI/LSI support, Rich querying & filtering |
| **SQS** | ✅ Active | Message queues, DLQ redrive, FIFO support |
| **SNS** |  📅 Planned | Topic publishing, basic subscriptions |
| **Lambda** | 📅 Planned | Local execution of serverless functions |
| **EventBridge** | 📅 Planned | Event-driven architecture simulation |

## 📦 The Ecosystem

MildStack isn't just an emulator; it's a complete development environment:

### 1. The Core Engine
Written in Go, our core provides a high-concurrency, low-latency API that mimics AWS service behavior with precision.

### 2. The MildStack CLI
A modern, terminal-based control center (powered by Charm/BubbleTea) to manage your local instances, view logs, and monitor service health.

### 3. The Desktop Browser
An Electron-powered visual console that gives you a "Production-like" experience for inspecting your local resources. Browse objects, edit items, and peek at messages with ease.

---

## 🗺 Roadmap

Our goal is to cover the 80% of AWS services used in 95% of applications. Check our [Roadmap](https://mildstack.dev/roadmap) to see what's coming next, including Lambda support, IAM simulation, and more.

## 🤝 Contributing

We love contributors! Whether you're fixing a bug, adding a new service, or improving the documentation, your help is welcome.

1. Check out our [Contribution Guidelines](CONTRIBUTING.md).
2. Join our [Discord community](https://discord.gg/your-invite) to discuss ideas.
3. Spread the word! 🌟

## 📄 License

MildStack is released under the **MIT License**. Build freely.

---

<p align="center">
  Built with ❤️ for developers by <a href="https://github.com/michasdev">Michel</a> and the community.
</p>
