import typescript from 'rollup-plugin-typescript2';
import json from '@rollup/plugin-json';
import { nodeResolve } from '@rollup/plugin-node-resolve';
import { codecovRollupPlugin } from "@codecov/rollup-plugin";

export default {
    input: 'src/main.tsx',             // make this your CLI entry
    output: {
        file: 'dist/cli.js',            // matches package.json "bin"
        format: 'esm',
        sourcemap: true,
        inlineDynamicImports: true,
        banner: '#!/usr/bin/env node\nprocess.env.NODE_ENV = process.env.NODE_ENV || "production";'   // Set NODE_ENV for production builds
    },
    external: [
        'node:fs','node:fs/promises','node:path','node:os','node:process','node:child_process',
        'node:http','node:https','node:url','node:stream','node:zlib',
        'node-pty','chalk','execa','react','react/jsx-runtime','ink','ink-text-input','yaml','string-width',
        'neverthrow','pino',
        'fs','path','os'
    ],
    plugins: [
        nodeResolve({
            preferBuiltins: true,
            extensions: ['.js', '.jsx', '.ts', '.tsx', '.json']
        }),
        json(),
        typescript({ tsconfig: './tsconfig.json' }),
	codecovRollupPlugin({
	    enableBundleAnalysis: process.env.CODECOV_TOKEN !== undefined,
	    bundleName: "argonaut",
	    uploadToken: process.env.CODECOV_TOKEN,
	}),
    ]
};
