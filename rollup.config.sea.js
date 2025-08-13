import typescript from 'rollup-plugin-typescript2';
import json from '@rollup/plugin-json';
import { nodeResolve } from '@rollup/plugin-node-resolve';
import commonjs from '@rollup/plugin-commonjs';
import { patchYogaTopLevelAwait } from './plugins/patch-yoga-toplevel-await.js';

export default {
  input: 'src/main.tsx',
  output: {
    file: 'dist/sea-bundle.cjs',
    format: 'cjs',
    sourcemap: false,
    banner: '#!/usr/bin/env node',
    exports: 'auto',
    inlineDynamicImports: true
  },
  external: [
    // Keep Node.js built-ins external
    'node:fs/promises', 'node:fs', 'node:path', 'node:os', 'node:process', 'node:child_process',
    'fs/promises', 'fs', 'path', 'os', 'process', 'child_process',
    // Keep native modules external (these need to be handled differently in SEA)
    'node-pty'
  ],
  plugins: [
    patchYogaTopLevelAwait(),
    nodeResolve({
      preferBuiltins: true,
      exportConditions: ['node', 'require'],
      browser: false
    }),
    commonjs({
      transformMixedEsModules: true,
      ignoreTryCatch: false
    }),
    json(),
    typescript({ 
      tsconfig: './tsconfig.sea.json',
      clean: true
    })
  ]
};