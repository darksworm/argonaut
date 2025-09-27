#!/usr/bin/env node
const { spawn } = require('child_process');
const { dirname, join } = require('path');
const pkg = require('../package.json');

function tryResolve(name) {
  try { return dirname(require.resolve(name + '/package.json')); }
  catch (_) { return null; }
}

// Discover platform packages dynamically from optionalDependencies.
const candidates = Object.keys(pkg.optionalDependencies || {});
const dir = candidates.map(tryResolve).find(Boolean);
if (!dir) {
  console.error('[argonaut] No platform binary package installed.');
  console.error('Install instructions: https://github.com/darksworm/argonaut#install');
  process.exit(1);
}

const exe = process.platform === 'win32' ? 'argonaut.exe' : 'argonaut';
const bin = join(dir, 'bin', exe);
const child = spawn(bin, process.argv.slice(2), { stdio: 'inherit' });
child.on('exit', code => process.exit(code));
