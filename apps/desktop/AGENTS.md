<!-- GSD:project-start source:PROJECT.md -->
## Project

**MildStack App**

MildStack App is a cross-platform desktop application for browsing and inspecting resources exposed by MildStack, a localstack-like local AWS environment. It uses Electron, React, and TypeScript to provide a frontend resource browser while keeping MildStack communication behind the Electron layer.

The app starts from an electron-vite starter and will evolve into an open source desktop console for local AWS workflows, with dedicated explorer experiences for S3 and DynamoDB.

**Core Value:** Developers can inspect and navigate MildStack resources locally through a reliable desktop UI without reaching through raw AWS CLI commands for every task.

### Constraints

- **Tech stack**: Electron, React, TypeScript, electron-vite, and npm remain the foundation — this is the existing working stack.
- **Renderer architecture**: Feature-based renderer modules should use kebab-case file and folder names — this is an explicit project convention.
- **Security boundary**: Renderer code must not import Electron or shell out directly — MildStack communication belongs in Electron main/preload APIs.
- **UI system**: CossUI should be installed through shadcn CLI and used according to `.agents/skills/coss/` guidance — this avoids inventing component APIs.
- **Styling**: Tailwind setup must align with CossUI expectations — Coss guidance assumes Tailwind CSS v4.
- **Platform**: Windows, macOS, and Linux packaging should remain supported — existing Electron Builder scripts already target all three.
- **Open source**: README and project documentation should be clear enough for outside contributors — the repository is intended to be public.
<!-- GSD:project-end -->

<!-- GSD:stack-start source:codebase/STACK.md -->
## Technology Stack

