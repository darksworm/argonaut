import type { View } from "../types/domain";
import type { Command, CommandContext } from "./types";

export class NavigationCommand implements Command {
  constructor(
    private targetView: View,
    _commandName: string,
    public aliases: string[] = [],
  ) {}

  get description() {
    return `Switch to ${this.targetView} view`;
  }

  canExecute(context: CommandContext): boolean {
    return context.state.mode === "normal";
  }

  execute(context: CommandContext, arg?: string): void {
    const { dispatch } = context;

    dispatch({ type: "RESET_NAVIGATION", payload: { view: this.targetView } });
    dispatch({ type: "SET_MODE", payload: "normal" });

    // Handle view-specific argument for selection
    if (arg) {
      switch (this.targetView) {
        case "clusters":
          dispatch({ type: "SET_SCOPE_CLUSTERS", payload: new Set([arg]) });
          break;
        case "namespaces":
          dispatch({ type: "SET_SCOPE_NAMESPACES", payload: new Set([arg]) });
          break;
        case "projects":
          dispatch({ type: "SET_SCOPE_PROJECTS", payload: new Set([arg]) });
          break;
        case "apps":
          dispatch({ type: "SET_SELECTED_APPS", payload: new Set([arg]) });
          break;
      }
    } else {
      // Clear selection when returning to view without argument
      switch (this.targetView) {
        case "clusters":
          dispatch({ type: "SET_SCOPE_CLUSTERS", payload: new Set() });
          break;
        case "namespaces":
          dispatch({ type: "SET_SCOPE_NAMESPACES", payload: new Set() });
          break;
        case "projects":
          dispatch({ type: "SET_SCOPE_PROJECTS", payload: new Set() });
          break;
        case "apps":
          dispatch({ type: "SET_SELECTED_APPS", payload: new Set() });
          break;
      }
    }
  }
}

export class ClearCommand implements Command {
  aliases = [];
  description = "Clear current view selection";

  canExecute(context: CommandContext): boolean {
    return context.state.mode === "normal";
  }

  execute(context: CommandContext): void {
    const { state, dispatch, statusLog } = context;
    const { view } = state.navigation;

    switch (view) {
      case "clusters":
        dispatch({ type: "SET_SCOPE_CLUSTERS", payload: new Set() });
        break;
      case "namespaces":
        dispatch({ type: "SET_SCOPE_NAMESPACES", payload: new Set() });
        break;
      case "projects":
        dispatch({ type: "SET_SCOPE_PROJECTS", payload: new Set() });
        break;
      case "apps":
        dispatch({ type: "SET_SELECTED_APPS", payload: new Set() });
        break;
    }

    statusLog.info("Selection cleared.", "user-action");
  }
}

export class UpCommand implements Command {
  aliases = ["up"];
  description = "Go up one level in navigation hierarchy";

  canExecute(context: CommandContext): boolean {
    return context.state.mode === "normal" || context.state.mode === "command";
  }

  execute(context: CommandContext): void {
    const { state, dispatch } = context;
    const { navigation } = state;
    const { view } = navigation;

    dispatch({ type: "SET_SELECTED_IDX", payload: 0 });
    dispatch({ type: "CLEAR_FILTERS" });

    switch (view) {
      case "apps":
        dispatch({ type: "SET_SELECTED_APPS", payload: new Set() });
        dispatch({ type: "SET_SCOPE_PROJECTS", payload: new Set() });
        dispatch({ type: "SET_VIEW", payload: "projects" });
        break;
      case "projects":
        dispatch({ type: "SET_SCOPE_NAMESPACES", payload: new Set() });
        dispatch({ type: "SET_VIEW", payload: "namespaces" });
        break;
      case "namespaces":
        dispatch({ type: "SET_SCOPE_CLUSTERS", payload: new Set() });
        dispatch({ type: "SET_VIEW", payload: "clusters" });
        break;
      case "clusters":
        dispatch({ type: "SET_SCOPE_CLUSTERS", payload: new Set() });
        break;
    }
  }
}

export class ClearAllCommand implements Command {
  aliases = [];
  description = "Clear all selections and filters";

  execute(context: CommandContext): void {
    const { dispatch, statusLog } = context;

    dispatch({ type: "CLEAR_ALL_SELECTIONS" });
    dispatch({ type: "CLEAR_FILTERS" });

    statusLog.info("All filtering cleared.", "user-action");
  }
}
