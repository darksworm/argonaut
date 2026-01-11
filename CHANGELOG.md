# Changelog

## [2.11.0](https://github.com/darksworm/argonaut/compare/v2.10.0...v2.11.0) (2026-01-09)


### Features

* add mouse text selection and clipboard copy ([#175](https://github.com/darksworm/argonaut/issues/175)) ([ac216f8](https://github.com/darksworm/argonaut/commit/ac216f8447fd269c85b5968e594a44c5ac6b3d00))


### Bug Fixes

* modals now how consistent backgrounds ([#178](https://github.com/darksworm/argonaut/issues/178)) ([7dfc7b8](https://github.com/darksworm/argonaut/commit/7dfc7b8a08a58644532f61eed06bdb3a62f33da7))
* the resource tree view is now cleared after watching a sync ([#182](https://github.com/darksworm/argonaut/issues/182)) ([b3abaf3](https://github.com/darksworm/argonaut/commit/b3abaf33e15b7d4bf9645de226fc86ea91c234ad))

## [2.10.0](https://github.com/darksworm/argonaut/compare/v2.9.0...v2.10.0) (2025-12-10)


### Features

* add :refresh and :refresh! commands for app refresh, available in the apps list and resource tree views ([66b9610](https://github.com/darksworm/argonaut/commit/66b9610908427204422e4f6cba4b5c52566ac2e3))

## [2.9.0](https://github.com/darksworm/argonaut/compare/v2.8.0...v2.9.0) (2025-12-07)


### Features

* you can now jump to k9s by pressing K when hovering a resource ([c3520a5](https://github.com/darksworm/argonaut/commit/c3520a52a4f225fc72caf8457894a54f418bf580))
* added a new ApplicationSet view for filtering apps by ApplicationSet ([c2c2ffd](https://github.com/darksworm/argonaut/commit/c2c2ffd004d6b6b2a8cea2509bf2649551a9d930))
* enabled command mode and improved tree view UX ([cc28be7](https://github.com/darksworm/argonaut/commit/cc28be73edfdb3f083a6d2d173b007f48661622a))
* added individual resource diff in the tree view ([#148](https://github.com/darksworm/argonaut/issues/148)) ([4e3964a](https://github.com/darksworm/argonaut/commit/4e3964a5a556ffca7f713c24864aa37f0850c98b))
* added individual resource deletion in the tree view with Ctrl+D ([6a67400](https://github.com/darksworm/argonaut/commit/6a67400a0d7e08e2608e77fb30ecdc50cd1f1e97))
* added individual resource sync in the tree view ([#162](https://github.com/darksworm/argonaut/issues/162)) ([c4a51a9](https://github.com/darksworm/argonaut/commit/c4a51a93f1082c0f8b6331729e479afd9a2d01bb))
* added sync status for individual resources in tree view ([62778f6](https://github.com/darksworm/argonaut/commit/62778f63736437a054d1dda85a1f9551efcea740))
* added support for ArgoCD CLI --port-forward access mode ([#159](https://github.com/darksworm/argonaut/issues/159)) ([be03cd3](https://github.com/darksworm/argonaut/commit/be03cd3f9f59c032b17617a2eccc87f8aba6bc3e))
* moved env vars to config ([671db05](https://github.com/darksworm/argonaut/commit/671db0509d18996818b076ee81eb8d199024e7a4))([2628ac8](https://github.com/darksworm/argonaut/commit/2628ac83b11fcfe4683246355705020cff79540e)) ([bf5de9b](https://github.com/darksworm/argonaut/commit/bf5de9b154162141d85a3c77af51567c9149dd8a)) ([f87e858](https://github.com/darksworm/argonaut/commit/f87e85817ac372637254b5d472460d46a040aa10))
* pressing Enter on an app now opens its resources tree view ([7394f85](https://github.com/darksworm/argonaut/commit/7394f8587e6396eb54a62c3bb552c5dd44e74c01))


### Bug Fixes

* removed expand arrow from tree view to clarify visual hierarchy ([7f290de](https://github.com/darksworm/argonaut/commit/7f290de9c74a0fbc038ee2c1778b4412558e8516))


## [2.8.0](https://github.com/darksworm/argonaut/compare/v2.7.0...v2.8.0) (2025-11-29)


### Features

* add :sort command for sorting apps list ([#144](https://github.com/darksworm/argonaut/issues/144)) ([c734d9b](https://github.com/darksworm/argonaut/commit/c734d9b0e3ee4a2cac1b4e58105dba2e8ef8d2cd))
* add what's new notification and :changelog command ([84091ff](https://github.com/darksworm/argonaut/commit/84091ffa467a581afbef02e5f8795c83e90016fa))
* **help:** reorganize help sections by view context ([beb94ab](https://github.com/darksworm/argonaut/commit/beb94ab00db13686cce7e8029d7a6ac2509cc378))
* page up and page down can be used to navigate lists now ([54251aa](https://github.com/darksworm/argonaut/commit/54251aa2047bb7c358e99c0469131e1f475832c6))
* **resources view:** add filter functionality ([#145](https://github.com/darksworm/argonaut/issues/145)) ([a0afd1f](https://github.com/darksworm/argonaut/commit/a0afd1f0b1c49564c2e4cb3b42bc9cd1a1838cde))


### Bug Fixes

* bordered full-screen views now fill terminal height correctly ([9e3a88e](https://github.com/darksworm/argonaut/commit/9e3a88e642e885d86772509f4229dffe709702bb))

## [2.7.0](https://github.com/darksworm/argonaut/compare/v2.6.1...v2.7.0) (2025-11-14)


### Features

* add ArgoCD core mode detection with suggested workaround ([#128](https://github.com/darksworm/argonaut/issues/128)) ([2702907](https://github.com/darksworm/argonaut/commit/27029070989f29f6bb72ee0a3f519faed04fe0a3))
* add grpc-web-root-path support for ArgoCD API requests ([f2506c3](https://github.com/darksworm/argonaut/commit/f2506c348fd3cbf9298b53181222a567446a2e17))
* added nix flake ([f6f68bc](https://github.com/darksworm/argonaut/commit/f6f68bceade28ae55b19b409a77a549c67863f00))
* support pasting into command and search bar ([#134](https://github.com/darksworm/argonaut/issues/134)) ([de99161](https://github.com/darksworm/argonaut/commit/de991613e4c0317aa5c8c52c2cdbdc5a2039d59b))

## [2.6.1](https://github.com/darksworm/argonaut/compare/v2.6.0...v2.6.1) (2025-11-11)


### Bug Fixes

* fixed some edge cases where the argocd config file path was not detected correctly ([#126](https://github.com/darksworm/argonaut/issues/126)) ([04905d7](https://github.com/darksworm/argonaut/commit/04905d76192dbbc1da206fee36866b773b3779e5))

## [2.6.0](https://github.com/darksworm/argonaut/compare/v2.5.0...v2.6.0) (2025-11-07)


### Features

* widen support for themes ([87aabd5](https://github.com/darksworm/argonaut/commit/87aabd567ba4f15d9721e05e1e32d7c5ad40ec95))

## [2.5.0](https://github.com/darksworm/argonaut/compare/v2.4.0...v2.5.0) (2025-11-07)


### Features

* command validation ([76eff2c](https://github.com/darksworm/argonaut/commit/76eff2cdb16ca4e5c5d49a74d5defad78190a229))
* theme config system ([#112](https://github.com/darksworm/argonaut/issues/112)) ([3b64701](https://github.com/darksworm/argonaut/commit/3b647012dc059edd37914c55b0f0739c374fbda9))
* use theme color palette in resources view ([a680be4](https://github.com/darksworm/argonaut/commit/a680be4531d29f44f6f238d8afdfd1ccc54607e6))


### Bug Fixes

* remove Unicode symbol check from TestSimpleInvalidCommand ([078165e](https://github.com/darksworm/argonaut/commit/078165e51c68a22cfaa947291b25121d28b7b256))

## [2.4.0](https://github.com/darksworm/argonaut/compare/v2.3.0...v2.4.0) (2025-11-04)


### Features

* add application deletion functionality ([565ba04](https://github.com/darksworm/argonaut/commit/565ba047e833039e3f6b979a9e95bf4b9d6152f6))
* enforce lexicographical ordering for lists ([#107](https://github.com/darksworm/argonaut/issues/107)) ([0f79fc7](https://github.com/darksworm/argonaut/commit/0f79fc778e343ef084f95a314545a01049985e7c))


### Bug Fixes

* make the escape debouncer global ([a3f4dbb](https://github.com/darksworm/argonaut/commit/a3f4dbb9be07751eb0ac12c1aaa51a537113c90f))
* sync command now targets correctly selected app ([5170fb7](https://github.com/darksworm/argonaut/commit/5170fb7c34e33c37c25798e8cc26224de966413e))

## [2.3.0](https://github.com/darksworm/argonaut/compare/v2.2.0...v2.3.0) (2025-09-30)


### Features

* add no differences modal with improved UX ([d52c867](https://github.com/darksworm/argonaut/commit/d52c867838031c8ac9ed97de629b5e293ce3a0ef))
* add vim-style quit key bindings ([777e2a2](https://github.com/darksworm/argonaut/commit/777e2a2eafd121e565bab37b2460c6af50ac2fcc))
* calculate diff just like the argocd web app ([7c1c8fd](https://github.com/darksworm/argonaut/commit/7c1c8fda509e77b48754b0f5ede332b2252300dc))
* disable q key for quitting app in normal mode ([918d58d](https://github.com/darksworm/argonaut/commit/918d58d0ba3c085cf07fabe5ceba250d1af9490f))
* mark releases as pre-release during build process ([82e75e1](https://github.com/darksworm/argonaut/commit/82e75e1fa904cd4c121bb8e98588a5097cc79597))
* show 'no difference' when trying to diff a synced app ([7b1fbe3](https://github.com/darksworm/argonaut/commit/7b1fbe3a3be082ba74dcfeb81a4cd49047f75a5f))
* simplify no-diff modal interaction ([42f3bfd](https://github.com/darksworm/argonaut/commit/42f3bfd7923199a0b68afa9922277648722eb27d))
* support more common vim commands to close the app ([1ed4966](https://github.com/darksworm/argonaut/commit/1ed4966fa5ce173dbba8650cd6f2edb4f4bfa8ac))

## [2.2.0](https://github.com/darksworm/argonaut/compare/v2.1.2...v2.2.0) (2025-09-29)


### Features

* add --version and --help command line flags ([194a9bd](https://github.com/darksworm/argonaut/commit/194a9bd6122d6f515ee08ae9c3e8058cbaf82cb5))
* add colorful styling to --help output using lipgloss ([0754fe5](https://github.com/darksworm/argonaut/commit/0754fe50a25cab0b9652df5e9bb729dd76773e3d))

## [2.1.2](https://github.com/darksworm/argonaut/compare/v2.1.1...v2.1.2) (2025-09-29)


### Bug Fixes

* truncate commit messages to single line in rollback view ([b4ca3b7](https://github.com/darksworm/argonaut/commit/b4ca3b7d55a3eb2d759d1b0d7ad6d120d24bb5ce))

## [2.1.1](https://github.com/darksworm/argonaut/compare/v2.1.0...v2.1.1) (2025-09-29)


### Bug Fixes

* increase scanner buffer size to handle large ArgoCD payloads ([f74e497](https://github.com/darksworm/argonaut/commit/f74e49707c9773da1baac1ce47b97c6549840de7))

## [2.1.0](https://github.com/darksworm/argonaut/compare/v2.0.3...v2.1.0) (2025-09-29)


### Features

* add in-app upgrade system with GitHub releases integration ([41cd5f0](https://github.com/darksworm/argonaut/commit/41cd5f048784d1b65960e35ca03ce87dd8ca63dc))


### Bug Fixes

* update install script and integration tests ([0ed406f](https://github.com/darksworm/argonaut/commit/0ed406f905a85e8bbede23b5af0418883e6d0da5))

## [2.0.3](https://github.com/darksworm/argonaut/compare/v2.0.2...v2.0.3) (2025-09-28)


### Bug Fixes

* **ci:** disabled npm publishing for pre-releases ([b7e07f3](https://github.com/darksworm/argonaut/commit/b7e07f37f19e92ac0291b9cc75e3f78c42b53d64))

## [2.0.2](https://github.com/darksworm/argonaut/compare/v2.0.1...v2.0.2) (2025-09-28)


### Bug Fixes

* **ci:** fixed npm publishing ([1818571](https://github.com/darksworm/argonaut/commit/18185713fb84a3e4df5429af14ffe47d6e735c1a))

## [2.0.1](https://github.com/darksworm/argonaut/compare/v2.0.0...v2.0.1) (2025-09-28)


### Bug Fixes

* align rollback modal spacing and height without hiding status line ([dcae1ec](https://github.com/darksworm/argonaut/commit/dcae1ec58c8e2d1a29bd7af2f5a58d4316f3f278))

## [2.0.0](https://github.com/darksworm/argonaut/compare/v1.16.1...v2.0.0) (2025-09-28)


### ⚠ BREAKING CHANGES

* go rewrite ([#84](https://github.com/darksworm/argonaut/issues/84))

### Features

* add client certificate authentication support ([fb1413f](https://github.com/darksworm/argonaut/commit/fb1413f71a52005bcdc7729aecf487af63260599))
* add detailed logging for client certificate configuration ([79f4760](https://github.com/darksworm/argonaut/commit/79f47605f2e4cdfbe7d596c3c0cb474d29cb2fe7))
* add GitHub Actions workflow for PR pre-release builds ([692003f](https://github.com/darksworm/argonaut/commit/692003f84c77f5245d8ff1d2080a38d74c011f5b))
* add TLS certificate trust feature with comprehensive e2e tests ([35d3277](https://github.com/darksworm/argonaut/commit/35d32773bb4a43a7508ea2145d6666c130cc087d))
* **apps view:** when cursor is on a selected app, use a distinct highlight (cyan) so the active row stands out ([3dda852](https://github.com/darksworm/argonaut/commit/3dda852b606e2e9e5d0bc87f64684e99fa02be6a))
* go rewrite ([#84](https://github.com/darksworm/argonaut/issues/84)) ([c574906](https://github.com/darksworm/argonaut/commit/c5749060cdbfd2cbbf157f470a086b60a54577eb))
* improve pre-release workflow with individual file downloads ([efcd260](https://github.com/darksworm/argonaut/commit/efcd26070c5bf887f39f17a8eeb640dfb673bd2e))
* publishing to npm ([1949f94](https://github.com/darksworm/argonaut/commit/1949f9419a385b1e5bc57bbe17181a78c51a358b))
* **treeview:** limit selection highlight to prefix+kind in resource tree; update golden snapshot ([4d886f8](https://github.com/darksworm/argonaut/commit/4d886f8e4b690c694d4acdce0e4840db8b2ffc87))


### Bug Fixes

* add .zip extension to download links and improve file detection ([aca5e5c](https://github.com/darksworm/argonaut/commit/aca5e5c8fc531f03eb6e341839afc998461a6d92))
* **apps header:** align compact S/H headers with row icons at narrow widths; update tests and golden ([d12b30c](https://github.com/darksworm/argonaut/commit/d12b30c48cd23508458d057937eb4480c29ede68))
* **apps table:** use responsive column widths for header (calculateColumnWidths) so SYNC values align with header at narrow widths ([8782411](https://github.com/darksworm/argonaut/commit/87824119eff091b0920465a8285d73302ecbc98e))
* **ci:** download artifacts ([abe764b](https://github.com/darksworm/argonaut/commit/abe764ba16bb5e5aa4ee83d724ab4398c4e204a5))
* create separate artifacts by file type for easier downloads ([44a3557](https://github.com/darksworm/argonaut/commit/44a35570d780d1041f4f5db7765adfcc9fb4908b))
* critical TLS implementation issues affecting test stability ([7f99eb7](https://github.com/darksworm/argonaut/commit/7f99eb746929ab8a1c522cdcfc195f6088e3f648))
* **e2e:** ensure streaming test sees initial state before update ([b36229f](https://github.com/darksworm/argonaut/commit/b36229fc507759b636f2213e3a61c129fb11d173))
* help test flakiness in CI by waiting for full app initialization ([4085017](https://github.com/darksworm/argonaut/commit/40850176415e17167c77eb3d2a00c68e70d6b658))
* **pr-prerelease:** use correct version output reference in package-artifacts job ([fdfb968](https://github.com/darksworm/argonaut/commit/fdfb9687756ab339fd12051ab4e2e4701dfa3582))
* properly detect and copy macOS binaries from GoReleaser subdirectories ([c350e6d](https://github.com/darksworm/argonaut/commit/c350e6d215a9379e3e29b5cd81e8e91736ba4bc0))
* remove paths-ignore that was blocking workflow trigger on markdown changes ([9c639f2](https://github.com/darksworm/argonaut/commit/9c639f2f9c471a80564a56ef8a60578918cf8655))
* resolve variable redeclaration error in client certificate logging ([c16b5de](https://github.com/darksworm/argonaut/commit/c16b5deda57e9f8006ecf1b33e56c41de59600c4))
* SSL_CERT_DIR colon-separated directory support [P1] ([732f4c2](https://github.com/darksworm/argonaut/commit/732f4c2dbc2dc93eeb45073d44428780eee0d793))
* **tree scroll:** keyboard handler uses line-based indices (SelectedLineIndex/VisibleLineCount) and exact viewport height to stay in sync with rendered separators ([16e1f4a](https://github.com/darksworm/argonaut/commit/16e1f4af891a056ffed4e51aa1f689e05cf6a5bc))
* **tree:** remove panel row shading and correct scrolling across app separators; feat(banner): compact header + width-based version; update goldens ([5ef37d6](https://github.com/darksworm/argonaut/commit/5ef37d67141fef657bac1bf3c3c9ddbcc7c34ff8))
* **ui:** align status column headers with content at all terminal widths ([9195479](https://github.com/darksworm/argonaut/commit/91954794da538b93893bec8f3c45d307e9b3f179))
* unify cli param names ([87bb7f8](https://github.com/darksworm/argonaut/commit/87bb7f82e4fdc5f7faa37316a02d679b2c55c069))
* **workflows:** add missing default values for workflow_call inputs ([6a67282](https://github.com/darksworm/argonaut/commit/6a67282e62f53d0311d4eff2121d7d00238abaaa))
* **workflows:** move name field to top of release-pipeline.yml ([8b6ac3a](https://github.com/darksworm/argonaut/commit/8b6ac3a7e38f429ecd75e14f9ee9d10b868f6172))

## [1.16.0](https://github.com/darksworm/argonaut/compare/v1.15.1...v1.16.0) (2025-09-06)


### Bug Fixes

* ensure react-reconciler uses production build in node and binary outputs ([6e95d24](https://github.com/darksworm/argonaut/commit/6e95d24be43e238c6d1ef9d3db4d710d039b7e6f))
* login to ghcr before goreleaser to fix docker build ([777b39a](https://github.com/darksworm/argonaut/commit/777b39a7f7da83179b25d23e9a645e921e560239))


## [1.15.1](https://github.com/darksworm/argonaut/compare/v1.15.0...v1.15.1) (2025-08-31)


### Bug Fixes

* disable broken docker build ([a7a980c](https://github.com/darksworm/argonaut/commit/a7a980c4045c2034227e44b4521b426b052883a9))

## [1.15.0](https://github.com/darksworm/argonaut/compare/v1.14.0...v1.15.0) (2025-08-31)


### Features

* add 'd' key shortcut for diff command in apps view ([28e72e9](https://github.com/darksworm/argonaut/commit/28e72e962cba1f15433dbf13f9b0038470f085d2))
* add command parameter autocomplete ([#66](https://github.com/darksworm/argonaut/issues/66)) ([6aa61ea](https://github.com/darksworm/argonaut/commit/6aa61ea919e22e2808760020d10497a550c1db1a))
* add command name autocomplete ([#69](https://github.com/darksworm/argonaut/issues/69)) ([0179d07](https://github.com/darksworm/argonaut/commit/0179d0799ddbb725043aa41f5ecb3fb0ddfc5cd3))
* enhance navigation commands with automatic drill-down ([a475f38](https://github.com/darksworm/argonaut/commit/a475f3869ffb8b03918c63a41b3867047ffaf63a))
* show command descriptions in command bar ([#71](https://github.com/darksworm/argonaut/issues/71)) ([f765296](https://github.com/darksworm/argonaut/commit/f765296f9bd6f04f325c93965bdbe30b03f3d018))


## [Unreleased]

### Features

* **docker:** add Dockerfiles, Docker scripts, and documentation for container usage

## [1.14.0](https://github.com/darksworm/argonaut/compare/v1.13.0...v1.14.0) (2025-08-29)


### Features

* add version flag ([#51](https://github.com/darksworm/argonaut/issues/51)) ([4b76a80](https://github.com/darksworm/argonaut/commit/4b76a8046a15788facc1f3e818e965c489c09976))


### Bug Fixes

* allow navigation commands to execute in command mode ([df4fd4e](https://github.com/darksworm/argonaut/commit/df4fd4ebd2203c14dc43d3581d06f1f967a0a028))
* navigation boundary issues and diff command app selection ([291ca03](https://github.com/darksworm/argonaut/commit/291ca03a70ca7bae3a360cbb8996ef10e45b56d0))
* prevent overlapping statuses in resource view ([#52](https://github.com/darksworm/argonaut/issues/52)) ([9745939](https://github.com/darksworm/argonaut/commit/97459397d253be12bf6a4e078d07b2f1695debd0))
* sync, rollback, and resources commands app selection in non-apps views ([768f4ae](https://github.com/darksworm/argonaut/commit/768f4ae1d062c00b45831f4c84256f2ff95c09ac))

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
