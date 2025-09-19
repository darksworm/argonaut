#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const binDir = path.join(__dirname, '..', 'bin');

try {
  if (fs.existsSync(binDir)) {
    fs.rmSync(binDir, { recursive: true, force: true });
    console.log('✓ Argonaut CLI uninstalled successfully');
  }
} catch (err) {
  console.error('Error during uninstall:', err);
  process.exit(1);
}