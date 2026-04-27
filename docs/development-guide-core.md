# Core Development Guide

**Part ID:** `core`

## Prerequisites

- Go `1.26.2`
- Access to repository root dependencies (`go.mod`, `go.sum`)

## Local Setup

```bash
cd /Users/michel.freitas/Documents/projects/others/mildstack
go mod download
```

## Build

```bash
go build ./...
```

## Test

```bash
go test ./...
```

## Run CLI Locally

```bash
go run ./core/cmd/mildstack start
```

Optional:

```bash
go run ./core/cmd/mildstack status
go run ./core/cmd/mildstack instances --json
```

## Core Working Conventions

- Keep service logic inside `resources/<service>/application`.
- Keep transport adaptation in `resources/<service>/infrastructure` or `delivery/http`.
- Preserve instance-scoped persistence via `instancepath`.
- Preserve copy-safe shared state publication via runtime hook.

