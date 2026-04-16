# MildStack App

MildStack App is an open source desktop app for browsing and inspecting resources from MildStack, a localstack-like local AWS environment.

The project is currently in its foundation stage. It starts from an electron-vite Electron,
React, and TypeScript application and is being shaped into a local developer console for
MildStack workflows.

## What it is

MildStack App gives developers a desktop UI for local MildStack resources so they do not
need to reach for raw command-line checks for every inspection task.

The version 1 target is local MildStack resource browsing. For version 1, real AWS account
management is not supported for the version 1 local MildStack browsing scope.

## Scope

Current scope:

- Cross-platform Electron desktop app using React and TypeScript.
- Contributor documentation, local setup, and build scripts from the existing project.
- A strict Electron boundary where privileged MildStack communication belongs outside the
  renderer.

Deferred roadmap work, not current setup:

- Tailwind CSS renderer styling.
- CossUI setup through the planned shadcn CLI path.
- Multipage desktop navigation.
- Typed MildStack IPC from renderer to preload to main.
- Generic MildStack resource browsing.
- Dedicated S3 explorer.
- Dedicated DynamoDB explorer.

No current local setup step depends on these deferred items.

## How it works

The app is split across the standard Electron process boundaries:

- `src/main/` owns the Electron main process, native window lifecycle, privileged IPC handlers,
  and future MildStack command coordination.
- `src/preload/` owns the safe bridge exposed to the renderer.
- `src/renderer/src/` owns the React and TypeScript UI that runs in the renderer process.

Renderer code must not import Electron directly or call shell APIs directly. MildStack
communication will be implemented behind Electron main/preload APIs in a later phase, then
exposed to React through a typed renderer-safe contract.

## Local setup

Install dependencies:

```bash
npm install
```

Start the local Electron development app:

```bash
npm run dev
```

Preview the built Electron app:

```bash
npm run start
```

## Available scripts

The current `package.json` scripts are:

- `npm run format` - Format the repository with Prettier.
- `npm run lint` - Run ESLint.
- `npm run typecheck:node` - Typecheck Electron main, preload, and build config code.
- `npm run typecheck:web` - Typecheck the React renderer and preload type declarations.
- `npm run typecheck` - Run node and web typechecks.
- `npm run start` - Preview the Electron app with electron-vite.
- `npm run dev` - Start the Electron development app.
- `npm run build` - Typecheck and build with electron-vite.
- `npm run postinstall` - Install Electron Builder app dependencies.
- `npm run build:unpack` - Build and produce an unpacked Electron Builder output.
- `npm run build:win` - Build a Windows package.
- `npm run build:mac` - Build a macOS package.
- `npm run build:linux` - Build Linux packages.

## Project structure

```text
src/
  main/              Electron main process
  preload/           Renderer-safe preload bridge
  renderer/
    index.html       Renderer HTML shell
    src/             React renderer source
resources/           Desktop app resources
build/               Packaging support files
```

Configuration lives in:

- `electron.vite.config.ts` - Electron Vite main, preload, and renderer build config.
- `tsconfig.node.json` - TypeScript config for Electron-side code.
- `tsconfig.web.json` - TypeScript config for renderer-side code.
- `electron-builder.yml` - Cross-platform packaging config.

## Roadmap

Version 1 is planned as a sequence of foundation and browsing work:

1. Project and renderer foundation: README, renderer structure, and renderer aliases.
2. Tailwind and CossUI setup: planned styling and component foundation.
3. Desktop app shell and navigation: planned multipage app shell.
4. Typed MildStack Electron API: planned renderer to preload to main communication.
5. Generic resource browser: planned local MildStack service and resource inspection.
6. S3 explorer: planned dedicated bucket, prefix, object, and metadata browsing.
7. DynamoDB explorer: planned dedicated table, metadata, and item browsing.

These roadmap items describe planned work, not current completed functionality.

## Contributing

MildStack App is intended to be public open source software. Contributions should preserve
the desktop Electron target, keep renderer code browser-safe, and document only behavior that
exists in the current codebase unless it is clearly marked as planned roadmap work.

Use the current npm scripts for validation before submitting changes:

```bash
npm run typecheck
npm run lint
```
