// jest.config.js
export default {
  testEnvironment: 'node',
  preset: 'ts-jest',
  setupFilesAfterEnv: ['<rootDir>/src/test-setup.js'],
  testMatch: [
    '<rootDir>/src/**/__tests__/**/*.{js,ts,tsx}',
    '<rootDir>/src/**/*.{test,spec}.{js,ts,tsx}',
    '!<rootDir>/src/**/__tests__/test-utils.{ts,tsx}' // Exclude utilities file
  ],
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1'
  },
  collectCoverageFrom: [
    'src/**/*.{ts,tsx}',
    '!src/**/*.d.ts',
    '!src/test-setup.js',
    '!src/**/__tests__/test-utils.{ts,tsx}' // Exclude utilities file from coverage
  ],
  transform: {
    '^.+\\.(ts|tsx)$': ['ts-jest', {
      tsconfig: {
        resolveJsonModule: true
      }
    }]
  },
  transformIgnorePatterns: [
    'node_modules/(?!(execa|.*\\.mjs$))'
  ],
  // Move ts-jest config to transform instead of deprecated globals
  extensionsToTreatAsEsm: ['.ts', '.tsx']
};