import typescript from 'rollup-plugin-typescript2';
import json from '@rollup/plugin-json';
import { codecovRollupPlugin } from "@codecov/rollup-plugin";

export default {
    input: 'src/main.tsx',             // make this your CLI entry
    output: {
        file: 'dist/cli.js',            // matches package.json "bin"
        format: 'esm',
        sourcemap: true,
        banner: '#!/usr/bin/env node'   // required for CLI
    },
    external: [
        'node:fs','node:fs/promises','node:path','node:os','node:process','node:child_process',
        'node-pty','chalk','execa','react','react/jsx-runtime','ink','ink-text-input','yaml','string-width',
        'fs','path','os'
    ],
    plugins: [
        json(),
        typescript({ tsconfig: './tsconfig.json' }),
	codecovRollupPlugin({
	    enableBundleAnalysis: process.env.CODECOV_TOKEN !== undefined,
	    bundleName: "argonaut",
	    uploadToken: process.env.CODECOV_TOKEN,
	}),
    ]
};
