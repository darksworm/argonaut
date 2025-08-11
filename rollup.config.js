import typescript from 'rollup-plugin-typescript2';
import json from '@rollup/plugin-json';

export default {
  input: 'src/main.tsx', // adjust if your entry point is different
  output: [
    {
      file: 'dist/cli.js',
      format: 'cjs',
      sourcemap: true,
    },
  ],
  plugins: [
    json(),
    typescript({
      tsconfig: './tsconfig.json'
    })
  ]
};
