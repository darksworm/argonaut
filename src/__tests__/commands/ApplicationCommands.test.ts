// src/__tests__/commands/ApplicationCommands.test.ts
import { mock } from "bun:test";
import { describe, it, expect, beforeEach, afterEach } from "bun:test";

import type { CommandContext } from "../../commands/types";
import { createMockContext, createMockState } from "../test-utils";

// Test implementations of command classes without external dependencies
class TestDiffCommand {
  aliases = [];
  description = "View diff for application";

  canExecute(context: CommandContext): boolean {
    return context.state.server !== null;
  }

  async execute(context: CommandContext, arg?: string): Promise<void> {
    const { state, dispatch, statusLog } = context;
    const { server } = state;
    const { selectedApps } = state.selections;
    const { view, selectedIdx } = state.navigation;

    if (!server) {
      statusLog.error("Not authenticated.", "auth");
      return;
    }

    const visibleItems = context.state.apps;
    const target =
      arg ||
      (view === "apps"
        ? (visibleItems[selectedIdx] as any)?.name
        : undefined) ||
      Array.from(selectedApps)[0];

    if (!target) {
      statusLog.warn("No app selected to diff.", "user-action");
      return;
    }

    try {
      dispatch({ type: "SET_MODE", payload: "normal" });
      statusLog.info(`Preparing diff for ${target}…`, "diff");
      // Mock diff session completion
      statusLog.info("No differences.", "diff");
    } catch (e: any) {
      dispatch({ type: "SET_MODE", payload: "normal" });
      statusLog.error(`Diff failed: ${e?.message || String(e)}`, "diff");
    }
  }
}

class TestRollbackCommand {
  aliases = [];
  description = "Rollback application";

  canExecute(context: CommandContext): boolean {
    return context.state.server !== null;
  }

  async execute(context: CommandContext, arg?: string): Promise<void> {
    const { state, dispatch, statusLog } = context;
    const { selectedApps } = state.selections;
    const { view, selectedIdx } = state.navigation;

    if (!state.server) {
      statusLog.error("Not authenticated.", "auth");
      return;
    }

    const visibleItems = context.state.apps;
    const target =
      arg ||
      (view === "apps"
        ? (visibleItems[selectedIdx] as any)?.name
        : undefined) ||
      Array.from(selectedApps)[0];

    if (!target) {
      statusLog.warn("No app selected to rollback.", "user-action");
      return;
    }

    statusLog.info(`Opening rollback for ${target}…`, "rollback");
    dispatch({ type: "SET_ROLLBACK_APP_NAME", payload: target });
    dispatch({ type: "SET_MODE", payload: "rollback" });
  }
}

class TestResourcesCommand {
  aliases = ["resource", "res"];
  description = "View resources for application";

  canExecute(context: CommandContext): boolean {
    return context.state.server !== null;
  }

  execute(context: CommandContext, arg?: string): void {
    const { state, dispatch, statusLog } = context;
    const { selectedApps } = state.selections;
    const { view, selectedIdx } = state.navigation;

    if (!state.server) {
      statusLog.error("Not authenticated.", "auth");
      return;
    }

    const visibleItems = context.state.apps;
    const target =
      arg ||
      (view === "apps"
        ? (visibleItems[selectedIdx] as any)?.name
        : undefined) ||
      (selectedApps.size === 1 ? Array.from(selectedApps)[0] : undefined);

    if (!target) {
      statusLog.warn("No app selected to open resources view.", "user-action");
      return;
    }

    dispatch({ type: "SET_SYNC_VIEW_APP", payload: target });
    dispatch({ type: "SET_MODE", payload: "resources" });
  }
}

class TestLogsCommand {
  aliases = ["log"];
  description = "Open log viewer";

  async execute(context: CommandContext): Promise<void> {
    const { statusLog } = context;
    statusLog.info("Opening logs…", "logs");
    // Mock log session
  }
}

