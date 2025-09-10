export interface UIState {
  searchQuery: string;
  activeFilter: string;
  command: string;
  isVersionOutdated: boolean;
  latestVersion?: string;
  commandInputKey: number;
}

export type UIAction =
  | { type: "SET_SEARCH_QUERY"; payload: string }
  | { type: "SET_ACTIVE_FILTER"; payload: string }
  | { type: "SET_COMMAND"; payload: string }
  | { type: "BUMP_COMMAND_INPUT_KEY" }
  | { type: "SET_VERSION_OUTDATED"; payload: boolean }
  | { type: "SET_LATEST_VERSION"; payload: string | undefined }
  | { type: "CLEAR_FILTERS" };

export const initialUIState: UIState = {
  searchQuery: "",
  activeFilter: "",
  command: "",
  isVersionOutdated: false,
  latestVersion: undefined,
  commandInputKey: 0,
};

/**
 * Pure reducer for UI state
 * Handles search, filters, commands, and version info
 */
export function uiReducer(state: UIState, action: UIAction): UIState {
  switch (action.type) {
    case "SET_SEARCH_QUERY":
      return {
        ...state,
        searchQuery: action.payload,
      };

    case "SET_ACTIVE_FILTER":
      return {
        ...state,
        activeFilter: action.payload,
      };

    case "SET_COMMAND":
      return {
        ...state,
        command: action.payload,
      };

    case "BUMP_COMMAND_INPUT_KEY":
      return {
        ...state,
        commandInputKey: state.commandInputKey + 1,
      };

    case "SET_VERSION_OUTDATED":
      return {
        ...state,
        isVersionOutdated: action.payload,
      };

    case "SET_LATEST_VERSION":
      return {
        ...state,
        latestVersion: action.payload,
      };

    case "CLEAR_FILTERS":
      return {
        ...state,
        activeFilter: "",
        searchQuery: "",
      };

    default:
      return state;
  }
}
