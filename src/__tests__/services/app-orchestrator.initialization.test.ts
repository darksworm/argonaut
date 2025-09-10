import { describe, expect, it, mock } from "bun:test";
import type { AppAction } from "../../contexts/AppStateContext";

// Simple ok-like helper
const okResult = <T>(value: T) => ({
  isOk: () => true,
  isErr: () => false,
  value,
});

// Mock dependencies before importing orchestrator
const mockCliConfig = {
  "current-context": "test",
  contexts: [{ name: "test", server: "https://test-server.com", user: "test" }],
  servers: [{ server: "https://test-server.com" }],
  users: [{ name: "test", "auth-token": "test-token" }],
};

mock.module("../../config/cli-config", () => ({
  readCLIConfig: () => Promise.resolve(okResult(mockCliConfig)),
  getCurrentServerConfigObj: () =>
    okResult({ baseUrl: "https://test-server.com", insecure: false }),
  tokenFromConfig: () => okResult("test-token"),
}));

mock.module("../../api/version", () => ({
  getApiVersion: () => Promise.resolve("v1"),
}));

mock.module("../../api/session", () => ({
  getUserInfo: () => Promise.resolve(okResult({})),
}));

mock.module("../../api/applications.query", () => ({
  listApps: () => Promise.resolve(okResult([{}])),
}));

mock.module("../../services/app-mapper", () => ({
  appToItem: () => ({
    name: "app1",
    sync: "Synced",
    health: "Healthy",
    clusterId: "in-cluster",
    clusterLabel: "in-cluster",
    namespace: "default",
    appNamespace: "argocd",
    project: "default",
    lastSyncAt: "",
  }),
}));

mock.module("../../utils/version-check", () => ({
  checkVersion: () => Promise.resolve(okResult({ isOutdated: false })),
}));

mock.module("../../services/api-errors", () => ({
  getDisplayMessage: (e: any) => String(e?.message ?? e),
  requiresUserAction: () => false,
}));

const { DefaultAppOrchestrator } = await import(
  "../../services/app-orchestrator"
);
const { createMockStatusLog } = await import("../test-utils");

describe("initializeApp", () => {
  it("loads apps before leaving loading mode", async () => {
    const orchestrator = new DefaultAppOrchestrator();
    const dispatch = mock<(action: AppAction) => void>();
    const statusLog = createMockStatusLog();

    await orchestrator.initializeApp(dispatch, statusLog);

    const calls = dispatch.mock.calls.map((c) => c[0]);
    const appsIndex = calls.findIndex((a) => a.type === "SET_APPS");
    const modeIndex = calls.findIndex(
      (a) => a.type === "SET_MODE" && a.payload === "normal",
    );

    expect(appsIndex).toBeGreaterThan(-1);
    expect(modeIndex).toBeGreaterThan(appsIndex);
  });
});
