import type { View } from "../types/domain";

/**
 * Result of a navigation operation
 */
export interface NavigationUpdate {
  newView?: View;
  scopeClusters?: Set<string>;
  scopeNamespaces?: Set<string>;
  scopeProjects?: Set<string>;
  selectedApps?: Set<string>;
  shouldResetNavigation?: boolean;
  shouldClearLowerLevelSelections?: boolean;
}

/**
 * Selection operation result
 */
export interface SelectionUpdate {
  selectedApps: Set<string>;
}

/**
 * Pure navigation service extracted from React hooks
 * Contains all navigation business logic without React dependencies
 * Perfect for testing and Go migration
 */
export class NavigationService {
  /**
   * Drill down navigation logic - moves from clusters -> namespaces -> projects -> apps
   */
  static drillDown(
    currentView: View,
    _selectedItem: any,
    visibleItems: any[],
    selectedIdx: number,
  ): NavigationUpdate | null {
    const item = visibleItems[selectedIdx];
    if (!item) return null;

    const val = String(item);
    const next = new Set([val]);

    const result: NavigationUpdate = {
      shouldResetNavigation: true,
      shouldClearLowerLevelSelections: true,
    };

    switch (currentView) {
      case "clusters":
        result.newView = "namespaces";
        result.scopeClusters = next;
        break;
      case "namespaces":
        result.newView = "projects";
        result.scopeNamespaces = next;
        break;
      case "projects":
        result.newView = "apps";
        result.scopeProjects = next;
        break;
      default:
        return null; // Can't drill down from apps view
    }

    return result;
  }

  /**
   * Toggle selection logic - only works in apps view
   */
  static toggleSelection(
    currentView: View,
    _selectedItem: any,
    visibleItems: any[],
    selectedIdx: number,
    currentSelectedApps: Set<string>,
  ): SelectionUpdate | null {
    // Only allow toggle selection in apps view
    if (currentView !== "apps") {
      return null;
    }

    const item = visibleItems[selectedIdx];
    if (!item) return null;

    const appName = (item as any).name;
    const next = new Set(currentSelectedApps);

    if (next.has(appName)) {
      next.delete(appName);
    } else {
      next.add(appName);
    }

    return {
      selectedApps: next,
    };
  }

  /**
   * Validate and adjust selectedIdx to stay within bounds
   */
  static validateBounds(selectedIdx: number, itemCount: number): number {
    if (itemCount === 0) return 0;
    return Math.max(0, Math.min(selectedIdx, itemCount - 1));
  }

  /**
   * Clear lower-level selections based on current view
   */
  static clearLowerLevelSelections(view: View): {
    scopeNamespaces?: Set<string>;
    scopeProjects?: Set<string>;
    selectedApps?: Set<string>;
  } {
    const emptySet = new Set<string>();
    const result: any = {};

    switch (view) {
      case "clusters":
        result.scopeNamespaces = emptySet;
        result.scopeProjects = emptySet;
        result.selectedApps = emptySet;
        break;
      case "namespaces":
        result.scopeProjects = emptySet;
        result.selectedApps = emptySet;
        break;
      case "projects":
        result.selectedApps = emptySet;
        break;
    }

    return result;
  }

  /**
   * Reset navigation state to defaults
   */
  static resetNavigation(view?: View): {
    selectedIdx: number;
    view?: View;
    activeFilter: string;
    searchQuery: string;
  } {
    return {
      selectedIdx: 0,
      view,
      activeFilter: "",
      searchQuery: "",
    };
  }

  /**
   * Clear all selections
   */
  static clearAllSelections(): {
    scopeClusters: Set<string>;
    scopeNamespaces: Set<string>;
    scopeProjects: Set<string>;
    selectedApps: Set<string>;
  } {
    return {
      scopeClusters: new Set(),
      scopeNamespaces: new Set(),
      scopeProjects: new Set(),
      selectedApps: new Set(),
    };
  }

  /**
   * Clear all filters and search
   */
  static clearFilters(): {
    activeFilter: string;
    searchQuery: string;
  } {
    return {
      activeFilter: "",
      searchQuery: "",
    };
  }

  /**
   * Determine if drill down is possible from current view
   */
  static canDrillDown(view: View): boolean {
    return view !== "apps";
  }

  /**
   * Determine if selection toggle is possible from current view
   */
  static canToggleSelection(view: View): boolean {
    return view === "apps";
  }

  /**
   * Get the next view in the drill down hierarchy
   */
  static getNextView(currentView: View): View | null {
    switch (currentView) {
      case "clusters":
        return "namespaces";
      case "namespaces":
        return "projects";
      case "projects":
        return "apps";
      default:
        return null;
    }
  }

  /**
   * Get the previous view in the drill down hierarchy
   */
  static getPreviousView(currentView: View): View | null {
    switch (currentView) {
      case "apps":
        return "projects";
      case "projects":
        return "namespaces";
      case "namespaces":
        return "clusters";
      default:
        return null;
    }
  }
}
