# Changelog

## [1.13.0](https://github.com/darksworm/argonaut/compare/v1.12.0...v1.13.0) (2025-08-27)


### Features

* add centralized AppStateContext for state management ([0b58e89](https://github.com/darksworm/argonaut/commit/0b58e89f7b6363adcae0b3defeff69e6f6fa8571))
* add comprehensive integration tests with reusable action ([6ce4f07](https://github.com/darksworm/argonaut/commit/6ce4f073734c62c10c87f31fcbd03ce845a40c8e))
* add comprehensive npm installation testing ([2dd6b38](https://github.com/darksworm/argonaut/commit/2dd6b388b99982539bc1e9b39cecc548b66a9067))
* add comprehensive UI tests for CommandBar and SearchBar components ([ef2b90d](https://github.com/darksworm/argonaut/commit/ef2b90d78cf11a4b7394499f27572ec7183aa61a))
* add comprehensive UI tests for ConfirmSyncModal with ANSI handling ([1376173](https://github.com/darksworm/argonaut/commit/1376173034ea98d91aaca4896e58f7e1cd191997))
* add comprehensive UI tests for successful authentication and cluster display ([87618dc](https://github.com/darksworm/argonaut/commit/87618dc26ba9b2517ded1defc5768df76f7e8442))
* add retry logic and version-specific npm testing ([3db633c](https://github.com/darksworm/argonaut/commit/3db633c5fbb25f515c98e5904fc5a6b555ae6d5d))
* complete App.tsx refactoring with bug fixes ([a585d83](https://github.com/darksworm/argonaut/commit/a585d83789efc5d8de3bebc3d9ed5d196f8e275b))
* create business logic orchestrator and specialized hooks ([e4fdd0b](https://github.com/darksworm/argonaut/commit/e4fdd0bb17928455bb775dd480d270e4ec553bbf))
* enhance install.sh with musl detection and POSIX compatibility ([8d432c3](https://github.com/darksworm/argonaut/commit/8d432c3b48853ef46e2e1b2bba7e16eec6c02d81))
* esc and :up to go up ([8666ab8](https://github.com/darksworm/argonaut/commit/8666ab8128211052929cb7c22b90333fa09caa8e))
* extract modal components from monolithic App.tsx ([b8da818](https://github.com/darksworm/argonaut/commit/b8da8183222ffd6ee6acd8894777d78d181660bb))
* extract view components and data processing logic ([e8a6ac3](https://github.com/darksworm/argonaut/commit/e8a6ac3098989f5cb7e6f9622dd8d5d35f7928c8))
* **help:** document :diff command ([b278878](https://github.com/darksworm/argonaut/commit/b27887819ab4c8fd6b6f435e95c3e71852f3e07c))
* implement comprehensive command pattern system ([14c3ef7](https://github.com/darksworm/argonaut/commit/14c3ef767055ce4e3b3dda54a8514bd2eb266537))
* implement comprehensive real UI testing with ink-testing-library ([9b5ae20](https://github.com/darksworm/argonaut/commit/9b5ae208797d0b806d9b28e1a9fa77f6d680ec23))


### Bug Fixes

* **build:** resolve TypeScript type errors ([9bf1e84](https://github.com/darksworm/argonaut/commit/9bf1e84722c50ffdda3ce832ad504bcf78459a37))
* clean dist directory before builds ([9dbbf1c](https://github.com/darksworm/argonaut/commit/9dbbf1c9199a91ce6d7aa5d240199696cbde8883))
* correct auth-required view and input handling ([657a115](https://github.com/darksworm/argonaut/commit/657a11513df57632901a32e61b34f60ec12d3f9b))
* correct modal positioning in Ink layout system ([b9c9a8a](https://github.com/darksworm/argonaut/commit/b9c9a8acd4580c3c280978172c3aab97e671109d))
* correct npm package and binary naming ([ffcafe7](https://github.com/darksworm/argonaut/commit/ffcafe732d82282ab44e41fcc4c230e231398938))
* ctrl-c and q exits app always ([693a35f](https://github.com/darksworm/argonaut/commit/693a35f6ea6ca865d7d82f5ddfa303dbcded91fd))
* disable space in all views except apps ([65cb291](https://github.com/darksworm/argonaut/commit/65cb2918d42975425215197775ccb15ab17fd52f))
* move bun install hooks to global before section ([943e5ee](https://github.com/darksworm/argonaut/commit/943e5eec4a02493f887978aa5866392f91f50081))
* preserve license files when cleaning dist directory ([c9356b8](https://github.com/darksworm/argonaut/commit/c9356b8b489ecb6379f113ec164acf1228fc7044))
* remove npm testing from test workflow ([6585c43](https://github.com/darksworm/argonaut/commit/6585c431cfc45f1a70446627d95eb264f5febf81))
* remove unused directive ([943cdba](https://github.com/darksworm/argonaut/commit/943cdbaa29f13f171b73d3eb82bc456167b3dd14))
* resolve terminal refresh issue after external viewers ([83e3689](https://github.com/darksworm/argonaut/commit/83e36896fb802da31bee1d3ab7bf2e5fa4cdd79f))
* resovled compilation error ([7f46a8e](https://github.com/darksworm/argonaut/commit/7f46a8e56fabebd677035f5f79f2425b483f94c9))
* separate Linux packages for standard and musl builds ([2b032d6](https://github.com/darksworm/argonaut/commit/2b032d65d98b452e1256a63d78a49f5a2699ec6d))
* separate musl and standard builds to fix GoReleaser archive error ([96b2d4f](https://github.com/darksworm/argonaut/commit/96b2d4fe567106a75055387e86a5f26236157f69))

## [1.12.0](https://github.com/darksworm/argonaut/compare/v1.11.0...v1.12.0) (2025-08-24)


### Features

* add :logs command to help screen ([b5b0102](https://github.com/darksworm/argonaut/commit/b5b0102f763c614368a02ba036f5f8fa847413b9))
* allow ? key to close help screen ([f205e92](https://github.com/darksworm/argonaut/commit/f205e925a6c9143a26b323a3e63104b9925f8d1d))
* improve diff cleaning and header stripping ([9cee1c6](https://github.com/darksworm/argonaut/commit/9cee1c6a2d1306a4af3a9f8da44ac8df16471245))
* improve help screen responsive layout ([fdfeef9](https://github.com/darksworm/argonaut/commit/fdfeef9d751a54c332e49dfb056d234a24fe8f86))
* strip garbage from diff ([b34bb2a](https://github.com/darksworm/argonaut/commit/b34bb2a57563f07417b8513e98b5d980489376e5))

## [1.11.0](https://github.com/darksworm/argonaut/compare/v1.10.2...v1.11.0) (2025-08-24)


### Features

* add automatic view tracking to all logs ([2fddbab](https://github.com/darksworm/argonaut/commit/2fddbabc94d7af7d18aaeb7fc3a8571339dcaec3))
* **auth-required:** allow switching to logs view ([3fe2f35](https://github.com/darksworm/argonaut/commit/3fe2f35127f17de765269edf5f7956079953bae3))
* implement comprehensive API error handling system ([ce79b9a](https://github.com/darksworm/argonaut/commit/ce79b9a6c7a4b2df02688b7a64eaebd5f2d155fd))
* proper error handling, logs and display ([c659b78](https://github.com/darksworm/argonaut/commit/c659b78f3fddc08512f960a3159da476a0d97f1e))
* replace less + pty with ink native pager ([60dd959](https://github.com/darksworm/argonaut/commit/60dd959ff73fed7e3129a02672e7995e142a6eb2))


### Bug Fixes

* **banner:** make sure the version number doesn't break the logo ([46a7559](https://github.com/darksworm/argonaut/commit/46a75590ea755fe7957161afae7008e20ac70fc8))
* cast mutableStdout and mutableStdin to appropriate stream types ([4eb0d50](https://github.com/darksworm/argonaut/commit/4eb0d507a990cdce6be34c246ce50aa4a9292e33))
* correct import path in tools/logs.ts ([a6f895c](https://github.com/darksworm/argonaut/commit/a6f895ca9cb22a3e08b9b69de9ad4b5f7858e2cd))
* **diff:** stop ink rendering when showing diff ([2331331](https://github.com/darksworm/argonaut/commit/23313314b808f6a281fbf49ad6972195bf37e6ff))
* **diff:** stop ink rendering when showing diff ([8dedafb](https://github.com/darksworm/argonaut/commit/8dedafb2703fbdaf851261b70f12030eef42e600))
* **diff:** stop ink rendering when showing diff ([7c12fbe](https://github.com/darksworm/argonaut/commit/7c12fbe5925bb9837a01bca08ca78830380964bb))
* **diff:** stop ink rendering when showing diff ([a01024e](https://github.com/darksworm/argonaut/commit/a01024efac4ba8732d1ae3493ba6e1eec8344aef))
* ensure continuous session following in log tailer ([e773bd8](https://github.com/darksworm/argonaut/commit/e773bd8befd8d5aa2f63d474ce7c4d92f53d8a44))
* remove unused bun-pty and node-pty dependencies ([c1f247d](https://github.com/darksworm/argonaut/commit/c1f247d7a2937698970c3cef2c76f9d3af06b56f))
* remove unused props from LogViewer component ([a89f793](https://github.com/darksworm/argonaut/commit/a89f793bd4f9d6b7e8bfef903f18999a3ef0379a))
* replace non-existent rerender() with status log messages ([8a4e09a](https://github.com/darksworm/argonaut/commit/8a4e09ade7abf80d6003de5f9e694f2afaf0a9c4))
* resolve circular dependency between main.tsx and ink-control.ts ([8ce000f](https://github.com/darksworm/argonaut/commit/8ce000ffdf76f2afe03727798a8cb3b7b4fb9100))
* resolve TypeScript build errors ([a0e7693](https://github.com/darksworm/argonaut/commit/a0e76933bb4e67e528a909f82de7dac0948f88f5))

## [1.10.2](https://github.com/darksworm/argonaut/compare/v1.10.1...v1.10.2) (2025-08-21)


### Bug Fixes

* **multi-platform:** update Nix configuration for argonaut package ([f04e6f6](https://github.com/darksworm/argonaut/commit/f04e6f680868b039b611974953eba5dbcb3c6611))

## [1.10.1](https://github.com/darksworm/argonaut/compare/v1.10.0...v1.10.1) (2025-08-21)


### Bug Fixes

* **multi-platform:** add Nix installation step in release pipeline ([79517ce](https://github.com/darksworm/argonaut/commit/79517ceb7d2e8353ee012c4d18a8d9f261c030d6))

## [1.10.0](https://github.com/darksworm/argonaut/compare/v1.9.0...v1.10.0) (2025-08-21)


### Features

* **multi-platform:** add nix configuration for argonaut package in goreleaser ([6807f59](https://github.com/darksworm/argonaut/commit/6807f5974dd1f78c243ac787fa63c69c5937773c))

## [1.9.0](https://github.com/darksworm/argonaut/compare/v1.8.2...v1.9.0) (2025-08-21)


### Features

* **multi-platform:** add nfpms configuration for Linux packages in goreleaser ([b66f3ff](https://github.com/darksworm/argonaut/commit/b66f3ff9d656e3b91e11ca2cc3534e315971d0ba))

## [1.8.2](https://github.com/darksworm/argonaut/compare/v1.8.1...v1.8.2) (2025-08-21)


### Bug Fixes

* **multi-platform:** update AUR configuration for argonaut-bin package ([72db1fb](https://github.com/darksworm/argonaut/commit/72db1fb3b55fbb0484a16b008078e84171a71bdc))

## [1.8.1](https://github.com/darksworm/argonaut/compare/v1.8.0...v1.8.1) (2025-08-21)


### Bug Fixes

* **multi-platform:** generate licenses file in pre-build step ([d1111b9](https://github.com/darksworm/argonaut/commit/d1111b952d59985edc3d6114f80f3a1650f4149c))
* **multi-platform:** generate licenses file in pre-build step ([bf572bc](https://github.com/darksworm/argonaut/commit/bf572bc6e7180fa6a8d0be14ad9bb020487af224))

## [1.8.0](https://github.com/darksworm/argonaut/compare/v1.7.0...v1.8.0) (2025-08-21)


### Features

* **multi-platform:** enable AUR support for argonaut-bin in GoReleaser configuration ([ea081e9](https://github.com/darksworm/argonaut/commit/ea081e9a1d1987ebe015aef389914af7477ba244))


### Bug Fixes

* **multi-platform:** generate licenses file in pre-build step ([805b6c3](https://github.com/darksworm/argonaut/commit/805b6c3a6f9840eee5c1588aaac03978eed04feb))

## [1.7.0](https://github.com/darksworm/argonaut/compare/v1.6.1...v1.7.0) (2025-08-21)


### Features

* **multi-platform:** update GoReleaser configuration for Bun builds and packaging ([ced3591](https://github.com/darksworm/argonaut/commit/ced3591d214a1863535962364de87717dc385491))

## [1.6.1](https://github.com/darksworm/argonaut/compare/v1.6.0...v1.6.1) (2025-08-21)


### Bug Fixes

* **multi-platform:** update GoReleaser configuration file name in release pipeline ([821342e](https://github.com/darksworm/argonaut/commit/821342e142a78edf0b99db374fccb1f8cb075bf8))

## [1.6.0](https://github.com/darksworm/argonaut/compare/v1.5.3...v1.6.0) (2025-08-21)


### Features

* **release:** add GoReleaser configuration for building and publishing binaries ([000a435](https://github.com/darksworm/argonaut/commit/000a4355c322ae279c3c85cffe6bbb7cfb686a07))


### Bug Fixes

* **build:** hopefully fix linux binary ([351a66b](https://github.com/darksworm/argonaut/commit/351a66b4e88fae042143b2b7abf9e9080dd971ae))

## [1.5.3](https://github.com/darksworm/argonaut/compare/v1.5.2...v1.5.3) (2025-08-21)


### Bug Fixes

* **build:** release to homebrew ([9ac165c](https://github.com/darksworm/argonaut/commit/9ac165c56296d361f9c239e9d906d0f598864479))

## [1.5.2](https://github.com/darksworm/argonaut/compare/v1.5.1...v1.5.2) (2025-08-20)


### Bug Fixes

* **ci:** update release asset upload to use new action and parameters ([18180f1](https://github.com/darksworm/argonaut/commit/18180f1f4fe559a4c78c2738d4142a2e63cbe148))

## [1.5.1](https://github.com/darksworm/argonaut/compare/v1.5.0...v1.5.1) (2025-08-20)


### Bug Fixes

* **ci:** npm auth for releases ([733ee32](https://github.com/darksworm/argonaut/commit/733ee320bd4b4286a036c8e7cb248821370b4896))

## [1.5.0](https://github.com/darksworm/argonaut/compare/v1.4.0...v1.5.0) (2025-08-20)


### Features

* **binary:** refactor(core): migrated project to bun ([35042f4](https://github.com/darksworm/argonaut/commit/35042f41b306ec4ca555183c5555922ed3bf7e77))


### Bug Fixes

* **binary:** diff works in binary build ([06b2997](https://github.com/darksworm/argonaut/commit/06b2997f06b0bd7af9f911c254502a6fbc29746b))
* **build:** downgrade upload-release-asset action to v1 ([79ffc41](https://github.com/darksworm/argonaut/commit/79ffc4174029f797270f006b1a9248de1bece83c))
* **config:** respect insecure attribute in config ([0fa3742](https://github.com/darksworm/argonaut/commit/0fa374280092b78f6b354f0dfea842246d57eb8e))
* **config:** respect plain-text attribute in config ([36ad4a4](https://github.com/darksworm/argonaut/commit/36ad4a4c34cb206fe2582f39b42327d98e5b6abc))
* **help:** update help close command from '?' to 'q' ([cdeffb7](https://github.com/darksworm/argonaut/commit/cdeffb7c9fcec238308145dfa70f95df37e2fa13))
* **http:** implement proper insecure flag handling with native Node.js HTTP ([1c36e68](https://github.com/darksworm/argonaut/commit/1c36e6882b483974ce737e934e8c2003b5cf68ac))
* **http:** improve signal handling and streaming implementation ([5a49618](https://github.com/darksworm/argonaut/commit/5a49618ba61b85243b949135fc0c21fe4b583821))
* **licenses:** working with bun-pty ([7d73787](https://github.com/darksworm/argonaut/commit/7d7378796f9d86037de73d85710b7aeb24873853))
* **streaming:** replace all fetch calls with HTTP client for insecure flag support ([9458d01](https://github.com/darksworm/argonaut/commit/9458d013613404ebe49818c936e55a9ccc01961f))

## [1.4.0](https://github.com/darksworm/argonaut/compare/v1.3.0...v1.4.0) (2025-08-15)


### Features

* **help:** add licenses command ([0515221](https://github.com/darksworm/argonaut/commit/05152213a5994499b3c13ff215a98ac386aba2ad))
* **licenses:** add licenses view and command ([a419286](https://github.com/darksworm/argonaut/commit/a41928670f6b45e72d56e631f7f9b791ae937106))


### Bug Fixes

* **rollback:** temporary fix for apps with multiple sources not showing revisions in the rollback view ([776d89c](https://github.com/darksworm/argonaut/commit/776d89c8f673206454dacef67ac9d3b6de50cc50))

## [1.3.0](https://github.com/darksworm/argonaut/compare/v1.2.0...v1.3.0) (2025-08-13)


### Features

* **version-checker:** add version checker with npm registry integration ([#11](https://github.com/darksworm/argonaut/issues/11)) ([3dd5b8a](https://github.com/darksworm/argonaut/commit/3dd5b8ab4cf55f27b2ddca6001ce73229940a5e7))

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
