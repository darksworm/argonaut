# Repository Guidelines

## Project Structure & Module Organization
- `src/`: TypeScript sources for the Ink/React CLI.
  - `components/` (PascalCase .tsx), `services/`, `commands/`, `state/`, `hooks/`, `api/`, `config/`, `utils/`.
- `src/__tests__/`: Unit/UI tests (`*.test.ts`, `*.ui.test.tsx`).
- `assets/`, `docs/`, `argocd/`: images, docs, example manifests.
- `go-app/`: Go TUI port (Bubble Tea) and migration work.
- `dist/`: build outputs (generated). Do not edit.

## Build, Test, and Development Commands
- Dev: `bun run dev` — runs the CLI in dev mode.
- Build (Node bundle): `bun run build:node` — Rollup bundle to `dist/`.
- Build (binary): `bun run build:binary` — Bun compile to a single executable.
- Lint/Format: `bun run lint` / `bun run lint:fix` / `bun run format`.
- Tests: `bun run test` (watch: `bun run test:watch`). Coverage: `bun run test:coverage`.
- Mutation tests: `bun run test:mutation` (or targeted variants).
- Docker: `bun run docker:build` then `bun run docker:run`.

Requirements: Bun ≥ 1.2.20, Node ≥ 18.

## Coding Style & Naming Conventions
- TypeScript strict mode; 2‑space indentation; double quotes (Biome enforced).
- Files: modules `kebab-case.ts`; React components `PascalCase.tsx`.
- Tests mirror source paths under `src/__tests__/`.
- Run Biome before opening a PR; no unused exports or dead code.

## Testing Guidelines
- Runner: Bun test with React Testing Library and `ink-testing-library`.
- Name tests `*.test.ts` and UI tests `*.ui.test.tsx`.
- Aim for meaningful coverage on services and command logic; CI uses lcov.
- Example: `bun test src/__tests__/services/status-service.test.ts`.

## Commit & Pull Request Guidelines
- Conventional Commits: `feat:`, `fix:`, `docs:`, `chore:`, etc. (see git log).
- Branch per change; keep commits focused and descriptive.
- PRs include: summary of what/why, linked issues, test evidence (output or screenshots), and docs updates when relevant.

## Security & Configuration Tips
- ArgoCD config path: `ARGOCD_CONFIG` or `${XDG_CONFIG_HOME}/argocd/config`.
- Do not commit tokens or real cluster data; use mock fixtures in tests.
- License file is generated: `bun run generate:licenses` (writes to `dist/`).
