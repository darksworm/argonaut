// src/__tests__/test-utils.ts
import { mock } from "bun:test";
/**
 * Test utilities and mock factories for unit testing
 * This file contains only utilities and does not have its own tests
 */

// Import the actual AppState type instead of creating our own
import type { AppState } from "../contexts/AppStateContext";
import type { AppItem, Mode, View } from "../types/domain";

export interface MockCommandContext {
  state: AppState;
  dispatch: mock.Mock;
  statusLog: ReturnType<typeof createMockStatusLog>;
  cleanupAndExit: mock.Mock;
  navigationActions?: {
    drillDown: mock.Mock;
    toggleSelection: mock.Mock;
  };
  executeCommand: mock.Mock;
}

// Mock creation helpers
export function createMockContext(
  overrides: Partial<MockCommandContext> = {},
): MockCommandContext {
  return {
    state: createMockState(),
    dispatch: mock(),
    statusLog: createMockStatusLog(),
    cleanupAndExit: mock(),
    navigationActions: {
      drillDown: mock(),
      toggleSelection: mock(),
    },
    executeCommand: mock(),
    ...overrides,
  };
}

export function createMockState(overrides: Partial<AppState> = {}): AppState {
  const base: AppState = {
    mode: "normal" as Mode,
    terminal: { rows: 24, cols: 80 },
    navigation: {
      view: "apps" as View,
      selectedIdx: 0,
      lastGPressed: 0,
      lastEscPressed: 0,
    },
    selections: {
      scopeClusters: new Set(),
      scopeNamespaces: new Set(),
      scopeProjects: new Set(),
      selectedApps: new Set(),
    },
    ui: {
      searchQuery: "",
      activeFilter: "",
      command: "",
      isVersionOutdated: false,
      latestVersion: undefined,
      commandInputKey: 0,
    },
    modals: {
      confirmTarget: null,
      confirmSyncPrune: false,
      confirmSyncWatch: true,
      rollbackAppName: null,
      syncViewApp: null,
    },
    server: {
      config: { baseUrl: "https://test-server.com" },
      token: "test-token",
    },
    apps: [],
    apiVersion: "v2.9.0",
    loadingAbortController: null,
  };

  return {
    ...base,
    ...overrides,
    navigation: { ...base.navigation, ...(overrides.navigation ?? {}) },
    selections: { ...base.selections, ...(overrides.selections ?? {}) },
    ui: { ...base.ui, ...(overrides.ui ?? {}) },
    modals: { ...base.modals, ...(overrides.modals ?? {}) },
  };
}

export function createMockStatusLog() {
  return {
    info: mock(),
    warn: mock(),
    error: mock(),
    debug: mock(),
    set: mock(),
    clear: mock(),
  };
}

export function createMockApps(): AppItem[] {
  return [
    {
      name: "app1",
      sync: "Synced",
      health: "Healthy",
      clusterId: "in-cluster",
      clusterLabel: "in-cluster",
      namespace: "default",
      appNamespace: "argocd",
      project: "default",
      lastSyncAt: "2023-12-01T10:00:00Z",
    },
    {
      name: "app2",
      sync: "OutOfSync",
      health: "Progressing",
      clusterId: "staging",
      clusterLabel: "staging",
      namespace: "app-namespace",
      appNamespace: "argocd",
      project: "team-a",
      lastSyncAt: "2023-12-01T09:30:00Z",
    },
  ];
}

export function createMockCommand(overrides: Partial<any> = {}) {
  return {
    execute: mock(),
    canExecute: mock().mockReturnValue(true),
    description: "Test command",
    aliases: [],
    ...overrides,
  };
}

// Test data factories
export const mockCliConfig = {
  currentContext: "test-context",
  contexts: [
    {
      name: "test-context",
      server: "https://test-server.com",
      user: "test-user",
    },
  ],
  users: [
    {
      name: "test-user",
      "auth-token": "test-token",
    },
  ],
};

export const mockServerConfig = {
  server: "https://test-server.com",
  username: "test-user",
};

// Mock API responses
export const mockApiResponses = {
  listApps: {
    isOk: () => true,
    value: createMockApps(),
  },
  syncApp: {
    isOk: () => true,
    value: { operationState: { phase: "Running" } },
  },
  listClusters: {
    isOk: () => true,
    value: ["in-cluster", "staging", "production"],
  },
};

// UI Test utilities
/**
 * Strips ANSI escape codes from terminal output for easier testing
 * @param text The text containing ANSI codes
 * @returns Clean text without ANSI codes
 */
export function stripAnsi(text: string): string {
  // biome-ignore lint/suspicious/noControlCharactersInRegex: ANSI escape sequence is intentional
  return text.replace(/\u001b\[[0-9;]*m/g, "");
}