## Languages
- TypeScript 5.9.3 - Main process code in `src/main/index.ts`, preload code in `src/preload/index.ts`, renderer code in `src/renderer/src/main.tsx`, and build configuration in `electron.vite.config.ts`.
- TSX/React JSX - Renderer UI in `src/renderer/src/App.tsx` and `src/renderer/src/components/Versions.tsx`.
- HTML - Renderer shell in `src/renderer/index.html`.
- CSS - Renderer styling in `src/renderer/src/assets/base.css` and `src/renderer/src/assets/main.css`.
- YAML - Formatter/build/update configuration in `.prettierrc.yaml`, `electron-builder.yml`, and `dev-app-update.yml`.
- XML property list - macOS entitlements in `build/entitlements.mac.plist`.
## Runtime
- Electron 39.2.6 - Desktop runtime declared in `package.json` and used through Electron main/preload APIs in `src/main/index.ts` and `src/preload/index.ts`.
- Node.js - Local toolchain detected as Node v20.19.5 via `node -v`; no committed Node version file such as `.nvmrc` or `.node-version` is present.
- Chromium - Renderer runtime is provided by Electron; `src/renderer/src/components/Versions.tsx` displays `window.electron.process.versions.chrome`.
- npm 10.8.2 - Local toolchain detected via `npm -v`.
- Lockfile: present at `package-lock.json` with lockfileVersion 3.
- npm configuration: `.npmrc` is present and should be treated as secret-bearing package manager configuration; do not read or quote its contents.
## Frameworks
- Electron 39.2.6 - Cross-platform desktop app shell; `package.json` sets `"main": "./out/main/index.js"` and `src/main/index.ts` creates the `BrowserWindow`.
- electron-vite 5.0.0 - Development server and build pipeline used by `npm run dev`, `npm run start`, and `npm run build` in `package.json`.
- React 19.2.1 - Renderer UI framework used in `src/renderer/src/main.tsx`, `src/renderer/src/App.tsx`, and `src/renderer/src/components/Versions.tsx`.
- React DOM 19.2.1 - Renderer mount API used in `src/renderer/src/main.tsx`.
- Not detected - `package.json` has no test scripts and no test framework dependency is declared.
- Vite 7.2.6 - Bundler used under electron-vite; renderer config lives in `electron.vite.config.ts`.
- @vitejs/plugin-react 5.1.1 - React plugin configured in `electron.vite.config.ts`.
- electron-builder 26.0.12 - Packaging tool used by `build:unpack`, `build:win`, `build:mac`, and `build:linux` scripts in `package.json`; packaging config lives in `electron-builder.yml`.
- TypeScript compiler 5.9.3 - Typechecking uses `tsconfig.node.json` and `tsconfig.web.json` via scripts in `package.json`.
- ESLint 9.39.1 - Linting uses `eslint.config.mjs` and the `npm run lint` script in `package.json`.
- Prettier 3.7.4 - Formatting uses `.prettierrc.yaml` and the `npm run format` script in `package.json`.
## Key Dependencies
- `electron` ^39.2.6 - Required desktop runtime and main/preload API provider.
- `react` ^19.2.1 - Renderer UI component model.
- `react-dom` ^19.2.1 - Renderer root creation in `src/renderer/src/main.tsx`.
- `electron-vite` ^5.0.0 - Provides Electron-specific dev, preview, and production build commands.
- `@electron-toolkit/preload` ^3.0.2 - Exposes `electronAPI` through `contextBridge` in `src/preload/index.ts`.
- `@electron-toolkit/utils` ^4.0.0 - Provides `electronApp`, `optimizer`, and `is` helpers in `src/main/index.ts`.
- `electron-updater` ^6.3.9 - Auto-update support dependency declared in `package.json`; update endpoints are configured in `electron-builder.yml` and `dev-app-update.yml`.
- `electron-builder` ^26.0.12 - Produces Windows, macOS, and Linux artifacts from `electron-builder.yml`.
- `@electron-toolkit/tsconfig` ^2.0.0 - Base TypeScript configs extended by `tsconfig.node.json` and `tsconfig.web.json`.
- `@electron-toolkit/eslint-config-ts` ^3.1.0 - TypeScript ESLint preset imported by `eslint.config.mjs`.
- `@electron-toolkit/eslint-config-prettier` ^3.0.0 - Prettier compatibility preset imported by `eslint.config.mjs`.
- `eslint-plugin-react` ^7.37.5, `eslint-plugin-react-hooks` ^7.0.1, and `eslint-plugin-react-refresh` ^0.4.24 - React lint rules configured in `eslint.config.mjs`.
## Configuration
- Development renderer URL is read from `process.env['ELECTRON_RENDERER_URL']` in `src/main/index.ts`; when present in development, the main window loads that remote dev URL.
- Production renderer loads the bundled file at `../renderer/index.html` from `src/main/index.ts`.
- Renderer security policy is set through a Content-Security-Policy meta tag in `src/renderer/index.html`.
- No `.env` or `.env.*` files were detected by filename scan.
- `package.json` defines npm scripts for formatting, linting, node/web typechecking, dev, preview, build, packaging, and platform packaging.
- `electron.vite.config.ts` defines main, preload, and renderer build entries and configures the `@renderer/*` alias to `src/renderer/src/*`.
- `tsconfig.json` is a project-reference root for `tsconfig.node.json` and `tsconfig.web.json`.
- `tsconfig.node.json` includes `electron.vite.config.*`, `src/main/**/*`, and `src/preload/**/*`.
- `tsconfig.web.json` includes `src/renderer/src/**/*` and `src/preload/*.d.ts`, enables `jsx: react-jsx`, and defines the `@renderer/*` TypeScript path alias.
- `eslint.config.mjs` ignores `node_modules`, `dist`, and `out`; applies Electron Toolkit TypeScript rules, React rules, React Hooks rules, React Refresh Vite rules, and Prettier compatibility.
- `.prettierrc.yaml` sets single quotes, no semicolons, print width 100, and no trailing commas.
- `.editorconfig` sets UTF-8, LF endings, 2-space indentation, final newline insertion, and trailing whitespace trimming.
- `electron-builder.yml` configures app identity, packaged files, platform artifact names, macOS entitlements, Linux package targets, generic publishing, and Electron download mirror.
## Platform Requirements
- Use npm with `package-lock.json`; install with `npm install` as documented in `README.md`.
- Run local development with `npm run dev` from `package.json`.
- Run typechecking with `npm run typecheck`; it executes `typecheck:node` against `tsconfig.node.json` and `typecheck:web` against `tsconfig.web.json`.
- Use VS Code with ESLint and Prettier extensions as recommended in `README.md`; workspace settings exist under `.vscode/`.
- Deployment target is packaged desktop artifacts generated by electron-builder.
- Windows packaging uses NSIS from `electron-builder.yml` and `npm run build:win`.
- macOS packaging uses `build/entitlements.mac.plist`, disables notarization in `electron-builder.yml`, and runs through `npm run build:mac`.
- Linux packaging targets AppImage, snap, and deb in `electron-builder.yml` and runs through `npm run build:linux`.
- Auto-update publishing is configured as generic provider at `https://example.com/auto-updates` in `electron-builder.yml` and `dev-app-update.yml`.
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