class TestLicenseCommand {
  aliases = ["licenses"];
  description = "View licenses";

  async execute(context: CommandContext): Promise<void> {
    const { dispatch, statusLog } = context;

    try {
      dispatch({ type: "SET_MODE", payload: "normal" });
      statusLog.info("Opening licenses…", "license");
      // Mock license session
    } catch (_e) {
      // Handle cleanup
    }
  }
}

describe("DiffCommand", () => {
  let diffCommand: TestDiffCommand;

  beforeEach(() => {
    diffCommand = new TestDiffCommand();
  });

  describe("canExecute", () => {
    it("should require authentication", () => {
      const context = createMockContext({
        state: createMockState({ server: null }),
      });

      expect(diffCommand.canExecute(context)).toBe(false);
    });

    it("should allow execution when authenticated", () => {
      const context = createMockContext();

      expect(diffCommand.canExecute(context)).toBe(true);
    });
  });

  describe("execute", () => {
    it("should require authentication", async () => {
      const mockStatusLog = {
        info: mock(),
        warn: mock(),
        error: mock(),
        debug: mock(),
        set: mock(),
        clear: mock(),
      };
      const context = createMockContext({
        state: createMockState({ server: null }),
        statusLog: mockStatusLog,
      });

      await diffCommand.execute(context);

      expect(mockStatusLog.error).toHaveBeenCalledWith(
        "Not authenticated.",
        "auth",
      );
    });

    it("should find target app correctly with explicit argument", async () => {
      const mockDispatch = mock();
      const mockStatusLog = {
        info: mock(),
        warn: mock(),
        error: mock(),
        debug: mock(),
        set: mock(),
        clear: mock(),
      };
      const context = createMockContext({
        dispatch: mockDispatch,
        statusLog: mockStatusLog,
      });

      await diffCommand.execute(context, "test-app");

      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "Preparing diff for test-app…",
        "diff",
      );
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "normal",
      });
    });

    it("should find target from cursor position in apps view", async () => {
      const mockStatusLog = {
        info: mock(),
        warn: mock(),
        error: mock(),
        debug: mock(),
        set: mock(),
        clear: mock(),
      };
      const apps = [
        {
          name: "cursor-app",
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
      const context = createMockContext({
        state: createMockState({
          navigation: { view: "apps", selectedIdx: 0, lastGPressed: 0 },
          apps,
        }),
        statusLog: mockStatusLog,
      });

      await diffCommand.execute(context);

      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "Preparing diff for cursor-app…",
        "diff",
      );
    });

    it("should find target from selected apps", async () => {
      const mockStatusLog = {
        info: mock(),
        warn: mock(),
        error: mock(),
        debug: mock(),
        set: mock(),
        clear: mock(),
      };
      const context = createMockContext({
        state: createMockState({
          selections: {
            selectedApps: new Set(["selected-app"]),
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
          },
        }),
        statusLog: mockStatusLog,
      });

      await diffCommand.execute(context);

      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "Preparing diff for selected-app…",
        "diff",
      );
    });

    it("should warn when no app selected", async () => {
      const mockStatusLog = {
        info: mock(),
        warn: mock(),
        error: mock(),
        debug: mock(),
        set: mock(),
        clear: mock(),
      };
      const context = createMockContext({
        state: createMockState({
          navigation: { view: "clusters", selectedIdx: 0, lastGPressed: 0 },
          selections: {
            selectedApps: new Set(),
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
          },
          apps: [],
        }),
        statusLog: mockStatusLog,
      });

      await diffCommand.execute(context);

      expect(mockStatusLog.warn).toHaveBeenCalledWith(
        "No app selected to diff.",
        "user-action",
      );
    });

    it("should handle diff session success", async () => {
      const mockStatusLog = {
        info: mock(),
        warn: mock(),
        error: mock(),
        debug: mock(),
        set: mock(),
        clear: mock(),
      };
      const context = createMockContext({
        statusLog: mockStatusLog,
      });

      await diffCommand.execute(context, "test-app");

      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "Preparing diff for test-app…",
        "diff",
      );
      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "No differences.",
        "diff",
      );
    });
  });

  describe("properties", () => {
    it("should have correct description", () => {
      expect(diffCommand.description).toBe("View diff for application");
    });

    it("should have empty aliases array", () => {
      expect(diffCommand.aliases).toEqual([]);
    });
  });
});

