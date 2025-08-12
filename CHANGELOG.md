# Changelog

## [1.2.0](https://github.com/darksworm/argonaut/compare/v1.1.1...v1.2.0) (2025-08-12)


### Features

* add application control plane namespace support ([c69e6b1](https://github.com/darksworm/argonaut/commit/c69e6b197f2dda4f407c2bc4aa07256e0a3546fc))
* add appNamespace support to ResourceStream ([0e90068](https://github.com/darksworm/argonaut/commit/0e90068dd6968e3861e06c23eeba56d03a9cc922))
* add appNamespace support to sync and rollback APIs ([2663133](https://github.com/darksworm/argonaut/commit/26631339d819d5f43c1f3f472686791c850076b0))
* add vim-style gg and G navigation hotkeys ([dad0314](https://github.com/darksworm/argonaut/commit/dad0314d2c5adf7fc48dd958c7a9be2cdfd99b7c))
* **rollback:** consistent UI/UX with sync ([e5fe7b2](https://github.com/darksworm/argonaut/commit/e5fe7b22b35249bda104a825355c11cffcc87f9f))
* **rollback:** show resources view when rolling back ([c47c1ac](https://github.com/darksworm/argonaut/commit/c47c1ac56ed8d19fe3d677b7e7992ca651e75bc5))

## [1.1.1](https://github.com/darksworm/argonaut/compare/v1.1.0...v1.1.1) (2025-08-12)


### Bug Fixes

* **login:** show AuthRequired component when no server found ([e802ab4](https://github.com/darksworm/argonaut/commit/e802ab45f7a89a9e6f05b81b8a0ef8bb693a42ba))
* **sync:** improve layout of confirm sync popup for better readability ([5d617bf](https://github.com/darksworm/argonaut/commit/5d617bf1659f7f4ab3c23d524c5db25da15ffba3))

## [1.1.0](https://github.com/darksworm/argonaut/compare/v1.0.5...v1.1.0) (2025-08-11)


### Features

* **filter:** allow up/down navigation with arrow keys when filtering ([a795e07](https://github.com/darksworm/argonaut/commit/a795e07a4f5e9eb00e435a25b471b7b7b3f676ed))


### Bug Fixes

* **sync:** correct spacing when syncing multiple apps ([469f309](https://github.com/darksworm/argonaut/commit/469f309981757acf572e85fddd8b1455def94937))

## [1.0.5](https://github.com/darksworm/argonaut/compare/v1.0.4...v1.0.5) (2025-08-11)


### Bug Fixes

* **Banner:** remove unused termRows parameter ([67b7236](https://github.com/darksworm/argonaut/commit/67b72360be48f3c13550db7b7c7c43e87e7b7c3d))

## [1.0.4](https://github.com/darksworm/argonaut/compare/v1.0.3...v1.0.4) (2025-08-11)


### Bug Fixes

* **package:** add prepublish script and specify Node.js engine version ([501bf43](https://github.com/darksworm/argonaut/commit/501bf4377edbb684b5f3ecca3d5b0fd67fabedba))
* **rollup:** rename output file to cli.js ([ebd15fe](https://github.com/darksworm/argonaut/commit/ebd15fe0f2462900b49341acf689e987f0343a44))
* **rollup:** update output format to ESM and add shebang for CLI ([4e9ec59](https://github.com/darksworm/argonaut/commit/4e9ec592a86f312df6afa341864076b5f4bc2725))

## [1.0.3](https://github.com/darksworm/argonaut/compare/v1.0.2...v1.0.3) (2025-08-11)


### Bug Fixes

* **package:** update package name to argonaut-cli ([94a72d0](https://github.com/darksworm/argonaut/commit/94a72d0fe25a55462e1a5e34b81f6233a6717d18))

## [1.0.2](https://github.com/darksworm/argonaut/compare/v1.0.1...v1.0.2) (2025-08-11)


### Bug Fixes

* **import:** correct casing of ArgoNautBanner import ([cecbba3](https://github.com/darksworm/argonaut/commit/cecbba3ed0a8e0fb87759d3d3c8ae09b0c7f9a0b))

## [1.0.1](https://github.com/darksworm/argonaut/compare/v1.0.0...v1.0.1) (2025-08-11)


### Bug Fixes

* **import:** correct casing of Banner import ([c2c9c72](https://github.com/darksworm/argonaut/commit/c2c9c722f6b5dd76e7ab3913ad5f18b5c18a927f))

## 1.0.0 (2025-08-11)


### Features

* **api:** add Argo API layer, hooks, types; integrate with UI ([37b91cb](https://github.com/darksworm/argonaut/commit/37b91cb52ac597f4d8bd0743e4633c8ee8348b45))
* **auth:** add AuthRequiredView for handling authentication prompts ([1c22ace](https://github.com/darksworm/argonaut/commit/1c22ace9492ae20a9da565ad303963722dc7b108))
* **build:** switch from TypeScript compiler to Rollup for building ([24f264d](https://github.com/darksworm/argonaut/commit/24f264d48c093dd7f94c25ad86d27965aac2df97))
* **contexts:** removed context switcher ([2f5ea9f](https://github.com/darksworm/argonaut/commit/2f5ea9f34940b4f9f495c044ae88b410a2b7c9e6))
* **diff:** add managed resource diffs command and external diff viewer ([606df56](https://github.com/darksworm/argonaut/commit/606df56d9dc1b622e4f20864d57fc1b3b65af8bb))
* **diff:** add resource name and namespace to ResourceDiff type and implement git diff check ([88539a9](https://github.com/darksworm/argonaut/commit/88539a975b7c3597b3ca26c4af01159f13f06248))
* **diff:** add resource name and namespace to ResourceDiff type and implement git diff check ([25129c8](https://github.com/darksworm/argonaut/commit/25129c8d60ad60837eca174be105a4985ac303c2))
* **diff:** enhance external diff mode with improved state management and terminal output ([316a85a](https://github.com/darksworm/argonaut/commit/316a85a80b0034e7e2a80bed8f7358395a35d8f5))
* **license:** add GNU General Public License v3 ([149dfaf](https://github.com/darksworm/argonaut/commit/149dfaff69b1b9b6f51f9bcc5dade2dd16b3efce))
* **login:** removed login feature ([79b80f6](https://github.com/darksworm/argonaut/commit/79b80f6fa4aba084dba8985230730960d073c1e8))
* **resource-stream:** add resource streaming component for single-app sync view ([b6e91e0](https://github.com/darksworm/argonaut/commit/b6e91e08e8baa60579655845d6271982991b37b3))
* **resource-stream:** enhance resource streaming with sync status tracking ([2895644](https://github.com/darksworm/argonaut/commit/2895644692978ccdfcf1da3902016ed8ac80b04f))
* **resource-stream:** update resource view with initial data fetch without stream ([16ab4c4](https://github.com/darksworm/argonaut/commit/16ab4c4d8485095c639350da6da95b35318a365c))
* **rollback:** allow apps to be rolled back ([846e75e](https://github.com/darksworm/argonaut/commit/846e75e8da56d49fd79fc1733bd6766adbbd59e1))
* **rollback:** don't allow rolling back to current version ([fbdf7a3](https://github.com/darksworm/argonaut/commit/fbdf7a3334b871dc391e063ebae06babfb2964a4))
* **session:** add user info retrieval function ([4aacc8e](https://github.com/darksworm/argonaut/commit/4aacc8ed0008e8ae892c1570c63c4608c934feb5))
* **sync:** add prune checkbox ([c8b5311](https://github.com/darksworm/argonaut/commit/c8b5311f741312026ddba1a694ecf81e7c1219ce))
* **sync:** cleaned up input ([2a0e35b](https://github.com/darksworm/argonaut/commit/2a0e35b3c257d9dd6357ac1761711d39da878640))
* **sync:** clear selection after syncing multiple apps ([7369937](https://github.com/darksworm/argonaut/commit/736993792b1e19f6b122df641f3b6f57aad5f1bc))
* **ui:** add banner component ([121fda9](https://github.com/darksworm/argonaut/commit/121fda93c2d4b0a64e13d8c570a319a05520d55d))
* **ui:** add filtering and search tweaks ([f093ad4](https://github.com/darksworm/argonaut/commit/f093ad42fd5ed920f5642e60353cfa48d74a64f2))
* **ui:** add initial TUI entrypoint and Node version file ([036e51e](https://github.com/darksworm/argonaut/commit/036e51e77bf8ecb2cf9c5e53e22a2955e4279a9a))
* **ui:** enhance keybindings and view logic ([3eb865c](https://github.com/darksworm/argonaut/commit/3eb865ccb8c25e58cf75d972782167f04ee94502))
* **ui:** expand main screen and interactions ([a2bb68e](https://github.com/darksworm/argonaut/commit/a2bb68ea5aca68a4e599a6dac42667c1776e47cd))
* **ui:** improve navigation and layout ([85a023b](https://github.com/darksworm/argonaut/commit/85a023b74d99b1ec3cecf20ef7dae4169623e056))
* **ui:** improve rendering and truncation ([c039ff7](https://github.com/darksworm/argonaut/commit/c039ff71b4f5b984e0f7eb6b2b19588476043adf))
* **ui:** integrate banner into main app layout ([635127b](https://github.com/darksworm/argonaut/commit/635127b0b3f7b17beee4140cafccceedbc890b50))
* **ui:** iterate on app UI behavior ([986ddea](https://github.com/darksworm/argonaut/commit/986ddea35157d73d32ba770ff7a094d83101d4fd))
* **ui:** polish columns and status line ([efc8108](https://github.com/darksworm/argonaut/commit/efc810868912efebb7caad0ac71f1188f7bb8713))
* **ui:** refine banner and main view; chore(git): update ignore ([914c08f](https://github.com/darksworm/argonaut/commit/914c08f0b892d86e082a62db4868a4485bae22d0))
* **ui:** refine list rendering and inputs ([9ea54b3](https://github.com/darksworm/argonaut/commit/9ea54b3955a9eaae2cccbccba5558fc9fdee9491))


### Bug Fixes

* **api:** improve transport and UI integration ([d4a0673](https://github.com/darksworm/argonaut/commit/d4a067373430dd46c4a880be8752ab7db0940838))
* **apps-slow:** correct application name and improve template syntax ([d6e27e0](https://github.com/darksworm/argonaut/commit/d6e27e0daaf10449dda54faa45d3f76ff35e3f27))
* **diff:** correct argument order in delta command for proper diff display ([91599de](https://github.com/darksworm/argonaut/commit/91599de97bc9a1e0dffdd9bbdd81e3b45a44df32))
* **diff:** improve terminal output for diffs and enhance user experience ([b7e9bd1](https://github.com/darksworm/argonaut/commit/b7e9bd1cf58f1f5aa52dfc540e0e7a5a84a9321c))
* **diff:** streamline delta and git diff handling with improved pager configuration ([4256001](https://github.com/darksworm/argonaut/commit/4256001cff10f992935abf27161f6da883878b8a))
* **index:** adjust HEADER_CONTEXT to reserve space for ASCII logo in wide mode ([0930651](https://github.com/darksworm/argonaut/commit/09306511a387cf47f889959c7bec2dd4079b4f1a))
* **index:** adjust margin for sync applications prompt display ([8ade0de](https://github.com/darksworm/argonaut/commit/8ade0de498092d8d19e0e22f22557d7db539a6bf))
* **index:** streamline confirm sync logic and enhance health label width ([ea745a5](https://github.com/darksworm/argonaut/commit/ea745a5398f167dc80f616b0a512655c77b12382))
* **index:** streamline confirm sync logic and enhance health label width ([f3df375](https://github.com/darksworm/argonaut/commit/f3df375b2a2b81deb5b77676f72585012f5f5aea))
* **resize:** proper display on small width terminals ([23f9e7f](https://github.com/darksworm/argonaut/commit/23f9e7f2252cd3b889d3c7d368f30f90b33c1a6a))
* **resource-stream:** adjust Box properties for consistent layout behavior ([14528ec](https://github.com/darksworm/argonaut/commit/14528ec12bee6b8f46c51f6104cef580bd382656))
* **stream:** add AbortSignal support and cancelation wiring ([04162ce](https://github.com/darksworm/argonaut/commit/04162ce1e49d1bc1fcc420abdccda49ea7aba4a0))
