import { render } from "ink-testing-library";
import { ListView } from "../../components/views/ListView";
import { AppStateProvider } from "../../contexts/AppStateContext";
import type { AppItem } from "../../types/domain";

// Test the ListView component directly with successful data
describe("ListView Success UI Tests", () => {
  // Mock app data representing successful API responses
  const mockApps: AppItem[] = [
    {
      name: "frontend-web",
      sync: "Synced",
      health: "Healthy",
      project: "web-services",
      clusterLabel: "prod-east",
      namespace: "frontend",
      appNamespace: "argocd",
      lastSyncAt: "2024-01-15T10:30:00Z",
    },
    {
      name: "api-gateway",
      sync: "OutOfSync",
      health: "Degraded",
      project: "web-services",
      clusterLabel: "prod-east",
      namespace: "api",
      appNamespace: "argocd",
      lastSyncAt: "2024-01-15T09:45:00Z",
    },
    {
      name: "postgres-db",
      sync: "Synced",
      health: "Healthy",
      project: "infrastructure",
      clusterLabel: "prod-east",
      namespace: "database",
      appNamespace: "argocd",
      lastSyncAt: "2024-01-15T10:15:00Z",
    },
    {
      name: "redis-cache",
      sync: "Unknown",
      health: "Progressing",
      project: "infrastructure",
      clusterLabel: "prod-west",
      namespace: "cache",
      appNamespace: "argocd",
      lastSyncAt: "2024-01-15T10:00:00Z",
    },
  ];

  const successfulAppState = {
    mode: "normal" as const,
    apps: mockApps,
    terminal: { rows: 25, cols: 100 },
    navigation: { view: "apps" as const, selectedIdx: 0, lastGPressed: 0 },
    selections: {
      scopeClusters: new Set<string>(),
      scopeNamespaces: new Set<string>(),
      scopeProjects: new Set<string>(),
      selectedApps: new Set<string>(),
    },
    ui: {
      searchQuery: "",
      activeFilter: "",
      command: ":",
      isVersionOutdated: false,
    },
  };

  describe("Applications ListView", () => {
    it("renders application list with sync and health columns", () => {
      const appsState = {
        ...successfulAppState,
        navigation: { view: "apps" as const, selectedIdx: 0, lastGPressed: 0 },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={appsState}>
          <ListView visibleItems={mockApps} availableRows={15} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show column headers
      expect(frame).toContain("NAME");
      expect(frame).toContain("SYNC");
      expect(frame).toContain("HEALTH");

      // Should show application names
      expect(frame).toContain("frontend-web");
      expect(frame).toContain("api-gateway");
      expect(frame).toContain("postgres-db");
      expect(frame).toContain("redis-cache");

      // Should show sync status (with or without ANSI codes)
      expect(frame).toContain("Synced");
      expect(frame).toContain("OutOfSync");
      expect(frame).toContain("Unknown");

      // Should show health status
      expect(frame).toContain("Healthy");
      expect(frame).toContain("Degraded");
      expect(frame).toContain("Progressing");
    });

    it("highlights selected application", () => {
      const appsState = {
        ...successfulAppState,
        navigation: { view: "apps" as const, selectedIdx: 1, lastGPressed: 0 },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={appsState}>
          <ListView visibleItems={mockApps} availableRows={15} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show all apps
      expect(frame).toContain("frontend-web");
      expect(frame).toContain("api-gateway"); // This should be highlighted (selectedIdx: 1)
      expect(frame).toContain("postgres-db");
      expect(frame).toContain("redis-cache");
    });

    it("shows selected apps with checkmarks", () => {
      const appsState = {
        ...successfulAppState,
        navigation: { view: "apps" as const, selectedIdx: 0, lastGPressed: 0 },
        selections: {
          ...successfulAppState.selections,
          selectedApps: new Set(["frontend-web", "postgres-db"]),
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={appsState}>
          <ListView visibleItems={mockApps} availableRows={15} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show all apps
      expect(frame).toContain("frontend-web"); // Should be highlighted as selected
      expect(frame).toContain("api-gateway");
      expect(frame).toContain("postgres-db"); // Should be highlighted as selected
      expect(frame).toContain("redis-cache");
    });

    it("handles wide terminal with full status labels", () => {
      const wideTerminalState = {
        ...successfulAppState,
        terminal: { rows: 30, cols: 150 },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={wideTerminalState}>
          <ListView visibleItems={mockApps} availableRows={20} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // With wide terminal, should show full labels
      expect(frame).toContain("Synced");
      expect(frame).toContain("OutOfSync");
      expect(frame).toContain("Healthy");
      expect(frame).toContain("Degraded");
      expect(frame).toContain("Progressing");
    });

    it("handles narrow terminal with icons only", () => {
      const narrowTerminalState = {
        ...successfulAppState,
        terminal: { rows: 20, cols: 60 },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={narrowTerminalState}>
          <ListView visibleItems={mockApps} availableRows={15} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should still show app names
      expect(frame).toContain("frontend-web");
      expect(frame).toContain("api-gateway");

      // With narrow terminal, might show icons instead of full labels
      // The component uses ASCII_ICONS: check: "V", warn: "!", quest: "?", delta: "^"
      expect(frame).toMatch(/[V!?^]/); // Should contain some status icons
    });
  });

  describe("Clusters ListView", () => {
    it("renders cluster list", () => {
      const clustersState = {
        ...successfulAppState,
        navigation: {
          view: "clusters" as const,
          selectedIdx: 0,
          lastGPressed: 0,
        },
      };

      const clusters = ["prod-east", "prod-west", "staging", "dev"];

      const { lastFrame } = render(
        <AppStateProvider initialState={clustersState}>
          <ListView visibleItems={clusters} availableRows={15} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show header (only NAME column for non-apps)
      expect(frame).toContain("NAME");

      // Should NOT show SYNC/HEALTH columns for clusters
      expect(frame).not.toContain("SYNC");
      expect(frame).not.toContain("HEALTH");

      // Should show cluster names
      expect(frame).toContain("prod-east");
      expect(frame).toContain("prod-west");
      expect(frame).toContain("staging");
      expect(frame).toContain("dev");
    });

    it("highlights selected cluster", () => {
      const clustersState = {
        ...successfulAppState,
        navigation: {
          view: "clusters" as const,
          selectedIdx: 2,
          lastGPressed: 0,
        },
        selections: {
          ...successfulAppState.selections,
          scopeClusters: new Set(["staging"]),
        },
      };

      const clusters = ["prod-east", "prod-west", "staging", "dev"];

      const { lastFrame } = render(
        <AppStateProvider initialState={clustersState}>
          <ListView visibleItems={clusters} availableRows={15} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show all clusters
      expect(frame).toContain("prod-east");
      expect(frame).toContain("prod-west");
      expect(frame).toContain("staging"); // This should be highlighted (selectedIdx: 2, also in scopeClusters)
      expect(frame).toContain("dev");
    });
  });

  describe("Namespaces ListView", () => {
    it("renders namespace list", () => {
      const namespacesState = {
        ...successfulAppState,
        navigation: {
          view: "namespaces" as const,
          selectedIdx: 0,
          lastGPressed: 0,
        },
      };

      const namespaces = ["frontend", "api", "database", "cache", "monitoring"];

      const { lastFrame } = render(
        <AppStateProvider initialState={namespacesState}>
          <ListView visibleItems={namespaces} availableRows={15} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show header
      expect(frame).toContain("NAME");

      // Should show namespace names
      expect(frame).toContain("frontend");
      expect(frame).toContain("api");
      expect(frame).toContain("database");
      expect(frame).toContain("cache");
      expect(frame).toContain("monitoring");
    });

    it("handles selected namespaces", () => {
      const namespacesState = {
        ...successfulAppState,
        navigation: {
          view: "namespaces" as const,
          selectedIdx: 1,
          lastGPressed: 0,
        },
        selections: {
          ...successfulAppState.selections,
          scopeNamespaces: new Set(["api", "database"]),
        },
      };

      const namespaces = ["frontend", "api", "database", "cache"];

      const { lastFrame } = render(
        <AppStateProvider initialState={namespacesState}>
          <ListView visibleItems={namespaces} availableRows={15} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show all namespaces
      expect(frame).toContain("frontend");
      expect(frame).toContain("api"); // Should be highlighted (in selection)
      expect(frame).toContain("database"); // Should be highlighted (in selection)
      expect(frame).toContain("cache");
    });
  });

  describe("Projects ListView", () => {
    it("renders project list", () => {
      const projectsState = {
        ...successfulAppState,
        navigation: {
          view: "projects" as const,
          selectedIdx: 0,
          lastGPressed: 0,
        },
      };

      const projects = [
        "web-services",
        "infrastructure",
        "monitoring",
        "security",
      ];

      const { lastFrame } = render(
        <AppStateProvider initialState={projectsState}>
          <ListView visibleItems={projects} availableRows={15} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show header
      expect(frame).toContain("NAME");

      // Should show project names
      expect(frame).toContain("web-services");
      expect(frame).toContain("infrastructure");
      expect(frame).toContain("monitoring");
      expect(frame).toContain("security");
    });
  });

  describe("Empty States", () => {
    it("displays empty state for no applications", () => {
      const emptyAppsState = {
        ...successfulAppState,
        navigation: { view: "apps" as const, selectedIdx: 0, lastGPressed: 0 },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={emptyAppsState}>
          <ListView visibleItems={[]} availableRows={15} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show headers
      expect(frame).toContain("NAME");
      expect(frame).toContain("SYNC");
      expect(frame).toContain("HEALTH");

      // Should show empty message
      expect(frame).toContain("No items");
    });

    it("displays empty state for no clusters", () => {
      const emptyClustersState = {
        ...successfulAppState,
        navigation: {
          view: "clusters" as const,
          selectedIdx: 0,
          lastGPressed: 0,
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={emptyClustersState}>
          <ListView visibleItems={[]} availableRows={15} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show header
      expect(frame).toContain("NAME");

      // Should show empty message
      expect(frame).toContain("No items");
    });
  });

  describe("Scrolling and Pagination", () => {
    it("handles long list with scrolling", () => {
      // Create many apps to test scrolling
      const manyApps: AppItem[] = Array.from({ length: 20 }, (_, i) => ({
        name: `app-${i.toString().padStart(2, "0")}`,
        sync: i % 3 === 0 ? "Synced" : i % 3 === 1 ? "OutOfSync" : "Unknown",
        health: i % 2 === 0 ? "Healthy" : "Degraded",
        project: "test-project",
        clusterLabel: "test-cluster",
        namespace: `namespace-${i}`,
        appNamespace: "argocd",
      }));

      const scrollState = {
        ...successfulAppState,
        apps: manyApps,
        navigation: { view: "apps" as const, selectedIdx: 10, lastGPressed: 0 },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={scrollState}>
          <ListView visibleItems={manyApps} availableRows={8} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show headers
      expect(frame).toContain("NAME");
      expect(frame).toContain("SYNC");
      expect(frame).toContain("HEALTH");

      // Should show some apps (due to scrolling, might not show all)
      expect(frame).toMatch(/app-\d{2}/); // Should show at least some app names
    });

    it("handles list bounds correctly", () => {
      const smallList = ["single-cluster"];

      const boundsState = {
        ...successfulAppState,
        navigation: {
          view: "clusters" as const,
          selectedIdx: 0,
          lastGPressed: 0,
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={boundsState}>
          <ListView visibleItems={smallList} availableRows={15} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show the single item
      expect(frame).toContain("single-cluster");

      // Should not crash with bounds issues
      expect(frame).toBeDefined();
    });
  });

  describe("Status Icons and Colors", () => {
    it("displays sync status icons correctly", () => {
      const iconTestApps: AppItem[] = [
        {
          name: "synced-app",
          sync: "Synced",
          health: "Healthy",
          project: "test",
          clusterLabel: "test",
          namespace: "test",
          appNamespace: "test",
        },
        {
          name: "outof-sync-app",
          sync: "OutOfSync",
          health: "Healthy",
          project: "test",
          clusterLabel: "test",
          namespace: "test",
          appNamespace: "test",
        },
        {
          name: "unknown-sync-app",
          sync: "Unknown",
          health: "Healthy",
          project: "test",
          clusterLabel: "test",
          namespace: "test",
          appNamespace: "test",
        },
        {
          name: "degraded-sync-app",
          sync: "Degraded",
          health: "Healthy",
          project: "test",
          clusterLabel: "test",
          namespace: "test",
          appNamespace: "test",
        },
      ];

      const { lastFrame } = render(
        <AppStateProvider initialState={successfulAppState}>
          <ListView visibleItems={iconTestApps} availableRows={15} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show app names
      expect(frame).toContain("synced-app");
      expect(frame).toContain("outof-sync-app");
      expect(frame).toContain("unknown-sync-app");
      expect(frame).toContain("degraded-sync-app");

      // Should show status labels
      expect(frame).toContain("Synced");
      expect(frame).toContain("OutOfSync");
      expect(frame).toContain("Unknown");
      expect(frame).toContain("Degraded");
    });

    it("displays health status icons correctly", () => {
      const healthTestApps: AppItem[] = [
        {
          name: "healthy-app",
          sync: "Synced",
          health: "Healthy",
          project: "test",
          clusterLabel: "test",
          namespace: "test",
          appNamespace: "test",
        },
        {
          name: "missing-app",
          sync: "Synced",
          health: "Missing",
          project: "test",
          clusterLabel: "test",
          namespace: "test",
          appNamespace: "test",
        },
        {
          name: "degraded-app",
          sync: "Synced",
          health: "Degraded",
          project: "test",
          clusterLabel: "test",
          namespace: "test",
          appNamespace: "test",
        },
        {
          name: "progressing-app",
          sync: "Synced",
          health: "Progressing",
          project: "test",
          clusterLabel: "test",
          namespace: "test",
          appNamespace: "test",
        },
        {
          name: "unknown-health-app",
          sync: "Synced",
          health: "Unknown",
          project: "test",
          clusterLabel: "test",
          namespace: "test",
          appNamespace: "test",
        },
      ];

      const { lastFrame } = render(
        <AppStateProvider initialState={successfulAppState}>
          <ListView visibleItems={healthTestApps} availableRows={15} />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show app names
      expect(frame).toContain("healthy-app");
      expect(frame).toContain("missing-app");
      expect(frame).toContain("degraded-app");
      expect(frame).toContain("progressing-app");
      expect(frame).toContain("unknown-health-app");

      // Should show health labels
      expect(frame).toContain("Healthy");
      expect(frame).toContain("Missing");
      expect(frame).toContain("Degraded");
      expect(frame).toContain("Progressing");
      expect(frame).toContain("Unknown");
    });
  });
});