## Naming Patterns
- Use PascalCase for React component files in `src/renderer/src/`, such as `src/renderer/src/App.tsx` and `src/renderer/src/components/Versions.tsx`.
- Use lowercase descriptive names for application entry files, such as `src/main/index.ts`, `src/preload/index.ts`, and `src/renderer/src/main.tsx`.
- Use lowercase asset stylesheet names in `src/renderer/src/assets/`, such as `src/renderer/src/assets/base.css` and `src/renderer/src/assets/main.css`.
- Use declaration files for global type augmentation and environment types: `src/preload/index.d.ts` and `src/renderer/src/env.d.ts`.
- Use PascalCase function declarations for React components returning `React.JSX.Element`, as in `function App(): React.JSX.Element` in `src/renderer/src/App.tsx` and `function Versions(): React.JSX.Element` in `src/renderer/src/components/Versions.tsx`.
- Use camelCase function declarations for non-component helpers, as in `createWindow()` in `src/main/index.ts`.
- Use camelCase const arrow functions for local handlers, as in `ipcHandle` in `src/renderer/src/App.tsx`.
- Annotate void-returning functions explicitly when the function body performs side effects, as in `createWindow(): void` in `src/main/index.ts` and `ipcHandle = (): void` in `src/renderer/src/App.tsx`.
- Use camelCase for local variables and state values, such as `mainWindow` in `src/main/index.ts`, `electronLogo` in `src/renderer/src/App.tsx`, and `versions` in `src/renderer/src/components/Versions.tsx`.
- Use uppercase package/tool names only when imported APIs require them. React component types use `React.JSX.Element` in `src/renderer/src/App.tsx` and `src/renderer/src/components/Versions.tsx`.
- Use global interface augmentation for renderer window APIs in `src/preload/index.d.ts`.
- Use `unknown` for intentionally untyped exposed APIs, as in `api: unknown` in `src/preload/index.d.ts`.
- Use the Vite client triple-slash reference in `src/renderer/src/env.d.ts`.
## Code Style
- Use Prettier via `npm run format` from `package.json`.
- Prettier settings are declared in `.prettierrc.yaml`: single quotes, no semicolons, `printWidth: 100`, and no trailing commas.
- Keep imports and statements semicolon-free in TypeScript and TSX files, matching `src/main/index.ts`, `src/preload/index.ts`, `src/renderer/src/main.tsx`, `src/renderer/src/App.tsx`, and `src/renderer/src/components/Versions.tsx`.
- Use two-space indentation in TypeScript, TSX, config files, and CSS, matching `eslint.config.mjs`, `electron.vite.config.ts`, and `src/renderer/src/assets/main.css`.
- Use ESLint via `npm run lint` from `package.json`.
- ESLint configuration lives in `eslint.config.mjs`.
- ESLint ignores `**/node_modules`, `**/dist`, and `**/out` via `eslint.config.mjs`.
- Use `@electron-toolkit/eslint-config-ts` recommended rules, React flat recommended rules, React JSX runtime rules, React Hooks recommended rules, React Refresh Vite rules, and `@electron-toolkit/eslint-config-prettier` from `eslint.config.mjs`.
## Import Organization
- Use `@renderer/*` for renderer source imports when a non-relative path is clearer. The TypeScript alias is configured in `tsconfig.web.json`; the Vite/Electron alias is configured in `electron.vite.config.ts`.
- Relative imports are currently used for nearby files, such as `./App`, `./components/Versions`, and `./assets/electron.svg` in `src/renderer/src/main.tsx` and `src/renderer/src/App.tsx`.
## Error Handling
- Wrap preload bridge exposure in `try`/`catch` when `process.contextIsolated` is true, as in `src/preload/index.ts`.
- Log preload bridge exposure failures with `console.error(error)` in `src/preload/index.ts`.
- Return explicit Electron handler results where required by the API, such as `{ action: 'deny' }` from `mainWindow.webContents.setWindowOpenHandler` in `src/main/index.ts`.
- Use platform guards for OS-specific behavior, such as Linux icon selection and macOS window lifecycle handling in `src/main/index.ts`.
## Logging
- Use `console.error` for caught errors in preload setup, as in `src/preload/index.ts`.
- Use `console.log` only for scaffold/debug IPC behavior, as in the `ipcMain.on('ping')` handler in `src/main/index.ts`.
- No project-level logging abstraction is present.
## Comments
- Keep comments short and explanatory around Electron lifecycle and scaffold behavior, matching `src/main/index.ts` and `src/preload/index.ts`.
- Avoid commenting self-explanatory React rendering and simple state access, matching `src/renderer/src/App.tsx` and `src/renderer/src/components/Versions.tsx`.
- JSDoc/TSDoc is not used in the current source files.
- Prefer explicit TypeScript annotations for exported or lifecycle-sensitive functions instead of JSDoc when following current patterns, such as `createWindow(): void` in `src/main/index.ts`.
## Function Design
## Module Design
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

