#!/usr/bin/env node

const message = `

Argonaut is no longer available via npm.

Please use another install method instead:

- Homebrew (Linux/macOS):
  brew tap darksworm/homebrew-tap
  brew install darksworm/tap/argonaut

- Install script (Linux/macOS):
  curl -sSL https://raw.githubusercontent.com/darksworm/argonaut/main/install.sh | sh

- AUR (Arch):
  yay -S argonaut-bin

- Binaries: https://github.com/darksworm/argonaut/releases/latest

Docs: https://github.com/darksworm/argonaut#install
`;

console.error(message);
process.exit(1);

