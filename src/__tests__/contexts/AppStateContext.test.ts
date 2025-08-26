// src/__tests__/contexts/AppStateContext.test.ts
import {
  type AppAction,
  type AppState,
  appStateReducer,
  initialState,
} from "../../contexts/AppStateContext";
import type { Mode, View } from "../../types/domain";

describe("appStateReducer", () => {
  describe("navigation state", () => {
    it("should handle SET_MODE transitions", () => {
      const action: AppAction = { type: "SET_MODE", payload: "normal" as Mode };
      const newState = appStateReducer(initialState, action);

      expect(newState.mode).toBe("normal");
      expect(newState).not.toBe(initialState); // immutability check
    });

    it("should handle SET_VIEW with proper state update", () => {
      const action: AppAction = { type: "SET_VIEW", payload: "apps" as View };
      const newState = appStateReducer(initialState, action);

      expect(newState.navigation.view).toBe("apps");
      expect(newState.navigation.selectedIdx).toBe(
        initialState.navigation.selectedIdx,
      );
    });

    it("should update selectedIdx within valid range", () => {
      const action: AppAction = { type: "SET_SELECTED_IDX", payload: 5 };
      const newState = appStateReducer(initialState, action);

      expect(newState.navigation.selectedIdx).toBe(5);
    });

    it("should handle vim navigation timing (lastGPressed)", () => {
      const timestamp = Date.now();
      const action: AppAction = {
        type: "SET_LAST_G_PRESSED",
        payload: timestamp,
      };
      const newState = appStateReducer(initialState, action);

      expect(newState.navigation.lastGPressed).toBe(timestamp);
    });

    it("should reset navigation with RESET_NAVIGATION", () => {
      const currentState: AppState = {
        ...initialState,
        navigation: { ...initialState.navigation, selectedIdx: 10 },
        ui: { ...initialState.ui, activeFilter: "test", searchQuery: "search" },
      };

      const action: AppAction = { type: "RESET_NAVIGATION" };
      const newState = appStateReducer(currentState, action);

      expect(newState.navigation.selectedIdx).toBe(0);
      expect(newState.ui.activeFilter).toBe("");
      expect(newState.ui.searchQuery).toBe("");
    });

    it("should reset navigation with specific view", () => {
      const action: AppAction = {
        type: "RESET_NAVIGATION",
        payload: { view: "apps" as View },
      };
      const newState = appStateReducer(initialState, action);

      expect(newState.navigation.view).toBe("apps");
      expect(newState.navigation.selectedIdx).toBe(0);
    });
  });

  describe("selection management", () => {
    it("should clear lower-level selections when drilling up from clusters view", () => {
      const currentState: AppState = {
        ...initialState,
        selections: {
          ...initialState.selections,
          scopeClusters: new Set(["cluster1"]),
          scopeNamespaces: new Set(["ns1", "ns2"]),
          scopeProjects: new Set(["proj1"]),
          selectedApps: new Set(["app1", "app2"]),
        },
      };

      const action: AppAction = {
        type: "CLEAR_LOWER_LEVEL_SELECTIONS",
        payload: "clusters" as View,
      };
      const newState = appStateReducer(currentState, action);

      expect(newState.selections.scopeClusters).toEqual(new Set(["cluster1"]));
      expect(newState.selections.scopeNamespaces).toEqual(new Set());
      expect(newState.selections.scopeProjects).toEqual(new Set());
      expect(newState.selections.selectedApps).toEqual(new Set());
    });

    it("should clear lower-level selections when drilling up from namespaces view", () => {
      const currentState: AppState = {
        ...initialState,
        selections: {
          ...initialState.selections,
          scopeClusters: new Set(["cluster1"]),
          scopeNamespaces: new Set(["ns1"]),
          scopeProjects: new Set(["proj1"]),
          selectedApps: new Set(["app1"]),
        },
      };

      const action: AppAction = {
        type: "CLEAR_LOWER_LEVEL_SELECTIONS",
        payload: "namespaces" as View,
      };
      const newState = appStateReducer(currentState, action);

      expect(newState.selections.scopeClusters).toEqual(new Set(["cluster1"]));
      expect(newState.selections.scopeNamespaces).toEqual(new Set(["ns1"]));
      expect(newState.selections.scopeProjects).toEqual(new Set());
      expect(newState.selections.selectedApps).toEqual(new Set());
    });

    it("should clear lower-level selections when drilling up from projects view", () => {
      const currentState: AppState = {
        ...initialState,
        selections: {
          ...initialState.selections,
          scopeProjects: new Set(["proj1"]),
          selectedApps: new Set(["app1", "app2"]),
        },
      };

      const action: AppAction = {
        type: "CLEAR_LOWER_LEVEL_SELECTIONS",
        payload: "projects" as View,
      };
      const newState = appStateReducer(currentState, action);

      expect(newState.selections.scopeProjects).toEqual(new Set(["proj1"]));
      expect(newState.selections.selectedApps).toEqual(new Set());
    });

    it("should handle scope selections (clusters)", () => {
      const clusters = new Set(["cluster1", "cluster2"]);
      const action: AppAction = {
        type: "SET_SCOPE_CLUSTERS",
        payload: clusters,
      };
      const newState = appStateReducer(initialState, action);

      expect(newState.selections.scopeClusters).toEqual(clusters);
    });

    it("should handle scope selections (namespaces)", () => {
      const namespaces = new Set(["default", "kube-system"]);
      const action: AppAction = {
        type: "SET_SCOPE_NAMESPACES",
        payload: namespaces,
      };
      const newState = appStateReducer(initialState, action);

      expect(newState.selections.scopeNamespaces).toEqual(namespaces);
    });

    it("should handle scope selections (projects)", () => {
      const projects = new Set(["default", "team-a"]);
      const action: AppAction = {
        type: "SET_SCOPE_PROJECTS",
        payload: projects,
      };
      const newState = appStateReducer(initialState, action);

      expect(newState.selections.scopeProjects).toEqual(projects);
    });

    it("should toggle app selections correctly", () => {
      const apps = new Set(["app1", "app2"]);
      const action: AppAction = { type: "SET_SELECTED_APPS", payload: apps };
      const newState = appStateReducer(initialState, action);

      expect(newState.selections.selectedApps).toEqual(apps);
    });

    it("should clear all selections", () => {
      const currentState: AppState = {
        ...initialState,
        selections: {
          scopeClusters: new Set(["cluster1"]),
          scopeNamespaces: new Set(["ns1"]),
          scopeProjects: new Set(["proj1"]),
          selectedApps: new Set(["app1"]),
        },
      };

      const action: AppAction = { type: "CLEAR_ALL_SELECTIONS" };
      const newState = appStateReducer(currentState, action);

      expect(newState.selections.scopeClusters).toEqual(new Set());
      expect(newState.selections.scopeNamespaces).toEqual(new Set());
      expect(newState.selections.scopeProjects).toEqual(new Set());
      expect(newState.selections.selectedApps).toEqual(new Set());
    });
  });

  describe("UI state", () => {
    it("should handle search query updates", () => {
      const action: AppAction = { type: "SET_SEARCH_QUERY", payload: "my-app" };
      const newState = appStateReducer(initialState, action);

      expect(newState.ui.searchQuery).toBe("my-app");
    });

    it("should handle active filter updates", () => {
      const action: AppAction = {
        type: "SET_ACTIVE_FILTER",
        payload: "OutOfSync",
      };
      const newState = appStateReducer(initialState, action);

      expect(newState.ui.activeFilter).toBe("OutOfSync");
    });

    it("should handle command input state", () => {
      const action: AppAction = { type: "SET_COMMAND", payload: ":sync myapp" };
      const newState = appStateReducer(initialState, action);

      expect(newState.ui.command).toBe(":sync myapp");
    });

    it("should handle terminal resize events", () => {
      const action: AppAction = {
        type: "SET_TERMINAL_SIZE",
        payload: { rows: 50, cols: 120 },
      };
      const newState = appStateReducer(initialState, action);

      expect(newState.terminal.rows).toBe(50);
      expect(newState.terminal.cols).toBe(120);
    });

    it("should handle version outdated flag", () => {
      const action: AppAction = { type: "SET_VERSION_OUTDATED", payload: true };
      const newState = appStateReducer(initialState, action);

      expect(newState.ui.isVersionOutdated).toBe(true);
    });

    it("should handle latest version update", () => {
      const action: AppAction = {
        type: "SET_LATEST_VERSION",
        payload: "2.0.0",
      };
      const newState = appStateReducer(initialState, action);

      expect(newState.ui.latestVersion).toBe("2.0.0");
    });

    it("should clear filters", () => {
      const currentState: AppState = {
        ...initialState,
        ui: {
          ...initialState.ui,
          activeFilter: "OutOfSync",
          searchQuery: "test-app",
        },
      };

      const action: AppAction = { type: "CLEAR_FILTERS" };
      const newState = appStateReducer(currentState, action);

      expect(newState.ui.activeFilter).toBe("");
      expect(newState.ui.searchQuery).toBe("");
    });
  });

  describe("modal state", () => {
    it("should set confirm sync target and options", () => {
      const action: AppAction = {
        type: "SET_CONFIRM_TARGET",
        payload: "my-app",
      };
      const newState = appStateReducer(initialState, action);

      expect(newState.modals.confirmTarget).toBe("my-app");
    });

    it("should handle sync prune option", () => {
      const action: AppAction = {
        type: "SET_CONFIRM_SYNC_PRUNE",
        payload: true,
      };
      const newState = appStateReducer(initialState, action);

      expect(newState.modals.confirmSyncPrune).toBe(true);
    });

    it("should handle sync watch option", () => {
      const action: AppAction = {
        type: "SET_CONFIRM_SYNC_WATCH",
        payload: false,
      };
      const newState = appStateReducer(initialState, action);

      expect(newState.modals.confirmSyncWatch).toBe(false);
    });

    it("should handle rollback app name", () => {
      const action: AppAction = {
        type: "SET_ROLLBACK_APP_NAME",
        payload: "rollback-app",
      };
      const newState = appStateReducer(initialState, action);

      expect(newState.modals.rollbackAppName).toBe("rollback-app");
    });

    it("should manage resource view app", () => {
      const action: AppAction = {
        type: "SET_SYNC_VIEW_APP",
        payload: "view-app",
      };
      const newState = appStateReducer(initialState, action);

      expect(newState.modals.syncViewApp).toBe("view-app");
    });
  });

  describe("server and data", () => {
    it("should handle server authentication state", () => {
      const server = {
        config: {
          baseUrl: "https://argocd.example.com",
        },
        token: "test-token",
      };
      const action: AppAction = { type: "SET_SERVER", payload: server };
      const newState = appStateReducer(initialState, action);

      expect(newState.server).toEqual(server);
    });

    it("should update apps array", () => {
      const apps = [
        {
          name: "app1",
          sync: "Synced",
          health: "Healthy",
          clusterId: "cluster1",
          clusterLabel: "cluster1",
          namespace: "default",
          appNamespace: "argocd",
          project: "default",
          lastSyncAt: "2023-12-01T10:00:00Z",
        },
      ];
      const action: AppAction = { type: "SET_APPS", payload: apps };
      const newState = appStateReducer(initialState, action);

      expect(newState.apps).toEqual(apps);
    });

    it("should update API version", () => {
      const action: AppAction = { type: "SET_API_VERSION", payload: "v2.9.0" };
      const newState = appStateReducer(initialState, action);

      expect(newState.apiVersion).toBe("v2.9.0");
    });

    it("should manage loading abort controller", () => {
      const controller = new AbortController();
      const action: AppAction = {
        type: "SET_LOADING_ABORT_CONTROLLER",
        payload: controller,
      };
      const newState = appStateReducer(initialState, action);

      expect(newState.loadingAbortController).toBe(controller);
    });
  });

  describe("state immutability", () => {
    it("should not mutate original state for SET_MODE", () => {
      const originalState = { ...initialState };
      const action: AppAction = { type: "SET_MODE", payload: "normal" as Mode };
      const newState = appStateReducer(initialState, action);

      expect(initialState).toEqual(originalState);
      expect(newState).not.toBe(initialState);
    });

    it("should not mutate nested objects for SET_TERMINAL_SIZE", () => {
      const originalTerminal = { ...initialState.terminal };
      const action: AppAction = {
        type: "SET_TERMINAL_SIZE",
        payload: { rows: 30, cols: 100 },
      };
      const newState = appStateReducer(initialState, action);

      expect(initialState.terminal).toEqual(originalTerminal);
      expect(newState.terminal).not.toBe(initialState.terminal);
    });

    it("should not mutate selections for SET_SELECTED_APPS", () => {
      const originalSelections = { ...initialState.selections };
      const apps = new Set(["app1"]);
      const action: AppAction = { type: "SET_SELECTED_APPS", payload: apps };
      const newState = appStateReducer(initialState, action);

      expect(initialState.selections).toEqual(originalSelections);
      expect(newState.selections).not.toBe(initialState.selections);
    });
  });

  describe("unknown actions", () => {
    it("should return current state for unknown actions", () => {
      // @ts-expect-error - Testing unknown action type
      const action: AppAction = { type: "UNKNOWN_ACTION", payload: "test" };
      const newState = appStateReducer(initialState, action);

      expect(newState).toBe(initialState);
    });
  });
});