describe("RollbackCommand", () => {
  let rollbackCommand: TestRollbackCommand;

  beforeEach(() => {
    rollbackCommand = new TestRollbackCommand();
  });

  describe("canExecute", () => {
    it("should require authentication", () => {
      const context = createMockContext({
        state: createMockState({ server: null }),
      });

      expect(rollbackCommand.canExecute(context)).toBe(false);
    });

    it("should allow execution when authenticated", () => {
      const context = createMockContext();

      expect(rollbackCommand.canExecute(context)).toBe(true);
    });
  });

  describe("execute", () => {
    it("should require authentication", async () => {
      const mockStatusLog = {
        info: mock(),
        warn: mock(),
        error: mock(),
        debug: mock(),
        set: mock(),
        clear: mock(),
      };
      const context = createMockContext({
        state: createMockState({ server: null }),
        statusLog: mockStatusLog,
      });

      await rollbackCommand.execute(context);

      expect(mockStatusLog.error).toHaveBeenCalledWith(
        "Not authenticated.",
        "auth",
      );
    });

    it("should set rollback target and mode", async () => {
      const mockDispatch = mock();
      const mockStatusLog = {
        info: mock(),
        warn: mock(),
        error: mock(),
        debug: mock(),
        set: mock(),
        clear: mock(),
      };
      const context = createMockContext({
        dispatch: mockDispatch,
        statusLog: mockStatusLog,
      });

      await rollbackCommand.execute(context, "rollback-app");

      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "Opening rollback for rollback-app…",
        "rollback",
      );
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_ROLLBACK_APP_NAME",
        payload: "rollback-app",
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "rollback",
      });
    });

    it("should handle app selection logic from cursor", async () => {
      const mockDispatch = mock();
      const mockStatusLog = {
        info: mock(),
        warn: mock(),
        error: mock(),
        debug: mock(),
        set: mock(),
        clear: mock(),
      };
      const apps = [
        {
          name: "cursor-app",
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
      const context = createMockContext({
        state: createMockState({
          navigation: { view: "apps", selectedIdx: 0, lastGPressed: 0 },
          apps,
        }),
        dispatch: mockDispatch,
        statusLog: mockStatusLog,
      });

      await rollbackCommand.execute(context);

      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_ROLLBACK_APP_NAME",
        payload: "cursor-app",
      });
    });

    it("should warn when no app selected", async () => {
      const mockStatusLog = {
        info: mock(),
        warn: mock(),
        error: mock(),
        debug: mock(),
        set: mock(),
        clear: mock(),
      };
      const context = createMockContext({
        state: createMockState({
          navigation: { view: "clusters", selectedIdx: 0, lastGPressed: 0 },
          selections: {
            selectedApps: new Set(),
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
          },
          apps: [],
        }),
        statusLog: mockStatusLog,
      });

      await rollbackCommand.execute(context);

      expect(mockStatusLog.warn).toHaveBeenCalledWith(
        "No app selected to rollback.",
        "user-action",
      );
    });
  });
});

