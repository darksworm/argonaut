import { getUserInfo } from "../api/session";
import { getApiVersion } from "../api/version";
import type { StatusLogger } from "../commands/types";
import {
  getCurrentServerConfigObj,
  readCLIConfig,
  tokenFromConfig,
} from "../config/cli-config";
import type { AppAction } from "../contexts/AppStateContext";
import { httpClientManager } from "../services/http-client";
import { log, setCurrentView } from "../services/logger";
import type { Server } from "../types/server";
import { checkVersion } from "../utils/version-check";

export interface AppOrchestrator {
  initializeApp(
    dispatch: React.Dispatch<AppAction>,
    statusLog: StatusLogger,
    signal?: AbortSignal,
  ): Promise<void>;
  cleanupAndExit(exit: () => void): void;
  handleAuthError(
    dispatch: React.Dispatch<AppAction>,
    statusLog: StatusLogger,
  ): void;
  updateView(mode: string, view: string): void;
  handleTerminalResize(dispatch: React.Dispatch<AppAction>): () => void;
}

export class DefaultAppOrchestrator implements AppOrchestrator {
  private loadingAbortController: AbortController | null = null;

  async initializeApp(
    dispatch: React.Dispatch<AppAction>,
    statusLog: StatusLogger,
    signal?: AbortSignal,
  ): Promise<void> {
    // Clean up any previous abort controller
    if (this.loadingAbortController) {
      this.loadingAbortController.abort();
    }

    const abortController = new AbortController();
    this.loadingAbortController = abortController;
    const effectiveSignal = signal || abortController.signal;

    dispatch({ type: "SET_MODE", payload: "loading" });
    dispatch({
      type: "SET_LOADING_ABORT_CONTROLLER",
      payload: abortController,
    });

    statusLog.info("Loading ArgoCD config…", "boot");
    log.debug("Begin loading Argo CD cli config");

    try {
      const cfg = await readCLIConfig();
      if (effectiveSignal.aborted) return;

      if (cfg.isErr()) {
        this.loadingAbortController = null;
        dispatch({ type: "SET_SERVER", payload: null });
        statusLog.error(
          `Could not load Argo CD config. Please run 'argocd login' to configure and authenticate. ${cfg.error.message}`,
          "auth",
        );
        dispatch({ type: "SET_MODE", payload: "auth-required" });
        return;
      }

      const serverConfig = getCurrentServerConfigObj(cfg.value);
      if (effectiveSignal.aborted) return;

      if (serverConfig.isErr()) {
        this.loadingAbortController = null;
        dispatch({ type: "SET_SERVER", payload: null });
        statusLog.error(
          `Could not load Argo CD config. Please run 'argocd login' to configure and authenticate. ${serverConfig.error.message}`,
          "auth",
        );
        dispatch({ type: "SET_MODE", payload: "auth-required" });
        return;
      }

      const tokenResult = tokenFromConfig(cfg.value);
      if (effectiveSignal.aborted) return;

      if (tokenResult.isErr()) {
        this.loadingAbortController = null;
        dispatch({ type: "SET_SERVER", payload: null });
        statusLog.error(tokenResult.error.message, "auth");
        dispatch({ type: "SET_MODE", payload: "auth-required" });
        return;
      }

      const serverObj: Server = {
        config: serverConfig.value,
        token: tokenResult.value,
      };

      dispatch({ type: "SET_SERVER", payload: serverObj });

      // Get API version
      log.debug("Fetching API version");
      try {
        const version = await getApiVersion(serverObj, {
          signal: effectiveSignal,
          timeout: 1000,
        });
        dispatch({ type: "SET_API_VERSION", payload: version });
      } catch (error: any) {
        if (effectiveSignal.aborted) return;
        this.loadingAbortController = null;
        dispatch({ type: "SET_SERVER", payload: null });
        const message = error?.message?.includes("timeout")
          ? "Could not connect to Argo CD! Connection timed out."
          : "Could not connect to Argo CD! Is it dead?";
        statusLog.error(message, "connection");
        dispatch({ type: "SET_MODE", payload: "error" });
        return;
      }
      log.debug("Fetched API version");

      // Verify user info
      const userInfoResult = await getUserInfo(serverObj, {
        signal: effectiveSignal,
        timeout: 1000,
      });

      if (effectiveSignal.aborted) {
        log.debug("Signal aborted after getUserInfo, exiting initialization");
        return;
      }

      if (userInfoResult.isErr()) {
        // Check if this was an abort error - if so, exit silently
        if (userInfoResult.error.message.includes("Request aborted")) {
          return;
        }
        this.loadingAbortController = null;
        dispatch({ type: "SET_SERVER", payload: null });
        statusLog.error(
          `user info fetch fail: ${userInfoResult.error.message}`,
          "auth",
        );
        dispatch({ type: "SET_MODE", payload: "auth-required" });
        return;
      }
      log.debug("Fetched user info");

      // Show ready status
      if (serverConfig.value.insecure) {
        statusLog.warn(
          "Ready • WARNING: Insecure TLS connection (certificate validation disabled)",
          "connection",
        );
      } else {
        statusLog.info("Ready", "connection");
      }

      // Check version in background
      this.checkVersionInBackground(dispatch, statusLog, effectiveSignal);

      this.loadingAbortController = null;
      dispatch({ type: "SET_MODE", payload: "normal" });
      statusLog.debug("Boot finish");
    } catch (error) {
      if (effectiveSignal.aborted) return;
      this.loadingAbortController = null;
      dispatch({ type: "SET_SERVER", payload: null });
      statusLog.error(
        `Initialization failed: ${error instanceof Error ? error.message : String(error)}`,
        "boot",
      );
      dispatch({ type: "SET_MODE", payload: "error" });
    }
  }

