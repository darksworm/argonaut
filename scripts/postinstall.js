#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const os = require('os');

// Determine the correct binary for this platform
function getBinaryName() {
  const platform = os.platform();
  const arch = os.arch();

  const binaries = {
    'darwin-arm64': 'a9s-darwin-arm64',
    'darwin-x64': 'a9s-darwin-x64',
    'linux-arm64': 'a9s-linux-arm64',
    'linux-x64': 'a9s-linux-x64',
    'win32-x64': 'a9s-windows-x64.exe',
  };

  const key = `${platform}-${arch === 'x64' ? 'x64' : arch}`;
  return binaries[key];
}

// Link the correct binary to bin/a9s and bin/argonaut
function setupBinaries() {
  const binaryName = getBinaryName();
  if (!binaryName) {
    console.error(`Unsupported platform: ${os.platform()}-${os.arch()}`);
    console.error('You may need to build from source: https://github.com/darksworm/argonaut');
    process.exit(1);
  }

  const binDir = path.join(__dirname, '..', 'bin');
  const platformBinDir = path.join(__dirname, '..', 'bin-platform');
  const sourceBinary = path.join(platformBinDir, binaryName);
  const targetBinary = path.join(binDir, 'a9s');
  const argonautLink = path.join(binDir, 'argonaut');

  // Check if platform binary exists
  if (!fs.existsSync(sourceBinary)) {
    console.error(`Binary not found: ${sourceBinary}`);
    console.error('This platform may not be supported. Please build from source.');
    process.exit(1);
  }

  // Create bin directory if it doesn't exist
  if (!fs.existsSync(binDir)) {
    fs.mkdirSync(binDir, { recursive: true });
  }

  // Remove existing links/files
  try {
    if (fs.existsSync(targetBinary)) fs.unlinkSync(targetBinary);
    if (fs.existsSync(argonautLink)) fs.unlinkSync(argonautLink);
  } catch (err) {
    // Ignore errors
  }

  // Copy the correct binary
  fs.copyFileSync(sourceBinary, targetBinary);
  fs.chmodSync(targetBinary, 0o755);

  // Create argonaut symlink
  fs.symlinkSync('a9s', argonautLink);

  console.log(`✓ Argonaut CLI installed for ${os.platform()}-${os.arch()}`);
}

try {
  setupBinaries();
} catch (err) {
  console.error('Failed to setup binaries:', err);
  process.exit(1);
}