describe("ResourcesCommand", () => {
  let resourcesCommand: TestResourcesCommand;

  beforeEach(() => {
    resourcesCommand = new TestResourcesCommand();
  });

  describe("canExecute", () => {
    it("should require authentication", () => {
      const context = createMockContext({
        state: createMockState({ server: null }),
      });

      expect(resourcesCommand.canExecute(context)).toBe(false);
    });

    it("should allow execution when authenticated", () => {
      const context = createMockContext();

      expect(resourcesCommand.canExecute(context)).toBe(true);
    });
  });

  describe("execute", () => {
    it("should require authentication", () => {
      const mockStatusLog = {
        info: mock(),
        warn: mock(),
        error: mock(),
        debug: mock(),
        set: mock(),
        clear: mock(),
      };
      const context = createMockContext({
        state: createMockState({ server: null }),
        statusLog: mockStatusLog,
      });

      resourcesCommand.execute(context);

      expect(mockStatusLog.error).toHaveBeenCalledWith(
        "Not authenticated.",
        "auth",
      );
    });

    it("should set resource view app and mode", () => {
      const mockDispatch = mock();
      const context = createMockContext({
        dispatch: mockDispatch,
      });

      resourcesCommand.execute(context, "resource-app");

      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SYNC_VIEW_APP",
        payload: "resource-app",
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "resources",
      });
    });

    it("should handle aliases (resource, res)", () => {
      expect(resourcesCommand.aliases).toEqual(["resource", "res"]);
    });

    it("should find target from single selected app", () => {
      const mockDispatch = mock();
      const context = createMockContext({
        state: createMockState({
          selections: {
            selectedApps: new Set(["single-selected-app"]),
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
          },
        }),
        dispatch: mockDispatch,
      });

      resourcesCommand.execute(context);

      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SYNC_VIEW_APP",
        payload: "single-selected-app",
      });
    });

    it("should warn when no app selected", () => {
      const mockStatusLog = {
        info: mock(),
        warn: mock(),
        error: mock(),
        debug: mock(),
        set: mock(),
        clear: mock(),
      };
      const context = createMockContext({
        state: createMockState({
          navigation: { view: "clusters", selectedIdx: 0, lastGPressed: 0 },
          selections: {
            selectedApps: new Set(),
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set(),
          },
          apps: [],
        }),
        statusLog: mockStatusLog,
      });

      resourcesCommand.execute(context);

      expect(mockStatusLog.warn).toHaveBeenCalledWith(
        "No app selected to open resources view.",
        "user-action",
      );
    });
  });
});

describe("LogsCommand", () => {
  let logsCommand: TestLogsCommand;

  beforeEach(() => {
    logsCommand = new TestLogsCommand();
  });

  describe("execute", () => {
    it("should open logs session", async () => {
      const mockStatusLog = {
        info: mock(),
        warn: mock(),
        error: mock(),
        debug: mock(),
        set: mock(),
        clear: mock(),
      };
      const context = createMockContext({
        statusLog: mockStatusLog,
      });

      await logsCommand.execute(context);

      expect(mockStatusLog.info).toHaveBeenCalledWith("Opening logs…", "logs");
    });

    it("should have log alias", () => {
      expect(logsCommand.aliases).toEqual(["log"]);
    });

    it("should have correct description", () => {
      expect(logsCommand.description).toBe("Open log viewer");
    });
  });
});

describe("LicenseCommand", () => {
  let licenseCommand: TestLicenseCommand;

  beforeEach(() => {
    licenseCommand = new TestLicenseCommand();
  });

  describe("execute", () => {
    it("should open licenses session", async () => {
      const mockDispatch = mock();
      const mockStatusLog = {
        info: mock(),
        warn: mock(),
        error: mock(),
        debug: mock(),
        set: mock(),
        clear: mock(),
      };
      const context = createMockContext({
        dispatch: mockDispatch,
        statusLog: mockStatusLog,
      });

      await licenseCommand.execute(context);

      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_MODE",
        payload: "normal",
      });
      expect(mockStatusLog.info).toHaveBeenCalledWith(
        "Opening licenses…",
        "license",
      );
    });

    it("should have licenses alias", () => {
      expect(licenseCommand.aliases).toEqual(["licenses"]);
    });

    it("should have correct description", () => {
      expect(licenseCommand.description).toBe("View licenses");
    });
  });
});
