#!/usr/bin/env node

const pkg = require('../package.json');

function tryResolve(name) {
  try {
    require.resolve(name + '/package.json');
    return true;
  }
  catch (_) {
    return false;
  }
}

// Check if any platform package is available
const platformPkgs = Object.keys(pkg.optionalDependencies || {});
const installed = platformPkgs.find(tryResolve);

if (!installed) {
  console.error('');
  console.error('╭─────────────────────────────────────────────────────────────╮');
  console.error('│  ⚠️  No platform binary installed                           │');
  console.error('│                                                             │');
  console.error('│  This may indicate an installation failure or that your     │');
  console.error('│  platform is not supported via npm.                         │');
  console.error('│                                                             │');
  console.error('│  Please install using one of these methods:                 │');
  console.error('│                                                             │');
  console.error('│  📦 Homebrew (macOS):                                       │');
  console.error('│      brew install darksworm/tap/argonaut                    │');
  console.error('│                                                             │');
  console.error('│  🐧 Linux/WSL:                                              │');
  console.error('│      curl -s https://api.github.com/repos/darksworm/        │');
  console.error('│      argonaut/releases/latest | grep browser_download_url   │');
  console.error('│      | cut -d\\" -f4 | wget -qi -                           │');
  console.error('│                                                             │');
  console.error('│  📋 More options:                                           │');
  console.error('│      https://github.com/darksworm/argonaut#install          │');
  console.error('╰─────────────────────────────────────────────────────────────╯');
  console.error('');
  process.exit(1);
} else {
  // Silent success - platform package installed correctly
  console.log('✅ Argonaut CLI installed successfully');
}