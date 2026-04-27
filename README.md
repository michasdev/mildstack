<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="apps/desktop/src/renderer/src/assets/logos/mildstack-logo-full-white.png">
    <source media="(prefers-color-scheme: light)" srcset="apps/desktop/src/renderer/src/assets/logos/mildstack-logo-full-black.png">
    <img alt="MildStack Logo" src="apps/desktop/src/renderer/src/assets/logos/mildstack-logo-full-black.png" width="450">
  </picture>
</p>

<p align="center">
  <strong>The Free Drop-in Replacement for LocalStack. Run AWS services locally.</strong><br />
  Fast, lightweight, local-first, and built to run without Docker.
</p>

<p align="center">
  <a href="https://mildstack.dev">Website</a> |
  <a href="https://mildstack.dev/docs">Docs</a> |
  <a href="#supported-services">Supported Services</a> |
  <a href="#quick-start">Quick Start</a>
</p>

---

## What is MildStack?

MildStack is a local-first AWS emulator that helps developers build and test cloud workflows on their own machine, without the need for Docker or containers. It runs natively in Go, does not require Docker, starts in around 200ms (with ~15MB of RAM usage), and keeps the feedback loop short.

It is designed for day-to-day AWS development: point your SDKs and CLI to a local endpoint, keep your state on disk, and use the desktop app to inspect resources visually.

## Why MildStack?

- No Docker or containers required.
- Native runtime built for speed and low overhead.
- Desktop app for browsing and managing local AWS resources.
- Works with official AWS SDKs and the AWS CLI.
- Offline-first development with persistent local state.
- Free and open source under GPL-3.0.

## Supported Services

MildStack currently supports the AWS services that cover the most common local development workflows.

| Service | Status | Notes |
| --- | --- | --- |
| S3 | Active | Buckets, objects, multipart uploads, versioning, metadata, and more |
| DynamoDB | Active | Tables, items, queries, scans, indexes, and batch operations |
| SQS | Active | Standard and FIFO queues, DLQs, visibility timeout, and message operations |
| SNS | Active | Topics, subscriptions, publish flows, and notifications |
| Lambda | In progress | Local function execution is being built now |
| EventBridge | Planned | Event routing and event-driven workflows |
| CloudWatch | Planned | Logs and observability support |

For the full API surface, examples, and service-specific details, visit the docs at [mildstack.dev/docs](https://mildstack.dev/docs).

## Quick Start

Start a MildStack instance:

```bash
mildstack start
# or: mildstack start 8080
# or: mildstack start --detach
```

The runtime starts on your machine in around 200ms. Once it is running, point your app or SDK to `http://localhost:4566`:

```bash
aws s3 mb s3://my-bucket --endpoint-url http://localhost:4566
aws s3 cp ./hello.txt s3://my-bucket/ --endpoint-url http://localhost:4566
```

## AWS CLI

MildStack works with the standard AWS CLI. The simplest setup is to create a local profile and reuse it with `--endpoint-url`.

```bash
aws configure set aws_access_key_id test --profile mildstack
aws configure set aws_secret_access_key test --profile mildstack
aws configure set region us-east-1 --profile mildstack
```

Then use that profile with your commands:

```bash
aws s3 ls --endpoint-url http://localhost:4566 --profile mildstack
aws dynamodb list-tables --endpoint-url http://localhost:4566 --profile mildstack
aws sqs list-queues --endpoint-url http://localhost:4566 --profile mildstack
aws sns list-topics --endpoint-url http://localhost:4566 --profile mildstack
```

If you prefer, create an alias to keep commands short:

```bash
alias awslocal='aws --endpoint-url http://localhost:4566 --profile mildstack'
```

Then the same commands become:

```bash
awslocal s3 ls
awslocal dynamodb list-tables
awslocal sqs list-queues
awslocal sns list-topics
```

MildStack ignores credentials entirely, so any dummy values work as long as the CLI has something to send.

## Ecosystem

MildStack is split into three parts that work together:

### Core Runtime

The Go engine owns AWS emulation logic, persistence, and instance-scoped state.

### MildStack CLI

The terminal interface manages instances, logs, and runtime health.

### Desktop App

The Electron app gives you a visual console to browse buckets, tables, queues, and topics.

## Documentation

The full documentation lives at [mildstack.dev/docs](https://mildstack.dev/docs).

- Getting started - overview, installation, and quick start
- Services - API coverage for each supported AWS service
- MildStack CLI - instance management and AWS Local usage
- Desktop App - browsing and managing local resources

## Contributing

We welcome bug reports and pull requests. If you find a missing feature or unexpected behavior, please open an issue on GitHub and include the steps to reproduce it.

## License

MildStack is released under the GPL-3.0 license. It is free and open source, and you can use, study, modify, and redistribute it under the same license terms. If you distribute modified versions, the GPL requires that they remain under compatible open-source terms.

---

<p align="center">
  MildStack: Run AWS services locally.
</p>
