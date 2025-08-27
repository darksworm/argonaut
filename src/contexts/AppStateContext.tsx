import React, {
  createContext,
  type ReactNode,
  useContext,
  useReducer,
} from "react";
import type { AppItem, Mode, View } from "../types/domain";
import type { Server } from "../types/server";

// State interfaces
export interface TerminalState {
  rows: number;
  cols: number;
}

export interface NavigationState {
  view: View;
  selectedIdx: number;
  lastGPressed: number;
  lastEscPressed: number;
}

export interface SelectionState {
  scopeClusters: Set<string>;
  scopeNamespaces: Set<string>;
  scopeProjects: Set<string>;
  selectedApps: Set<string>;
}

export interface UIState {
  searchQuery: string;
  activeFilter: string;
  command: string;
  isVersionOutdated: boolean;
  latestVersion?: string;
}

export interface ModalState {
  confirmTarget: string | null;
  confirmSyncPrune: boolean;
  confirmSyncWatch: boolean;
  rollbackAppName: string | null;
  syncViewApp: string | null;
}

export interface AppState {
  // Core state
  mode: Mode;
  terminal: TerminalState;
  navigation: NavigationState;
  selections: SelectionState;
  ui: UIState;
  modals: ModalState;

  // Data
  server: Server | null;
  apps: AppItem[];
  apiVersion: string;

  // Cleanup
  loadingAbortController: AbortController | null;
}

// Action types
export type AppAction =
  | { type: "SET_MODE"; payload: Mode }
  | { type: "SET_TERMINAL_SIZE"; payload: { rows: number; cols: number } }
  | { type: "SET_VIEW"; payload: View }
  | { type: "SET_SELECTED_IDX"; payload: number }
  | { type: "SET_SERVER"; payload: Server | null }
  | { type: "SET_APPS"; payload: AppItem[] }
  | { type: "SET_API_VERSION"; payload: string }
  | { type: "SET_SEARCH_QUERY"; payload: string }
  | { type: "SET_ACTIVE_FILTER"; payload: string }
  | { type: "SET_COMMAND"; payload: string }
  | { type: "SET_SCOPE_CLUSTERS"; payload: Set<string> }
  | { type: "SET_SCOPE_NAMESPACES"; payload: Set<string> }
  | { type: "SET_SCOPE_PROJECTS"; payload: Set<string> }
  | { type: "SET_SELECTED_APPS"; payload: Set<string> }
  | { type: "SET_CONFIRM_TARGET"; payload: string | null }
  | { type: "SET_CONFIRM_SYNC_PRUNE"; payload: boolean }
  | { type: "SET_CONFIRM_SYNC_WATCH"; payload: boolean }
  | { type: "SET_ROLLBACK_APP_NAME"; payload: string | null }
  | { type: "SET_SYNC_VIEW_APP"; payload: string | null }
  | { type: "SET_VERSION_OUTDATED"; payload: boolean }
  | { type: "SET_LATEST_VERSION"; payload: string | undefined }
  | { type: "SET_LAST_G_PRESSED"; payload: number }
  | { type: "SET_LAST_ESC_PRESSED"; payload: number }
  | { type: "SET_LOADING_ABORT_CONTROLLER"; payload: AbortController | null }
  | { type: "CLEAR_LOWER_LEVEL_SELECTIONS"; payload: View }
  | { type: "RESET_NAVIGATION"; payload?: { view?: View } }
  | { type: "CLEAR_ALL_SELECTIONS" }
  | { type: "CLEAR_FILTERS" };

// Initial state
export const initialState: AppState = {
  mode: "loading",
  terminal: {
    rows: process.stdout.rows || 24,
    cols: process.stdout.columns || 80,
  },
  navigation: {
    view: "clusters",
    selectedIdx: 0,
    lastGPressed: 0,
    lastEscPressed: 0,
  },
  selections: {
    scopeClusters: new Set(),
    scopeNamespaces: new Set(),
    scopeProjects: new Set(),
    selectedApps: new Set(),
  },
  ui: {
    searchQuery: "",
    activeFilter: "",
    command: ":",
    isVersionOutdated: false,
    latestVersion: undefined,
  },
  modals: {
    confirmTarget: null,
    confirmSyncPrune: false,
    confirmSyncWatch: true,
    rollbackAppName: null,
    syncViewApp: null,
  },
  server: null,
  apps: [],
  apiVersion: "",
  loadingAbortController: null,
};

