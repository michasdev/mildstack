# 🖥️ MildStack Desktop

**The Visual Console for your Local Cloud.**

MildStack Desktop is a cross-platform companion app for [MildStack](https://github.com/michasdev/mildstack). It provides an intuitive, production-grade interface to manage, browse, and inspect your local AWS-compatible resources without touching the command line.

<p align="center">
  <img src="../../apps/desktop/resources/screenshot-placeholder.png" alt="MildStack Desktop Interface" width="100%">
</p>

## ✨ Features

- **📂 S3 Explorer**: Browse buckets, navigate prefixes (folders), upload objects, and inspect metadata with a modern file-manager experience.
- **📊 DynamoDB Browser**: Query tables using a rich UI, filter items, edit attributes, and visualize your data structures instantly.
- **📩 SQS Monitor**: Peek at messages, monitor queue depths, and manage Dead Letter Queues (DLQ) with ease.
- **🛠 Instance Management**: Start, stop, and switch between multiple MildStack runtime instances directly from the UI.

## 🏗 Built for Developers

MildStack Desktop is built with a modern stack optimized for performance and safety:

- **Electron & Vite**: Fast startup and smooth transitions.
- **React & TypeScript**: Robust, type-safe UI components.
- **IPC Safety**: Privileged communication with the MildStack core is handled outside the renderer process for maximum security.
- **Clean Architecture**: Decoupled features that allow for rapid extension to new AWS services.

## 🗺 Roadmap

We are continuously adding new capabilities to the desktop experience:

- [x] S3 Bucket & Object Browsing
- [x] DynamoDB Table & Item Exploration
- [x] SQS Message Peeking
- [ ] Lambda Log Stream Monitoring
- [ ] IAM Policy Visualizer
- [ ] CloudFormation Stack Viewer

---

## 🤝 Contributing

We welcome contributions to the desktop app! Whether it's improving the UI, adding new service browsers, or fixing bugs.

1. Fork the repository.
2. Check the [Local Setup Guide](https://github.com/michasdev/mildstack/blob/main/apps/desktop/CONTRIBUTING.md) (coming soon).
3. Submit a PR!

## 📄 License

MIT. Part of the MildStack ecosystem.

---

<p align="center">
  Stop guessing. Start seeing. 🚀
</p>
