// src/test-setup.js
require('@testing-library/jest-dom');

// Mock external dependencies that don't work in test environment
jest.mock('./services/log-viewer');
jest.mock('./components/DiffView');
jest.mock('./components/LicenseView');