import type { View } from "../types/domain";

export interface NavigationState {
  view: View;
  selectedIdx: number;
  lastGPressed: number;
  lastEscPressed: number;
}

export type NavigationAction =
  | { type: "SET_VIEW"; payload: View }
  | { type: "SET_SELECTED_IDX"; payload: number }
  | { type: "SET_LAST_G_PRESSED"; payload: number }
  | { type: "SET_LAST_ESC_PRESSED"; payload: number }
  | { type: "RESET_NAVIGATION"; payload?: { view?: View } };

export const initialNavigationState: NavigationState = {
  view: "clusters",
  selectedIdx: 0,
  lastGPressed: 0,
  lastEscPressed: 0,
};

/**
 * Pure reducer for navigation state
 * Extracted from the large appStateReducer for better maintainability
 */
export function navigationReducer(
  state: NavigationState,
  action: NavigationAction,
): NavigationState {
  switch (action.type) {
    case "SET_VIEW":
      return {
        ...state,
        view: action.payload,
      };

    case "SET_SELECTED_IDX":
      return {
        ...state,
        selectedIdx: action.payload,
      };

    case "SET_LAST_G_PRESSED":
      return {
        ...state,
        lastGPressed: action.payload,
      };

    case "SET_LAST_ESC_PRESSED":
      return {
        ...state,
        lastEscPressed: action.payload,
      };

    case "RESET_NAVIGATION":
      return {
        ...state,
        selectedIdx: 0,
        view: action.payload?.view ?? state.view,
      };

    default:
      return state;
  }
}
