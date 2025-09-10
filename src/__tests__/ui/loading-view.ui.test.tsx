import { render } from "ink-testing-library";
import { LoadingView } from "../../components/views/LoadingView";
import { AppStateProvider } from "../../contexts/AppStateContext";
import { stripAnsi } from "../test-utils";

describe("LoadingView UI Tests", () => {
  const baseState = {
    mode: "loading" as const,
    server: {
      config: { baseUrl: "https://argocd.production.com", insecure: false },
      token: "mock-token",
    },
    apps: [],
    apiVersion: "v2.8.0",
    loadingMessage: "Connecting & fetching applications…",
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
      confirmTarget: null,
      confirmSyncPrune: false,
      confirmSyncWatch: true,
      rollbackAppName: null,
      syncViewApp: null,
    },
    loadingAbortController: null,
  };

  describe("Loading View Rendering", () => {
    it("renders the loading view when in loading mode", () => {
      const { lastFrame } = render(
        <AppStateProvider initialState={baseState}>
          <LoadingView />
        </AppStateProvider>,
      );

      const frame = lastFrame();
      const cleanFrame = stripAnsi(frame);

      // Should show loading content
      expect(frame).toBeDefined();
      expect(frame.length).toBeGreaterThan(0);

      // Should show LOADING text
      expect(cleanFrame).toContain("LOADING");

      // Should show View label
      expect(cleanFrame).toContain("View:");

      // Should show Context label
      expect(cleanFrame).toContain("Context:");
    });

    it("shows server context when server is available", () => {
      const { lastFrame } = render(
        <AppStateProvider initialState={baseState}>
          <LoadingView />
        </AppStateProvider>,
      );

      const frame = lastFrame();
      const cleanFrame = stripAnsi(frame);

      // Should show server hostname
      expect(cleanFrame).toContain("argocd.production.com");
    });

    it("shows placeholder when no server", () => {
      const noServerState = {
        ...baseState,
        server: null,
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={noServerState}>
          <LoadingView />
        </AppStateProvider>,
      );

      const frame = lastFrame();
      const cleanFrame = stripAnsi(frame);

      // Should show placeholder
      expect(cleanFrame).toContain("—");
      expect(cleanFrame).toContain("LOADING");
    });

    it("does not render when not in loading mode", () => {
      const normalModeState = {
        ...baseState,
        mode: "normal" as const,
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={normalModeState}>
          <LoadingView />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should be empty (view hidden)
      expect(frame).toBe("");
    });
  });

  describe("Loading Spinner and Content", () => {
    it("displays loading spinner character", () => {
      const { lastFrame } = render(
        <AppStateProvider initialState={baseState}>
          <LoadingView />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should contain spinner character (⠋ or similar)
      expect(frame).toMatch(/[⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏]/);
    });

    it("shows provided message from state", () => {
      const customState = { ...baseState, loadingMessage: "Preparing diff" };
      const { lastFrame } = render(
        <AppStateProvider initialState={customState}>
          <LoadingView />
        </AppStateProvider>,
      );

      const frame = lastFrame();
      const cleanFrame = stripAnsi(frame);

      // Should show custom message
      expect(cleanFrame).toContain("Preparing diff");
    });

    it("displays proper border styling", () => {
      const { lastFrame } = render(
        <AppStateProvider initialState={baseState}>
          <LoadingView />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should contain border characters (round border style)
      expect(frame).toMatch(/[╭╮╯╰─│]/);
    });
  });

  describe("Terminal Responsiveness", () => {
    it("adapts to narrow terminal", () => {
      const narrowState = {
        ...baseState,
        terminal: { rows: 20, cols: 60 },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={narrowState}>
          <LoadingView />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should render without crashing
      expect(frame).toBeDefined();
      expect(frame.length).toBeGreaterThan(0);

      // Should still show essential content
      const cleanFrame = stripAnsi(frame);
      expect(cleanFrame).toContain("LOADING");
    });

    it("adapts to wide terminal", () => {
      const wideState = {
        ...baseState,
        terminal: { rows: 40, cols: 200 },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={wideState}>
          <LoadingView />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should render properly with wide terminal
      expect(frame).toBeDefined();
      expect(frame.length).toBeGreaterThan(0);

      const cleanFrame = stripAnsi(frame);
      expect(cleanFrame).toContain("LOADING");
      expect(cleanFrame).toContain("argocd.production.com");
    });

    it("handles very small terminal gracefully", () => {
      const tinyState = {
        ...baseState,
        terminal: { rows: 8, cols: 40 },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={tinyState}>
          <LoadingView />
        </AppStateProvider>,
      );

      const frame = lastFrame();

      // Should still render something
      expect(frame).toBeDefined();
      expect(frame.length).toBeGreaterThan(0);
    });
  });

  describe("Server URL Variations", () => {
    it("displays different server URLs correctly", () => {
      const servers = [
        "https://argocd.staging.com",
        "https://argo.local:8080",
        "http://localhost:3000",
        "https://argocd-server.kube-system.svc.cluster.local",
      ];

      servers.forEach((url) => {
        const serverState = {
          ...baseState,
          server: {
            config: { baseUrl: url, insecure: false },
            token: "mock-token",
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={serverState}>
            <LoadingView />
          </AppStateProvider>,
        );

        const frame = lastFrame();
        const cleanFrame = stripAnsi(frame);

        // Should show the hostname from the URL
        const expectedHost = new URL(url).hostname;
        expect(cleanFrame).toContain(expectedHost);
      });
    });

    it("handles insecure server configuration", () => {
      const insecureState = {
        ...baseState,
        server: {
          config: { baseUrl: "http://argocd.insecure.com", insecure: true },
          token: "mock-token",
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={insecureState}>
          <LoadingView />
        </AppStateProvider>,
      );

      const frame = lastFrame();
      const cleanFrame = stripAnsi(frame);

      // Should show server info
      expect(cleanFrame).toContain("argocd.insecure.com");
      expect(cleanFrame).toContain("LOADING");
    });
  });

  describe("Mode Transitions", () => {
    it("does not render in other modes", () => {
      const modes = [
        "normal",
        "search",
        "command",
        "help",
        "confirm-sync",
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
            <LoadingView />
          </AppStateProvider>,
        );

        const frame = lastFrame();

        // Should be empty for all non-loading modes
        expect(frame).toBe("");
      });
    });
  });
});
