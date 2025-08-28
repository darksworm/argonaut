import { UpCommand } from "../navigation";
import type { CommandContext, InputHandler } from "../types";

export class NavigationInputHandler implements InputHandler {
  priority = 10; // High priority for navigation

  canHandle(context: CommandContext): boolean {
    return context.state.mode === "normal";
  }

  handleInput(input: string, key: any, context: CommandContext): boolean {
    const { state, dispatch } = context;
    const { navigation } = state;

    // Get visible items count (simplified - would need proper calculation)
    const visibleItemsLength = this.getVisibleItemsCount(context);

    // Basic navigation
    if (input === "j" || key.downArrow) {
      dispatch({
        type: "SET_SELECTED_IDX",
        payload: Math.min(
          navigation.selectedIdx + 1,
          Math.max(0, visibleItemsLength - 1),
        ),
      });
      return true;
    }

    if (input === "k" || key.upArrow) {
      dispatch({
        type: "SET_SELECTED_IDX",
        payload: Math.max(navigation.selectedIdx - 1, 0),
      });
      return true;
    }

    // Vim-style navigation: gg to go to top, G to go to bottom
    if (input === "g") {
      const now = Date.now();
      if (now - navigation.lastGPressed < 500) {
        // 500ms window for double g
        dispatch({ type: "SET_SELECTED_IDX", payload: 0 }); // Go to top
      }
      dispatch({ type: "SET_LAST_G_PRESSED", payload: now });
      return true;
    }

    if (input === "G") {
      dispatch({
        type: "SET_SELECTED_IDX",
        payload: Math.max(0, visibleItemsLength - 1),
      }); // Go to bottom
      return true;
    }

    // Enter to drill down
    if (key.return) {
      if (context.navigationActions?.drillDown) {
        context.navigationActions.drillDown();
      } else {
        this.drillDown(context);
      }
      return true;
    }

    // Space to toggle selection - only in apps view
    if (input === " ") {
      // Only allow space in apps view
      if (navigation.view !== "apps") {
        return false;
      }

      if (context.navigationActions?.toggleSelection) {
        context.navigationActions.toggleSelection();
      } else {
        this.toggleSelection(context);
      }
      return true;
    }

    if (key.escape) {
      const now = Date.now();
      // Debounce escape key to prevent rapid firing
      if (now - navigation.lastEscPressed < 200) {
        return true;
      }
      dispatch({ type: "SET_LAST_ESC_PRESSED", payload: now });

      if (context.state.selections.selectedApps.size > 1) {
        dispatch({ type: "SET_SELECTED_APPS", payload: new Set() });
      } else {
        new UpCommand().execute(context);
      }
      return true;
    }

    return false;
  }

  private getVisibleItemsCount(context: CommandContext): number {
    // Simplified - in real implementation, this would calculate visible items
    // based on current view, filters, etc.
    const { state } = context;
    const { navigation } = state;

    if (navigation.view === "apps") {
      return state.apps.length;
    }
    // For other views, return a reasonable default
    return 10;
  }

  private drillDown(context: CommandContext): void {
    const { state, dispatch } = context;
    const { navigation } = state;
    const { view } = navigation;

    // This is simplified - full implementation would need visible items calculation
    dispatch({ type: "SET_SELECTED_IDX", payload: 0 });
    dispatch({ type: "CLEAR_FILTERS" });
    dispatch({ type: "CLEAR_LOWER_LEVEL_SELECTIONS", payload: view });

    switch (view) {
      case "clusters":
        dispatch({ type: "SET_VIEW", payload: "namespaces" });
        break;
      case "namespaces":
        dispatch({ type: "SET_VIEW", payload: "projects" });
        break;
      case "projects":
        dispatch({ type: "SET_VIEW", payload: "apps" });
        break;
    }
  }

  private toggleSelection(context: CommandContext): void {
    const { state, dispatch } = context;
    const { navigation } = state;
    const { view } = navigation;

    // Only allow toggle selection in apps view
    if (view !== "apps") {
      return;
    }

    // Simplified - would need actual visible items and selected item
    dispatch({ type: "CLEAR_LOWER_LEVEL_SELECTIONS", payload: view });

    // This is a simplified implementation
    // Real implementation would get the actual selected item and toggle it
  }
}

export class ModeInputHandler implements InputHandler {
  priority = 20; // Higher priority for mode switches

  canHandle(context: CommandContext): boolean {
    return context.state.mode === "normal";
  }

  handleInput(input: string, _key: any, context: CommandContext): boolean {
    const { dispatch } = context;

    if (input === "?") {
      dispatch({ type: "SET_MODE", payload: "help" });
      return true;
    }

    if (input === "/") {
      dispatch({ type: "SET_MODE", payload: "search" });
      return true;
    }

    if (input === ":") {
      dispatch({ type: "SET_MODE", payload: "command" });
      dispatch({ type: "SET_COMMAND", payload: ":" });
      return true;
    }

    return false;
  }
}

export class SearchInputHandler implements InputHandler {
  priority = 30; // Highest priority when in search mode

  canHandle(context: CommandContext): boolean {
    return context.state.mode === "search";
  }

  handleInput(_input: string, key: any, context: CommandContext): boolean {
    const { dispatch } = context;

    if (key.escape) {
      dispatch({ type: "SET_MODE", payload: "normal" });
      dispatch({ type: "SET_SEARCH_QUERY", payload: "" });
      return true;
    }

    // Allow navigating the filtered list while typing
    const visibleItemsLength = this.getVisibleItemsCount(context);

    if (key.downArrow) {
      dispatch({
        type: "SET_SELECTED_IDX",
        payload: Math.min(
          context.state.navigation.selectedIdx + 1,
          Math.max(0, visibleItemsLength - 1),
        ),
      });
      return true;
    }

    if (key.upArrow) {
      dispatch({
        type: "SET_SELECTED_IDX",
        payload: Math.max(context.state.navigation.selectedIdx - 1, 0),
      });
      return true;
    }

    // Enter is handled by TextInput onSubmit; other typing goes to TextInput
    return false;
  }

  private getVisibleItemsCount(context: CommandContext): number {
    // Simplified implementation
    return context.state.apps.length;
  }
}

export class CommandInputHandler implements InputHandler {
  priority = 30; // Highest priority when in command mode

  canHandle(context: CommandContext): boolean {
    return context.state.mode === "command";
  }

  handleInput(_input: string, key: any, context: CommandContext): boolean {
    const { dispatch } = context;

    if (key.escape) {
      dispatch({ type: "SET_MODE", payload: "normal" });
      dispatch({ type: "SET_COMMAND", payload: ":" });
      return true;
    }

    // TextInput handles typing/enter
    return false;
  }
}

export class GlobalInputHandler implements InputHandler {
  priority = 0; // Lowest priority - catches global inputs

  canHandle(_context: CommandContext): boolean {
    return true; // Always available
  }

  handleInput(input: string, key: any, context: CommandContext): boolean {
    const { cleanupAndExit } = context;

    // Ctrl+C or specific escape sequence
    if ((key.ctrl && input === "c") || input === "\u0003") {
      cleanupAndExit();
      return true;
    }

    // Global quit
    if (
      input.toLowerCase() === "q" &&
      (context.state.mode === "normal" ||
        context.state.mode === "auth-required" ||
        context.state.mode === "loading")
    ) {
      cleanupAndExit();
      return true;
    }

    // Global log viewer
    if (input.toLowerCase() === "l" && context.state.mode === "auth-required") {
      void context.executeCommand("logs");
      return true;
    }

    return false;
  }
}