  private async checkVersionInBackground(
    dispatch: React.Dispatch<AppAction>,
    statusLog: StatusLogger,
    signal: AbortSignal,
  ): Promise<void> {
    try {
      // Import here to avoid circular dependencies
      const packageJson = await import("../../package.json");
      const versionResult = await checkVersion(packageJson.version);
      if (signal.aborted) return;

      if (versionResult.isOk()) {
        const result = versionResult.value;
        dispatch({ type: "SET_VERSION_OUTDATED", payload: result.isOutdated });
        if (result.latestVersion) {
          dispatch({
            type: "SET_LATEST_VERSION",
            payload: result.latestVersion,
          });
        }
      } else {
        if (signal.aborted) return;
        statusLog.warn(
          `Ready • Could not check for updates ${versionResult.error.message}`,
          "version-check",
        );
      }
    } catch (error) {
      // Version check failures are non-critical
      if (!signal.aborted) {
        log.debug(`Version check failed: ${error}`, "version-check");
      }
    }
  }

  cleanupAndExit(exit: () => void): void {
    log.debug("Cleaning up resources before exit", "cleanup");

    // Cancel any ongoing loading operations
    if (this.loadingAbortController) {
      this.loadingAbortController.abort();
    }

    // Clean up HTTP clients to prevent hanging
    httpClientManager.clearClients();

    log.debug("Calling exit()", "cleanup");
    exit();

    // Force exit after a short delay if the app doesn't exit gracefully
    setTimeout(() => {
      log.debug("Force exiting due to hanging handles", "cleanup");
      process.exit(0);
    }, 100);
  }

  handleAuthError(
    dispatch: React.Dispatch<AppAction>,
    statusLog: StatusLogger,
  ): void {
    dispatch({ type: "SET_SERVER", payload: null });
    statusLog.error(
      "please use argocd login to authenticate before running argonaut",
      "auth",
    );
    dispatch({ type: "SET_MODE", payload: "auth-required" });
  }

  // Handle view changes for logging
  updateView(mode: string, view: string): void {
    setCurrentView(`${mode}:${view}`);
  }

  // Handle terminal resize
  handleTerminalResize(dispatch: React.Dispatch<AppAction>): () => void {
    const onResize = () => {
      dispatch({
        type: "SET_TERMINAL_SIZE",
        payload: {
          rows: process.stdout.rows || 24,
          cols: process.stdout.columns || 80,
        },
      });
    };

    process.stdout.on("resize", onResize);

    return () => {
      process.stdout.off("resize", onResize);
    };
  }
}
