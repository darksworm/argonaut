import { describe, expect, it, mock } from "bun:test";
import { render } from "ink-testing-library";
import { MainLayout } from "../../components/views/MainLayout";
import { AppStateProvider } from "../../contexts/AppStateContext";
import type { AppItem } from "../../types/domain";

// Test the successful authentication flow with clusters and apps displayed
describe("Main App Success UI Tests", () => {
  // Mock app data representing successful API responses
  const mockApps: AppItem[] = [
    {
      name: "frontend-app",
      sync: "Synced",
      health: "Healthy",
      project: "web-platform",
      clusterLabel: "production-cluster",
      namespace: "frontend",
      appNamespace: "argocd",
      lastSyncAt: "2024-01-15T10:30:00Z",
    },
    {
      name: "backend-api",
      sync: "OutOfSync",
      health: "Degraded",
      project: "web-platform",
      clusterLabel: "production-cluster",
      namespace: "backend",
      appNamespace: "argocd",
      lastSyncAt: "2024-01-15T09:45:00Z",
    },
    {
      name: "database",
      sync: "Synced",
      health: "Healthy",
      project: "infrastructure",
      clusterLabel: "production-cluster",
      namespace: "database",
      appNamespace: "argocd",
      lastSyncAt: "2024-01-15T10:15:00Z",
    },
    {
      name: "staging-app",
      sync: "Synced",
      health: "Progressing",
      project: "web-platform",
      clusterLabel: "staging-cluster",
      namespace: "staging",
      appNamespace: "argocd",
      lastSyncAt: "2024-01-15T10:20:00Z",
    },
  ];

  const mockServer = {
    config: {
      baseUrl: "https://argocd.production.com",
      insecure: false,
    },
    token: "mock-jwt-token-12345",
  };

  const successfulAppState = {
    mode: "normal" as const,
    server: mockServer,
    apps: mockApps,
    apiVersion: "v2.8.0",
    terminal: { rows: 30, cols: 120 },
    navigation: { view: "clusters" as const, selectedIdx: 0, lastGPressed: 0 },
  };

  // Mock props for MainLayout
  const defaultMainLayoutProps = {
    visibleItems: [],
    onDrillDown: mock(),
    commandRegistry: { getCommands: () => [] },
    onExecuteCommand: mock(),
    status: "Ready",
    modal: null,
  };

  describe("Clusters View", () => {
    it("displays available clusters from app data", () => {
      const clustersState = {
        ...successfulAppState,
        navigation: {
          view: "clusters" as const,
          selectedIdx: 0,
          lastGPressed: 0,
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={clustersState}>
          <MainLayout
            {...defaultMainLayoutProps}
            visibleItems={["production-cluster", "staging-cluster"]}
          />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show header with server info
      expect(frame).toContain("argocd.production.com");
      expect(frame).toContain("v2.8.0");

      // Should show clusters view header
      expect(frame).toContain("NAME");

      // Should show available clusters
      expect(frame).toContain("production-cluster");
      expect(frame).toContain("staging-cluster");

      // Should show status and navigation
      expect(frame).toContain("Ready");
      expect(frame).toContain("<clusters>");
    });

    it("handles cluster selection and highlighting", () => {
      const clustersState = {
        ...successfulAppState,
        navigation: {
          view: "clusters" as const,
          selectedIdx: 1,
          lastGPressed: 0,
        },
        selections: {
          scopeClusters: new Set(["staging-cluster"]),
          scopeNamespaces: new Set(),
          scopeProjects: new Set(),
          selectedApps: new Set(),
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={clustersState}>
          <MainLayout
            {...defaultMainLayoutProps}
            visibleItems={["production-cluster", "staging-cluster"]}
          />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show both clusters
      expect(frame).toContain("production-cluster");
      expect(frame).toContain("staging-cluster");

      // Should show navigation position (2/2 means second item selected)
      expect(frame).toContain("2/2");
    });
  });

  describe("Applications View", () => {
    it("displays applications with health and sync status", () => {
      const appsState = {
        ...successfulAppState,
        navigation: { view: "apps" as const, selectedIdx: 0, lastGPressed: 0 },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={appsState}>
          <MainLayout {...defaultMainLayoutProps} visibleItems={mockApps} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show apps view headers
      expect(frame).toContain("NAME");
      expect(frame).toContain("SYNC");
      expect(frame).toContain("HEALTH");

      // Should show application names
      expect(frame).toContain("frontend-app");
      expect(frame).toContain("backend-api");
      expect(frame).toContain("database");
      expect(frame).toContain("staging-app");

      // Should show sync status (could be with ANSI codes)
      expect(frame).toContain("Synced");
      expect(frame).toContain("OutOfSync");

      // Should show health status
      expect(frame).toContain("Healthy");
      expect(frame).toContain("Degraded");
      expect(frame).toContain("Progressing");

      // Should show navigation info
      expect(frame).toContain("<apps>");
      expect(frame).toContain("1/4"); // First of 4 apps selected
    });

    it("handles application selection", () => {
      const appsState = {
        ...successfulAppState,
        navigation: { view: "apps" as const, selectedIdx: 1, lastGPressed: 0 },
        selections: {
          scopeClusters: new Set(),
          scopeNamespaces: new Set(),
          scopeProjects: new Set(),
          selectedApps: new Set(["backend-api"]),
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={appsState}>
          <MainLayout {...defaultMainLayoutProps} visibleItems={mockApps} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show all apps
      expect(frame).toContain("frontend-app");
      expect(frame).toContain("backend-api");
      expect(frame).toContain("database");

      // Should show correct position
      expect(frame).toContain("2/4"); // Second of 4 apps selected
    });

    it("displays apps filtered by cluster selection", () => {
      const filteredAppsState = {
        ...successfulAppState,
        navigation: { view: "apps" as const, selectedIdx: 0, lastGPressed: 0 },
        selections: {
          scopeClusters: new Set(["staging-cluster"]),
          scopeNamespaces: new Set(),
          scopeProjects: new Set(),
          selectedApps: new Set(),
        },
      };

      // Only staging cluster apps
      const stagingApps = mockApps.filter(
        (app) => app.clusterLabel === "staging-cluster",
      );

      const { lastFrame } = render(
        <AppStateProvider initialState={filteredAppsState}>
          <MainLayout {...defaultMainLayoutProps} visibleItems={stagingApps} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show staging app
      expect(frame).toContain("staging-app");

      // Should NOT show production apps
      expect(frame).not.toContain("frontend-app");
      expect(frame).not.toContain("backend-api");
      expect(frame).not.toContain("database");

      // Should show correct count
      expect(frame).toContain("1/1");
    });
  });

  describe("Projects and Namespaces Views", () => {
    it("displays projects from app data", () => {
      const projectsState = {
        ...successfulAppState,
        navigation: {
          view: "projects" as const,
          selectedIdx: 0,
          lastGPressed: 0,
        },
      };

      const projects = ["web-platform", "infrastructure"];

      const { lastFrame } = render(
        <AppStateProvider initialState={projectsState}>
          <MainLayout {...defaultMainLayoutProps} visibleItems={projects} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show projects
      expect(frame).toContain("web-platform");
      expect(frame).toContain("infrastructure");

      // Should show projects view
      expect(frame).toContain("<projects>");
      expect(frame).toContain("1/2");
    });

    it("displays namespaces from app data", () => {
      const namespacesState = {
        ...successfulAppState,
        navigation: {
          view: "namespaces" as const,
          selectedIdx: 0,
          lastGPressed: 0,
        },
      };

      const namespaces = ["frontend", "backend", "database", "staging"];

      const { lastFrame } = render(
        <AppStateProvider initialState={namespacesState}>
          <MainLayout {...defaultMainLayoutProps} visibleItems={namespaces} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show namespaces
      expect(frame).toContain("frontend");
      expect(frame).toContain("backend");
      expect(frame).toContain("database");
      expect(frame).toContain("staging");

      // Should show namespaces view
      expect(frame).toContain("<namespaces>");
      expect(frame).toContain("1/4");
    });
  });

  describe("Server and Context Information", () => {
    it("displays server connection information", () => {
      const { lastFrame } = render(
        <AppStateProvider initialState={successfulAppState}>
          <MainLayout
            {...defaultMainLayoutProps}
            visibleItems={["production-cluster"]}
          />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show server information
      expect(frame).toContain("argocd.production.com");
      expect(frame).toContain("v2.8.0");
      expect(frame).toContain("Ready"); // Connection status
    });

    it("displays scope and context information", () => {
      const scopedState = {
        ...successfulAppState,
        selections: {
          scopeClusters: new Set(["production-cluster"]),
          scopeNamespaces: new Set(["frontend"]),
          scopeProjects: new Set(["web-platform"]),
          selectedApps: new Set(),
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={scopedState}>
          <MainLayout {...defaultMainLayoutProps} visibleItems={[]} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show context information (might be in banner)
      expect(frame).toContain("production-cluster");
      expect(frame).toContain("frontend");
      expect(frame).toContain("web-platform");
    });
  });

  describe("Terminal Responsiveness", () => {
    it("handles wide terminal layout", () => {
      const wideTerminalState = {
        ...successfulAppState,
        terminal: { rows: 40, cols: 160 },
        navigation: { view: "apps" as const, selectedIdx: 0, lastGPressed: 0 },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={wideTerminalState}>
          <MainLayout {...defaultMainLayoutProps} visibleItems={mockApps} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should render without crashing
      expect(frame).toBeDefined();
      expect(frame.length).toBeGreaterThan(0);

      // Should show app data
      expect(frame).toContain("frontend-app");
      expect(frame).toContain("Healthy");
    });

    it("handles narrow terminal layout", () => {
      const narrowTerminalState = {
        ...successfulAppState,
        terminal: { rows: 20, cols: 60 },
        navigation: { view: "apps" as const, selectedIdx: 0, lastGPressed: 0 },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={narrowTerminalState}>
          <MainLayout {...defaultMainLayoutProps} visibleItems={mockApps} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should render without crashing
      expect(frame).toBeDefined();
      expect(frame.length).toBeGreaterThan(0);

      // Content might be truncated but should be present
      expect(frame).toContain("frontend-app");
    });
  });

  describe("Empty States", () => {
    it("handles empty app list gracefully", () => {
      const emptyAppsState = {
        ...successfulAppState,
        apps: [],
        navigation: { view: "apps" as const, selectedIdx: 0, lastGPressed: 0 },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={emptyAppsState}>
          <MainLayout {...defaultMainLayoutProps} visibleItems={[]} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show headers
      expect(frame).toContain("NAME");
      expect(frame).toContain("SYNC");
      expect(frame).toContain("HEALTH");

      // Should show empty state message
      expect(frame).toContain("No items");

      // Should show 0/0 navigation
      expect(frame).toContain("0/0");
    });

    it("handles no clusters available", () => {
      const noClustersState = {
        ...successfulAppState,
        navigation: {
          view: "clusters" as const,
          selectedIdx: 0,
          lastGPressed: 0,
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={noClustersState}>
          <MainLayout {...defaultMainLayoutProps} visibleItems={[]} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show header
      expect(frame).toContain("NAME");

      // Should show empty state
      expect(frame).toContain("No items");
      expect(frame).toContain("0/0");
    });
  });
});
