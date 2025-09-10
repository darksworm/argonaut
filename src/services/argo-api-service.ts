import { ok, type Result } from "neverthrow";
import { syncApp } from "../api/applications.command";
import {
  getManagedResourceDiffs,
  listApps,
  type ResourceDiff,
  watchApps,
} from "../api/applications.query";
import type { ArgoApplication } from "../types/argo";
import type { AppItem } from "../types/domain";
import type { Server } from "../types/server";
import {
  type ApiError,
  getDisplayMessage,
  requiresUserAction,
} from "./api-errors";
import { appToItem } from "./app-mapper";

/**
 * Events emitted by the ArgoApiService
 */
export type ArgoApiEvent =
  | { type: "apps-loaded"; apps: AppItem[] }
  | { type: "app-updated"; app: AppItem }
  | { type: "app-deleted"; appName: string }
  | { type: "auth-error"; error: Error }
  | { type: "api-error"; message: string }
  | { type: "status-change"; status: string };

/**
 * Callback for handling events from the ArgoApiService
 */
export type ArgoApiEventHandler = (event: ArgoApiEvent) => void;

/**
 * Pure API service for Argo CD operations
 * Separated from React hooks for better testability and Go migration
 */
export class ArgoApiService {
  private watchController: AbortController | null = null;

  /**
   * List all applications from Argo CD
   */
  async listApplications(
    server: Server,
    signal?: AbortSignal,
  ): Promise<Result<AppItem[], ApiError>> {
    const appsResult = await listApps(server, signal);

    if (appsResult.isErr()) {
      return appsResult;
    }

    // Transform ArgoApplication[] to AppItem[]
    const items = appsResult.value.map(appToItem);
    return ok(items);
  }

  /**
   * Start watching applications for changes
   * Returns a cleanup function to stop watching
   */
  async watchApplications(
    server: Server,
    eventHandler: ArgoApiEventHandler,
    signal?: AbortSignal,
  ): Promise<() => void> {
    // Create our own controller that we can cancel
    this.watchController = new AbortController();
    const effectiveSignal = signal || this.watchController.signal;

    // Load initial applications
    eventHandler({ type: "status-change", status: "Loading…" });

    try {
      const initialResult = await this.listApplications(
        server,
        effectiveSignal,
      );

      if (effectiveSignal.aborted) return () => {};

      if (initialResult.isErr()) {
        const apiError = initialResult.error;

        // Check if requires user action (re-authentication)
        if (requiresUserAction(apiError)) {
          eventHandler({
            type: "auth-error",
            error: new Error(getDisplayMessage(apiError)),
          });
          eventHandler({ type: "status-change", status: "Auth required" });
          return () => {};
        }

        eventHandler({
          type: "api-error",
          message: getDisplayMessage(apiError),
        });
        eventHandler({
          type: "status-change",
          status: `Error: ${getDisplayMessage(apiError)}`,
        });
        return () => {};
      }

      // Emit initial apps
      eventHandler({ type: "apps-loaded", apps: initialResult.value });
      eventHandler({ type: "status-change", status: "Live" });

      // Start watching for changes
      this.startWatching(server, eventHandler, effectiveSignal);
    } catch (e: any) {
      if (effectiveSignal.aborted) return () => {};

      // Handle unexpected errors
      const msg = e?.message || String(e);
      if (/\b(401|403)\b/i.test(msg) || /unauthorized/i.test(msg)) {
        eventHandler({
          type: "auth-error",
          error: e instanceof Error ? e : new Error(msg),
        });
        eventHandler({ type: "status-change", status: "Auth required" });
        return () => {};
      }

      eventHandler({ type: "api-error", message: msg });
      eventHandler({ type: "status-change", status: `Error: ${msg}` });
    }

    return () => {
      if (this.watchController) {
        this.watchController.abort();
        this.watchController = null;
      }
    };
  }

  /**
   * Sync an application
   */
  async syncApplication(
    server: Server,
    appName: string,
    opts?: { prune?: boolean; appNamespace?: string },
  ): Promise<Result<void, ApiError>> {
    try {
      await syncApp(server, appName, opts);
      return { isOk: () => true, value: undefined } as Result<void, ApiError>;
    } catch (error: any) {
      return {
        isOk: () => false,
        error: { message: error.message || String(error) },
      } as Result<void, ApiError>;
    }
  }

  /**
   * Get managed resource diffs for an application
   */
  async getResourceDiffs(
    server: Server,
    appName: string,
    signal?: AbortSignal,
  ): Promise<Result<ResourceDiff[], ApiError>> {
    return getManagedResourceDiffs(server, appName, signal);
  }

  /**
   * Stop all watchers and clean up
   */
  cleanup(): void {
    if (this.watchController) {
      this.watchController.abort();
      this.watchController = null;
    }
  }

  /**
   * Private method to handle the watch stream
   */
  private async startWatching(
    server: Server,
    eventHandler: ArgoApiEventHandler,
    signal: AbortSignal,
  ): Promise<void> {
    try {
      for await (const ev of watchApps(server, undefined, signal)) {
        if (signal.aborted) return;

        const { type, application } = ev || ({} as any);
        if (!application?.metadata?.name) continue;

        const appName = application.metadata.name;

        if (type === "DELETED") {
          eventHandler({ type: "app-deleted", appName });
        } else {
          const item = appToItem(application as ArgoApplication);
          eventHandler({ type: "app-updated", app: item });
        }
      }
    } catch (e: any) {
      if (signal.aborted) return;

      // Handle watch stream errors
      const msg = e?.message || String(e);
      if (/\b(401|403)\b/i.test(msg) || /unauthorized/i.test(msg)) {
        eventHandler({
          type: "auth-error",
          error: e instanceof Error ? e : new Error(msg),
        });
        eventHandler({ type: "status-change", status: "Auth required" });
        return;
      }

      eventHandler({ type: "api-error", message: msg });
      eventHandler({ type: "status-change", status: `Error: ${msg}` });
    }
  }
}
