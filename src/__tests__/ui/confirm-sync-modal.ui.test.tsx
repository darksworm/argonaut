import { render } from "ink-testing-library";
import React from "react";
import { ConfirmSyncModal } from "../../components/modals/ConfirmSyncModal";
import { AppStateProvider } from "../../contexts/AppStateContext";
import type { AppItem } from "../../types/domain";
import type { Server } from "../../types/server";
import { stripAnsi } from "../test-utils";

describe("ConfirmSyncModal UI Tests", () => {
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
  ];

  const mockServer: Server = {
    config: {
      baseUrl: "https://argocd.test.com",
      insecure: false,
    },
    token: "mock-jwt-token",
  };

  const baseState = {
    mode: "confirm-sync" as const,
    server: mockServer,
    apps: mockApps,
    apiVersion: "v2.8.0",
    terminal: { rows: 30, cols: 120 },
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
      command: "",
      isVersionOutdated: false,
    },
    modals: {
      confirmTarget: "frontend-app",
      confirmSyncPrune: false,
      confirmSyncWatch: true,
      rollbackAppName: null,
      syncViewApp: null,
    },
    loadingAbortController: null,
  };

  describe("Modal Rendering and Visibility", () => {
    it("renders the sync confirmation modal when in confirm-sync mode", () => {
      const { lastFrame } = render(
        <AppStateProvider initialState={baseState}>
          <ConfirmSyncModal />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show the confirmation dialog
      expect(frame).toContain("Sync application?");
      expect(frame).toContain("Do you want to sync");
      expect(frame).toContain("frontend-app");
      expect(frame).toContain("(y/n)");
    });

    it("does not render when not in confirm-sync mode", () => {
      const normalModeState = {
        ...baseState,
        mode: "normal" as const,
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={normalModeState}>
          <ConfirmSyncModal />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should be empty (modal hidden)
      expect(frame).toBe("");
    });

    it("does not render when confirmTarget is null", () => {
      const noTargetState = {
        ...baseState,
        modals: {
          ...baseState.modals,
          confirmTarget: null,
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={noTargetState}>
          <ConfirmSyncModal />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should be empty (modal hidden)
      expect(frame).toBe("");
    });
  });

  describe("Single App vs Multi-App Behavior", () => {
    it("displays single app confirmation dialog", () => {
      const { lastFrame } = render(
        <AppStateProvider initialState={baseState}>
          <ConfirmSyncModal />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Single app mode
      expect(frame).toContain("Sync application?");
      expect(frame).toContain("frontend-app");
      expect(frame).not.toContain("Sync applications?");
    });

    it("displays multi-app confirmation dialog", () => {
      const multiAppState = {
        ...baseState,
        selections: {
          ...baseState.selections,
          selectedApps: new Set(["frontend-app", "backend-api"]),
        },
        modals: {
          ...baseState.modals,
          confirmTarget: "__MULTI__",
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={multiAppState}>
          <ConfirmSyncModal />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Multi-app mode
      expect(frame).toContain("Sync applications?");
      expect(frame).toContain("2"); // Number of selected apps
      expect(frame).not.toContain("Sync application?");
    });

    it("shows selected app count in multi-app mode", () => {
      const multiAppState = {
        ...baseState,
        selections: {
          ...baseState.selections,
          selectedApps: new Set(["frontend-app", "backend-api"]),
        },
        modals: {
          ...baseState.modals,
          confirmTarget: "__MULTI__",
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={multiAppState}>
          <ConfirmSyncModal />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show count of selected apps
      expect(frame).toContain("2");
    });
  });

  describe("Option Toggles (Prune and Watch)", () => {
    it("displays prune option with current state", () => {
      const { lastFrame } = render(
        <AppStateProvider initialState={baseState}>
          <ConfirmSyncModal />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show prune option
      expect(frame).toContain("Prune [p]");
      expect(frame).toContain("off"); // confirmSyncPrune is false
    });

    it("displays watch option with current state for single app", () => {
      const { lastFrame } = render(
        <AppStateProvider initialState={baseState}>
          <ConfirmSyncModal />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show watch option
      expect(frame).toContain("Watch [w]");
      expect(frame).toContain("on"); // confirmSyncWatch is true
    });

    it("shows enabled prune option", () => {
      const pruneEnabledState = {
        ...baseState,
        modals: {
          ...baseState.modals,
          confirmSyncPrune: true,
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={pruneEnabledState}>
          <ConfirmSyncModal />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show prune as enabled
      expect(frame).toContain("Prune [p]");
      expect(frame).toContain("on");
    });

    it("disables watch option in multi-app mode", () => {
      const multiAppState = {
        ...baseState,
        selections: {
          ...baseState.selections,
          selectedApps: new Set(["frontend-app", "backend-api"]),
        },
        modals: {
          ...baseState.modals,
          confirmTarget: "__MULTI__",
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={multiAppState}>
          <ConfirmSyncModal />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Watch should be disabled in multi-app mode
      expect(frame).toContain("Watch");
      expect(frame).toContain("disabled");
    });

    it("shows both prune and watch options with mixed states", () => {
      const mixedOptionsState = {
        ...baseState,
        modals: {
          ...baseState.modals,
          confirmSyncPrune: true,
          confirmSyncWatch: false,
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={mixedOptionsState}>
          <ConfirmSyncModal />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show both options with their states
      expect(frame).toContain("Prune [p]");
      expect(frame).toContain("Watch [w]");

      // Check for the specific mixed states - prune on, watch off
      const cleanFrame = stripAnsi(frame);
      expect(cleanFrame).toContain("Prune [p]: on • Watch [w]: off");
    });
  });

  describe("Modal Content and Styling", () => {
    it("displays proper title and message structure", () => {
      const { lastFrame } = render(
        <AppStateProvider initialState={baseState}>
          <ConfirmSyncModal />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should have proper structure
      expect(frame).toContain("Sync application?");
      expect(frame).toContain("Do you want to sync");
      expect(frame).toContain("frontend-app");
      expect(frame).toContain("? (y/n):");
    });

    it("shows different app names correctly", () => {
      const differentAppState = {
        ...baseState,
        modals: {
          ...baseState.modals,
          confirmTarget: "backend-api",
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={differentAppState}>
          <ConfirmSyncModal />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show the different app name
      expect(frame).toContain("backend-api");
      expect(frame).not.toContain("frontend-app");
    });

    it("renders with correct border and layout", () => {
      const { lastFrame } = render(
        <AppStateProvider initialState={baseState}>
          <ConfirmSyncModal />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should contain border characters (round border style)
      expect(frame).toMatch(/[╭╮╯╰─│]/);
    });
  });

  describe("Empty and Edge Cases", () => {
    it("handles empty selected apps in multi-mode", () => {
      const emptyMultiState = {
        ...baseState,
        selections: {
          ...baseState.selections,
          selectedApps: new Set<string>(),
        },
        modals: {
          ...baseState.modals,
          confirmTarget: "__MULTI__",
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={emptyMultiState}>
          <ConfirmSyncModal />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should show 0 apps selected
      expect(frame).toContain("0");
      expect(frame).toContain("Sync applications?");
    });

    it("handles missing server gracefully", () => {
      const noServerState = {
        ...baseState,
        server: null,
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={noServerState}>
          <ConfirmSyncModal />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should still render the modal (server check happens on confirm)
      expect(frame).toContain("Sync application?");
      expect(frame).toContain("frontend-app");
    });

    it("renders with narrow terminal width", () => {
      const narrowState = {
        ...baseState,
        terminal: { rows: 20, cols: 60 },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={narrowState}>
          <ConfirmSyncModal />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should still render without crashing
      expect(frame).toBeDefined();
      expect(frame.length).toBeGreaterThan(0);
      expect(frame).toContain("Sync application?");
    });

    it("renders with wide terminal width", () => {
      const wideState = {
        ...baseState,
        terminal: { rows: 40, cols: 200 },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={wideState}>
          <ConfirmSyncModal />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should render properly with wide terminal
      expect(frame).toBeDefined();
      expect(frame.length).toBeGreaterThan(0);
      expect(frame).toContain("Sync application?");
      expect(frame).toContain("frontend-app");
    });
  });

  describe("State Variations", () => {
    it("handles different modes correctly", () => {
      // Test with different modes to ensure modal only shows in confirm-sync
      const modes = [
        "normal",
        "search",
        "command",
        "resources",
        "rollback",
      ] as const;

      modes.forEach((mode) => {
        const modeState = {
          ...baseState,
          mode,
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={modeState}>
            <ConfirmSyncModal />
          </AppStateProvider>,
        );

        const frame = lastFrame();

        // Should be empty for all non-confirm-sync modes
        expect(frame).toBe("");
      });
    });

    it("handles app with special characters in name", () => {
      const specialNameState = {
        ...baseState,
        modals: {
          ...baseState.modals,
          confirmTarget: "my-app-with-dashes_and_underscores.123",
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={specialNameState}>
          <ConfirmSyncModal />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should display the special app name correctly
      expect(frame).toContain("my-app-with-dashes_and_underscores.123");
    });

    it("handles very long app names", () => {
      const longNameState = {
        ...baseState,
        modals: {
          ...baseState.modals,
          confirmTarget:
            "very-long-application-name-that-might-cause-layout-issues-in-narrow-terminals",
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={longNameState}>
          <ConfirmSyncModal />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should render without breaking
      expect(frame).toBeDefined();
      expect(frame).toContain("very-long-application-name");
    });
  });
});
