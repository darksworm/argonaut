import { runAppDiffSession } from "../components/DiffView";
import { runLicenseSession } from "../components/LicenseView";
import { runLogViewerSession } from "../services/log-viewer";
import type { Command, CommandContext } from "./types";

export class SyncCommand implements Command {
  aliases = [];
  description = "Sync application(s)";

  canExecute(context: CommandContext): boolean {
    return context.state.server !== null;
  }

  execute(context: CommandContext, arg?: string): void {
    const { state, dispatch } = context;
    const { selectedApps } = state.selections;
    const { view, selectedIdx } = state.navigation;

    // Prefer explicit arg; otherwise if multiple apps are selected, confirm multi-sync.
    if (arg) {
      dispatch({ type: "SET_CONFIRM_TARGET", payload: arg });
      dispatch({ type: "SET_CONFIRM_SYNC_PRUNE", payload: false });
      dispatch({ type: "SET_CONFIRM_SYNC_WATCH", payload: true });
      dispatch({ type: "SET_MODE", payload: "confirm-sync" });
      return;
    }

    if (selectedApps.size > 1) {
      dispatch({ type: "SET_CONFIRM_TARGET", payload: "__MULTI__" });
      dispatch({ type: "SET_CONFIRM_SYNC_PRUNE", payload: false });
      dispatch({ type: "SET_CONFIRM_SYNC_WATCH", payload: false }); // disabled for multi
      dispatch({ type: "SET_MODE", payload: "confirm-sync" });
      return;
    }

    // Fallback to current cursor app (apps view) or the single selected app
    const visibleItems = this.getVisibleItems(context);
    const target =
      (view === "apps"
        ? (visibleItems[selectedIdx] as any)?.name
        : undefined) ||
      (selectedApps.size === 1 ? Array.from(selectedApps)[0] : undefined);

    if (!target) {
      context.statusLog.warn("No app selected to sync.", "user-action");
      return;
    }

    dispatch({ type: "SET_CONFIRM_TARGET", payload: target });
    dispatch({ type: "SET_CONFIRM_SYNC_PRUNE", payload: false });
    dispatch({ type: "SET_CONFIRM_SYNC_WATCH", payload: true });
    dispatch({ type: "SET_MODE", payload: "confirm-sync" });
  }

  private getVisibleItems(context: CommandContext): any[] {
    // This is a simplified version - in real implementation,
    // we'd need access to the full visible items calculation
    // For now, return apps as fallback
    return context.state.apps;
  }
}

export class DiffCommand implements Command {
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

    const visibleItems = this.getVisibleItems(context);
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

      await runAppDiffSession(server, target, {
        title: `${target} - Live vs Desired`,
      });

      // Small delay and trigger re-render with status message
      await new Promise((resolve) => setTimeout(resolve, 50));
      dispatch({ type: "SET_MODE", payload: "normal" });
      statusLog.info("No differences.", "diff");
    } catch (e: any) {
      try {
        const stdinAny = process.stdin as any;
        stdinAny.setRawMode?.(true);
        stdinAny.resume?.();
      } catch {}
      dispatch({ type: "SET_MODE", payload: "normal" });
      statusLog.error(`Diff failed: ${e?.message || String(e)}`, "diff");
    }
  }

  private getVisibleItems(context: CommandContext): any[] {
    return context.state.apps;
  }
}

export class RollbackCommand implements Command {
  aliases = [];
  description = "Rollback application";

  canExecute(context: CommandContext): boolean {
    return context.state.server !== null;
  }

  async execute(context: CommandContext, arg?: string): Promise<void> {
    const { state, dispatch, statusLog } = context;
    const { selectedApps } = state.selections;
    const { view, selectedIdx } = state.navigation;

    if (!context.state.server) {
      statusLog.error("Not authenticated.", "auth");
      return;
    }

    const visibleItems = this.getVisibleItems(context);
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

  private getVisibleItems(context: CommandContext): any[] {
    return context.state.apps;
  }
}

export class ResourcesCommand implements Command {
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

    const visibleItems = this.getVisibleItems(context);
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

  private getVisibleItems(context: CommandContext): any[] {
    return context.state.apps;
  }
}

export class LogsCommand implements Command {
  aliases = ["log"];
  description = "Open log viewer";

  async execute(context: CommandContext): Promise<void> {
    const { statusLog } = context;

    statusLog.info("Opening logs…", "logs");

    await runLogViewerSession({
      title: "Session Logs",
    });
  }
}

export class LicenseCommand implements Command {
  aliases = ["licenses"];
  description = "View licenses";

  async execute(context: CommandContext): Promise<void> {
    const { dispatch, statusLog } = context;

    try {
      dispatch({ type: "SET_MODE", payload: "normal" });
      statusLog.info("Opening licenses…", "license");

      await runLicenseSession({
        title: "Licenses",
      });
    } finally {
      // Ensure we always try to restore the UI
      try {
        const stdinAny = process.stdin as any;
        stdinAny.setRawMode?.(true);
        stdinAny.resume?.();
      } catch {}
      process.emit("external-exit");
    }
  }
}
