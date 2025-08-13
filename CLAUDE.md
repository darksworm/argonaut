# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## About the Project

Argonaut is a terminal UI (TUI) for Argo CD built with React + Ink. It provides a keyboard-first interface for managing Argo CD applications, similar to how k9s works for Kubernetes.

## Development Commands

- **Development server**: `npm run dev` - Runs the app directly with tsx
- **Build**: `npm run build` - Creates production build using Rollup
- **Start built app**: `npm start` - Runs the built CLI from dist/index.js
- **Prepare for publish**: `npm run prepublishOnly` - Builds before publishing

## Architecture Overview

### Core Structure
- **Entry point**: `src/main.tsx` - Sets up alternate screen mode and renders the React app
- **Main component**: `src/components/App.tsx` - Central state management and UI orchestration
- **API layer**: `src/api/` - Handles communication with Argo CD server
- **State management**: React hooks with local state, no external state library
- **Transport**: `src/api/transport.ts` - Common HTTP client for Argo CD API calls

### Key Components
- **App.tsx**: Main application state, navigation, and command handling
- **useApps hook**: Real-time application data via NDJSON streaming
- **ResourceStream**: Live resource status view during syncs and rollbacks
- **Rollback**: Guided rollback flow with revision selection
- **ConfirmationBox**: Reusable y/n confirmation component with options
- **DiffView**: External diff integration (prefers delta, falls back to git)

### Navigation Model
The app uses a hierarchical drilling model:
1. **Clusters** → **Namespaces** → **Projects** → **Apps**
2. Command palette (`:`) for actions: sync, diff, rollback, resources
3. Vim-like navigation (j/k, space to select, enter to drill down)

### API Integration
- Uses Argo CD CLI config for authentication (`~/.config/argocd/config`)
- Streams live updates via NDJSON endpoints (`/api/v1/stream/applications`)
- Token-based authentication with automatic validation
- Robust error handling for auth failures and network issues

### External Dependencies
- **Argo CD CLI**: Required for authentication setup
- **Delta**: Optional, preferred for enhanced diffs (falls back to git)
- **React + Ink**: Terminal UI framework
- **Node-pty**: Terminal process management for external commands

### Build Process
- **Rollup** with TypeScript plugin
- Bundles to single ESM file with shebang for CLI execution
- External dependencies are not bundled (node_modules required at runtime)
- Output: `dist/cli.js` (matches package.json bin entry)

## Important Patterns

### Streaming Data
The app maintains live connections to Argo CD using async generators that yield NDJSON events. The `useApps` hook manages this streaming with proper cleanup and error handling.

### Mode Management
The app uses a mode-based state system:
- `normal`: Standard navigation and selection
- `search`: Live filtering with immediate visual feedback
- `command`: Command palette input
- `external`: Pauses React rendering during external diff sessions
- `resources`: Full-screen resource monitoring during syncs

### Error Boundaries
Authentication errors bubble up to clear tokens and show re-auth prompts. Network errors are surfaced contextually without breaking the entire interface.