// Reducer
export function appStateReducer(state: AppState, action: AppAction): AppState {
  switch (action.type) {
    case "SET_MODE":
      return { ...state, mode: action.payload };

    case "SET_TERMINAL_SIZE":
      return {
        ...state,
        terminal: { ...state.terminal, ...action.payload },
      };

    case "SET_VIEW":
      return {
        ...state,
        navigation: { ...state.navigation, view: action.payload },
      };

    case "SET_SELECTED_IDX":
      return {
        ...state,
        navigation: { ...state.navigation, selectedIdx: action.payload },
      };

    case "SET_SERVER":
      return { ...state, server: action.payload };

    case "SET_APPS":
      return { ...state, apps: action.payload };

    case "SET_API_VERSION":
      return { ...state, apiVersion: action.payload };

    case "SET_SEARCH_QUERY":
      return {
        ...state,
        ui: { ...state.ui, searchQuery: action.payload },
      };

    case "SET_ACTIVE_FILTER":
      return {
        ...state,
        ui: { ...state.ui, activeFilter: action.payload },
      };

    case "SET_COMMAND":
      return {
        ...state,
        ui: { ...state.ui, command: action.payload },
      };

    case "SET_SCOPE_CLUSTERS":
      return {
        ...state,
        selections: { ...state.selections, scopeClusters: action.payload },
      };

    case "SET_SCOPE_NAMESPACES":
      return {
        ...state,
        selections: { ...state.selections, scopeNamespaces: action.payload },
      };

    case "SET_SCOPE_PROJECTS":
      return {
        ...state,
        selections: { ...state.selections, scopeProjects: action.payload },
      };

    case "SET_SELECTED_APPS":
      return {
        ...state,
        selections: { ...state.selections, selectedApps: action.payload },
      };

    case "SET_CONFIRM_TARGET":
      return {
        ...state,
        modals: { ...state.modals, confirmTarget: action.payload },
      };

    case "SET_CONFIRM_SYNC_PRUNE":
      return {
        ...state,
        modals: { ...state.modals, confirmSyncPrune: action.payload },
      };

    case "SET_CONFIRM_SYNC_WATCH":
      return {
        ...state,
        modals: { ...state.modals, confirmSyncWatch: action.payload },
      };

    case "SET_ROLLBACK_APP_NAME":
      return {
        ...state,
        modals: { ...state.modals, rollbackAppName: action.payload },
      };

    case "SET_SYNC_VIEW_APP":
      return {
        ...state,
        modals: { ...state.modals, syncViewApp: action.payload },
      };

    case "SET_VERSION_OUTDATED":
      return {
        ...state,
        ui: { ...state.ui, isVersionOutdated: action.payload },
      };

    case "SET_LATEST_VERSION":
      return {
        ...state,
        ui: { ...state.ui, latestVersion: action.payload },
      };

    case "SET_LAST_G_PRESSED":
      return {
        ...state,
        navigation: { ...state.navigation, lastGPressed: action.payload },
      };

    case "SET_LAST_ESC_PRESSED":
      return {
        ...state,
        navigation: { ...state.navigation, lastEscPressed: action.payload },
      };

    case "SET_LOADING_ABORT_CONTROLLER":
      return { ...state, loadingAbortController: action.payload };

    case "CLEAR_LOWER_LEVEL_SELECTIONS": {
      const view = action.payload;
      const emptySet = new Set<string>();
      const selections = { ...state.selections };

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

      return { ...state, selections };
    }

    case "RESET_NAVIGATION":
      return {
        ...state,
        navigation: {
          ...state.navigation,
          selectedIdx: 0,
          view: action.payload?.view ?? state.navigation.view,
        },
        ui: {
          ...state.ui,
          activeFilter: "",
          searchQuery: "",
        },
      };

    case "CLEAR_ALL_SELECTIONS":
      return {
        ...state,
        selections: {
          scopeClusters: new Set(),
          scopeNamespaces: new Set(),
          scopeProjects: new Set(),
          selectedApps: new Set(),
        },
      };

    case "CLEAR_FILTERS":
      return {
        ...state,
        ui: {
          ...state.ui,
          activeFilter: "",
          searchQuery: "",
        },
      };

    default:
      return state;
  }
}

// Context
export const AppStateContext = createContext<{
  state: AppState;
  dispatch: React.Dispatch<AppAction>;
} | null>(null);

// Provider component
export interface AppStateProviderProps {
  children: ReactNode;
  initialState?: Partial<AppState>;
}

export const AppStateProvider: React.FC<AppStateProviderProps> = ({
  children,
  initialState: providedInitialState,
}) => {
  const finalInitialState = providedInitialState
    ? { ...initialState, ...providedInitialState }
    : initialState;

  const [state, dispatch] = useReducer(appStateReducer, finalInitialState);

  return (
    <AppStateContext.Provider value={{ state, dispatch }}>
      {children}
    </AppStateContext.Provider>
  );
};

// Hook to use the context
export const useAppState = () => {
  const context = useContext(AppStateContext);
  if (!context) {
    throw new Error("useAppState must be used within an AppStateProvider");
  }
  return context;
};

// Convenience selector hooks
export const useMode = () => {
  const { state } = useAppState();
  return state.mode;
};

export const useNavigation = () => {
  const { state } = useAppState();
  return state.navigation;
};

export const useSelections = () => {
  const { state } = useAppState();
  return state.selections;
};

export const useTerminal = () => {
  const { state } = useAppState();
  return state.terminal;
};

export const useUI = () => {
  const { state } = useAppState();
  return state.ui;
};

export const useModals = () => {
  const { state } = useAppState();
  return state.modals;
};

export const useServer = () => {
  const { state } = useAppState();
  return state.server;
};

export const useAppsData = () => {
  const { state } = useAppState();
  return { apps: state.apps, apiVersion: state.apiVersion };
};
