# Deployment and Delivery Guide

## Overview

This repository currently uses GitHub Actions for CI/build/release/deploy flows.

## Workflows

## `core-go-ci.yml`

- Trigger: PRs to `main` affecting `core/**`
- Actions:
  - setup Go 1.26.2
  - `go mod download`
  - `go test ./...`
  - `go build ./...`

## `desktop-build.yml`

- Trigger: PRs to `main` affecting desktop/core workflow paths
- Actions:
  - build multi-arch CLI binaries from core
  - build desktop packages for macOS, Windows, Linux
  - upload build artifacts

## `mildstack-release.yml`

- Trigger: tag push `v*` or manual dispatch
- Actions:
  - build CLI binaries
  - build desktop distributables per platform
  - optionally publish curated GitHub release assets

## `deploy-mildstack-dev.yml`

- Trigger: pushes to `main` affecting website paths or manual dispatch
- Actions:
  - build `apps/web/www`
  - publish static output to GitHub Pages (`mildstack.dev`)

## Release Packaging Dependencies

Desktop release pipeline depends on core CLI binaries generated during workflow jobs and injected into desktop resources before final packaging.

## Operational Notes

- Core remains a local runtime executable and is not currently deployed as a long-running hosted backend in this repository.
- Website and desktop have independent pipeline concerns; core compatibility must remain stable to avoid desktop runtime regressions.

