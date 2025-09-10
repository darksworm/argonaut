import type { View } from "../types/domain";

export interface SelectionState {
  scopeClusters: Set<string>;
  scopeNamespaces: Set<string>;
  scopeProjects: Set<string>;
  selectedApps: Set<string>;
}

export type SelectionAction =
  | { type: "SET_SCOPE_CLUSTERS"; payload: Set<string> }
  | { type: "SET_SCOPE_NAMESPACES"; payload: Set<string> }
  | { type: "SET_SCOPE_PROJECTS"; payload: Set<string> }
  | { type: "SET_SELECTED_APPS"; payload: Set<string> }
  | { type: "CLEAR_LOWER_LEVEL_SELECTIONS"; payload: View }
  | { type: "CLEAR_ALL_SELECTIONS" };

export const initialSelectionState: SelectionState = {
  scopeClusters: new Set(),
  scopeNamespaces: new Set(),
  scopeProjects: new Set(),
  selectedApps: new Set(),
};

/**
 * Pure reducer for selection state
 * Handles cluster/namespace/project/app selection logic
 */
export function selectionReducer(
  state: SelectionState,
  action: SelectionAction,
): SelectionState {
  switch (action.type) {
    case "SET_SCOPE_CLUSTERS":
      return {
        ...state,
        scopeClusters: action.payload,
      };

    case "SET_SCOPE_NAMESPACES":
      return {
        ...state,
        scopeNamespaces: action.payload,
      };

    case "SET_SCOPE_PROJECTS":
      return {
        ...state,
        scopeProjects: action.payload,
      };

    case "SET_SELECTED_APPS":
      return {
        ...state,
        selectedApps: action.payload,
      };

    case "CLEAR_LOWER_LEVEL_SELECTIONS": {
      const view = action.payload;
      const emptySet = new Set<string>();
      const selections = { ...state };

      switch (view) {
        case "clusters":
          selections.scopeNamespaces = emptySet;
          selections.scopeProjects = emptySet;
          selections.selectedApps = emptySet;
          break;
        case "namespaces":
          selections.scopeProjects = emptySet;
          selections.selectedApps = emptySet;
          break;
        case "projects":
          selections.selectedApps = emptySet;
          break;
      }

      return selections;
    }

    case "CLEAR_ALL_SELECTIONS":
      return {
        scopeClusters: new Set(),
        scopeNamespaces: new Set(),
        scopeProjects: new Set(),
        selectedApps: new Set(),
      };

    default:
      return state;
  }
}