## Pattern Overview
- Keep Electron lifecycle, native window creation, app-level process behavior, and privileged IPC listeners in `src/main/index.ts`.
- Keep renderer-facing native APIs behind the preload boundary in `src/preload/index.ts`, with global TypeScript typing in `src/preload/index.d.ts`.
- Keep browser UI code under `src/renderer/src/`, rendered from `src/renderer/src/main.tsx` into `src/renderer/index.html`.
- Use `electron-vite` to build separate main, preload, and renderer bundles from `electron.vite.config.ts`.
- Use `electron-builder` packaging rules from `electron-builder.yml`; production runtime points to the compiled main entry `out/main/index.js` through `package.json`.
## Layers
- Purpose: Own Electron application lifecycle, `BrowserWindow` creation, OS integration, production/development renderer loading, and privileged IPC handling.
- Location: `src/main/index.ts`
- Contains: `createWindow()`, `app.whenReady()`, `app.on('activate')`, `app.on('window-all-closed')`, `ipcMain.on('ping')`, external-link interception through `mainWindow.webContents.setWindowOpenHandler()`.
- Depends on: `electron`, `path`, `@electron-toolkit/utils`, and the bundled asset import `resources/icon.png?asset`.
- Used by: Electron runtime via `package.json` `main` pointing at `out/main/index.js` after `electron-vite` build.
- Add new privileged OS, filesystem, native menu, application lifecycle, and IPC handler code here or in modules imported from this layer.
- Purpose: Provide the only direct bridge from the isolated renderer world to selected Electron APIs and custom app APIs.
- Location: `src/preload/index.ts`
- Contains: `contextBridge.exposeInMainWorld('electron', electronAPI)`, `contextBridge.exposeInMainWorld('api', api)`, and a fallback assignment path for non-isolated contexts.
- Depends on: `electron` `contextBridge` and `@electron-toolkit/preload` `electronAPI`.
- Used by: `BrowserWindow` in `src/main/index.ts` through `webPreferences.preload: join(__dirname, '../preload/index.js')`, and by renderer code through `window.electron` / `window.api`.
- Add new renderer-safe native capabilities by adding typed methods to the `api` object in `src/preload/index.ts` and the matching global shape in `src/preload/index.d.ts`.
- Purpose: Type the renderer-visible globals exposed by the preload bridge.
- Location: `src/preload/index.d.ts`
- Contains: `Window.electron: ElectronAPI` and `Window.api: unknown`.
- Depends on: `@electron-toolkit/preload`.
- Used by: Web TypeScript compilation through `tsconfig.web.json` include pattern `src/preload/*.d.ts`.
- Keep this file synchronized with `src/preload/index.ts`; do not add renderer uses of `window.api` without typing the actual shape here.
- Purpose: Host browser document markup and mount the React application.
- Location: `src/renderer/index.html` and `src/renderer/src/main.tsx`
- Contains: CSP meta tag, `<div id="root"></div>`, module script loading `/src/main.tsx`, React `createRoot()`, `StrictMode`, global CSS import `src/renderer/src/assets/main.css`, and root `App` rendering.
- Depends on: React, React DOM, Vite module loading, and local CSS/assets.
- Used by: `BrowserWindow.loadURL()` during development and `BrowserWindow.loadFile(join(__dirname, '../renderer/index.html'))` during production in `src/main/index.ts`.
- Add renderer-wide providers, routing, and app bootstrapping in `src/renderer/src/main.tsx`; keep static HTML concerns in `src/renderer/index.html`.
- Purpose: Implement user-facing UI with React function components.
- Location: `src/renderer/src/App.tsx` and `src/renderer/src/components/Versions.tsx`
- Contains: Root UI composition in `App`, an IPC trigger using `window.electron.ipcRenderer.send('ping')`, and a `Versions` component reading `window.electron.process.versions` into React state.
- Depends on: React JSX runtime, `window.electron` from preload, local assets from `src/renderer/src/assets/`, and component imports from `src/renderer/src/components/`.
- Used by: `src/renderer/src/main.tsx`.
- Add new UI components under `src/renderer/src/components/` and compose them from `src/renderer/src/App.tsx` or future feature-level renderer modules.
- Purpose: Provide CSS and static image assets for renderer UI.
- Location: `src/renderer/src/assets/`
- Contains: `src/renderer/src/assets/main.css`, `src/renderer/src/assets/base.css`, `src/renderer/src/assets/electron.svg`, and `src/renderer/src/assets/wavy-lines.svg`.
- Depends on: Vite asset handling and CSS bundling.
- Used by: `src/renderer/src/main.tsx` and `src/renderer/src/App.tsx`.
- Put renderer-only CSS and images here; put app/package resources used by Electron or packaging in `resources/` or `build/` instead.
- Purpose: Define compilation boundaries, renderer aliases, package outputs, and platform packaging behavior.
- Location: `electron.vite.config.ts`, `tsconfig.json`, `tsconfig.node.json`, `tsconfig.web.json`, `electron-builder.yml`, and `package.json`.
- Contains: `electron-vite` main/preload/renderer config, React Vite plugin setup, `@renderer/*` alias, TypeScript project references, npm scripts, Electron Builder file filters, platform targets, and app metadata.
- Depends on: `electron-vite`, Vite, TypeScript, Electron Builder, React plugin, and Electron Toolkit configs.
- Used by: npm scripts in `package.json`, development preview/runtime commands, and distributable packaging commands.
- Update this layer when adding path aliases, changing build entry boundaries, changing packaged resources, or changing desktop distribution targets.
## Data Flow
- Renderer state is local React component state only; `src/renderer/src/components/Versions.tsx` uses `useState(window.electron.process.versions)`.
- No global state library, client-side routing, persistence layer, or domain data store is present.
- Main-process state is limited to Electron lifecycle listeners and the active `BrowserWindow` local variable inside `createWindow()`.
- Add shared or persistent state deliberately; keep renderer state in React until a concrete cross-component or persistence requirement exists.
## Key Abstractions
- Purpose: Centralize native window creation and renderer loading behavior.
- Examples: `src/main/index.ts`
- Pattern: `createWindow(): void` creates one `BrowserWindow`, wires events, configures preload, and switches between dev URL and production HTML.
- Extend this function or split imported helpers from `src/main/` when adding menus, additional windows, or window-level services.
- Purpose: Constrain the renderer's access to native capabilities.
- Examples: `src/preload/index.ts`, `src/preload/index.d.ts`
- Pattern: `contextBridge.exposeInMainWorld()` exposes `window.electron` and `window.api`; TypeScript declares those globals.
- Put custom app-specific methods on `window.api`, not directly in React components with ad hoc Electron imports.
- Purpose: Compose top-level UI and renderer actions.
- Examples: `src/renderer/src/App.tsx`
- Pattern: Function component returning JSX, imported by `src/renderer/src/main.tsx`, with local handler functions for UI events.
- Keep cross-cutting renderer providers in `src/renderer/src/main.tsx`; keep app view composition in `src/renderer/src/App.tsx`.
- Purpose: Encapsulate small pieces of UI behavior.
- Examples: `src/renderer/src/components/Versions.tsx`
- Pattern: Default-exported React function component with local `useState` state.
- Place reusable renderer components in `src/renderer/src/components/` and import them with relative paths or `@renderer/*`.
- Purpose: Compile main, preload, and renderer as separate targets.
- Examples: `electron.vite.config.ts`, `tsconfig.node.json`, `tsconfig.web.json`
- Pattern: Empty `main` and `preload` build sections use default conventions; renderer config adds the React plugin and `@renderer` alias.
- Keep Node/Electron-only code inside files included by `tsconfig.node.json`; keep DOM/React code inside files included by `tsconfig.web.json`.
## Entry Points
- Location: `src/main/index.ts`
- Triggers: Electron runtime starts compiled `out/main/index.js` configured by `package.json`.
- Responsibilities: App readiness, window creation, app activation behavior, quit behavior, shortcut behavior, external URL handling, and IPC listener registration.
- Location: `src/preload/index.ts`
- Triggers: `BrowserWindow` `webPreferences.preload` configured in `src/main/index.ts`.
- Responsibilities: Expose safe renderer globals with `contextBridge`, or attach globals directly when context isolation is disabled.
- Location: `src/preload/index.d.ts`
- Triggers: Included by `tsconfig.web.json` during web typechecking.
- Responsibilities: Define `Window.electron` and `Window.api` for renderer TypeScript.
- Location: `src/renderer/index.html`
- Triggers: Loaded by dev server during `npm run dev`, and loaded from packaged output through `mainWindow.loadFile()` in production.
- Responsibilities: Document metadata, CSP, root DOM node, and Vite module script.
- Location: `src/renderer/src/main.tsx`
- Triggers: Browser loads `/src/main.tsx` from `src/renderer/index.html`.
- Responsibilities: Import global CSS, create the React root, enable `StrictMode`, and render `App`.
- Location: `src/renderer/src/App.tsx`
- Triggers: Rendered by `src/renderer/src/main.tsx`.
- Responsibilities: Compose starter UI, render `Versions`, and send the example `ping` IPC message.
- Location: `electron.vite.config.ts`
- Triggers: `electron-vite dev`, `electron-vite preview`, and `electron-vite build` scripts in `package.json`.
- Responsibilities: Define Electron Vite build targets, renderer aliasing, and React plugin usage.
- Location: `electron-builder.yml`
- Triggers: `electron-builder` scripts in `package.json`.
- Responsibilities: Define app ID, product name, packaged files, platform targets, icons/resources, and publishing metadata.
## Error Handling
- Catch and log preload bridge exposure failures in `src/preload/index.ts` with `try` / `catch (error) { console.error(error) }`.
- Prevent untrusted or unintended new Electron windows in `src/main/index.ts` by opening URLs externally with `shell.openExternal()` and returning `{ action: 'deny' }`.
- Keep app quit behavior explicit in `src/main/index.ts`: quit on `window-all-closed` except on macOS.
- No centralized domain error model, renderer error boundary, IPC request/response error wrapper, or logging service is present.
- Add renderer error boundaries and typed IPC result/error contracts before introducing complex workflows or recoverable background operations.
## Cross-Cutting Concerns
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->
## Project Skills

| Skill | Description | Path |
|-------|-------------|------|
| coss | Helps implement coss UI components correctly. Use when building UIs with coss primitives (buttons, dialogs, selects, forms, menus, tabs, inputs, toasts, etc.), migrating from shadcn/Radix to coss/Base UI, composing trigger-based overlays, or troubleshooting coss component behavior. Covers imports, accessibility, Tailwind styling, and common pitfalls. | `.agents/skills/coss/SKILL.md` |
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->



<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
