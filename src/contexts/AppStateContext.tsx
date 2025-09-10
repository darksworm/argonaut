import React, {
  createContext,
  type ReactNode,
  useContext,
  useReducer,
} from "react";
import {
  initialModalState,
  type ModalAction,
  type ModalState,
  modalReducer,
} from "../state/modal-reducer";
import {
  initialNavigationState,
  type NavigationAction,
  type NavigationState,
  navigationReducer,
} from "../state/navigation-reducer";
import {
  initialSelectionState,
  type SelectionAction,
  type SelectionState,
  selectionReducer,
} from "../state/selection-reducer";
import {
  initialServerState,
  type ServerAction,
  type ServerState,
  serverReducer,
} from "../state/server-reducer";
import {
  initialUIState,
  type UIAction,
  type UIState,
  uiReducer,
} from "../state/ui-reducer";
import type { AppItem, Mode } from "../types/domain";
import type { Server } from "../types/server";

// Re-export types from individual reducers
export type {
  NavigationState,
  SelectionState,
  UIState,
  ModalState,
  ServerState,
};

// State interfaces
export interface TerminalState {
  rows: number;
  cols: number;
}

export interface AppState {
  // Core state
  mode: Mode;
  terminal: TerminalState;
  navigation: NavigationState;
  selections: SelectionState;
  ui: UIState;
  modals: ModalState;
  serverState: ServerState;

  // Data
  apps: AppItem[];
  apiVersion: string;

  // Cleanup
  loadingAbortController: AbortController | null;

  // Backward compatibility
  server: Server | null;
}

// Legacy server access for backward compatibility
export const getServer = (state: AppState): Server | null =>
  state.serverState.server;

// Combined action type
export type AppAction =
  | { type: "SET_MODE"; payload: Mode }
  | { type: "SET_TERMINAL_SIZE"; payload: { rows: number; cols: number } }
  | { type: "SET_APPS"; payload: AppItem[] }
  | { type: "SET_API_VERSION"; payload: string }
  | { type: "SET_LOADING_ABORT_CONTROLLER"; payload: AbortController | null }
  | NavigationAction
  | SelectionAction
  | UIAction
  | ModalAction
  | ServerAction;

// Initial state using individual reducer states
export const initialState: AppState = {
  mode: "loading",
  terminal: {
    rows: process.stdout.rows || 24,
    cols: process.stdout.columns || 80,
  },
  navigation: initialNavigationState,
  selections: initialSelectionState,
  ui: initialUIState,
  modals: initialModalState,
  serverState: initialServerState,
  apps: [],
  apiVersion: "",
  loadingAbortController: null,
};

// Combined reducer using individual reducers
export function appStateReducer(state: AppState, action: AppAction): AppState {
  switch (action.type) {
    case "SET_MODE":
      return { ...state, mode: action.payload };

    case "SET_TERMINAL_SIZE":
      return {
        ...state,
        terminal: { ...state.terminal, ...action.payload },
      };

    case "SET_APPS":
      return { ...state, apps: action.payload };

    case "SET_API_VERSION":
      return { ...state, apiVersion: action.payload };

    case "SET_LOADING_ABORT_CONTROLLER":
      return { ...state, loadingAbortController: action.payload };

    default: {
      // Delegate to individual reducers
      const newServerState = serverReducer(
        state.serverState,
        action as ServerAction,
      );
      return {
        ...state,
        navigation: navigationReducer(
          state.navigation,
          action as NavigationAction,
        ),
        selections: selectionReducer(
          state.selections,
          action as SelectionAction,
        ),
        ui: uiReducer(state.ui, action as UIAction),
        modals: modalReducer(state.modals, action as ModalAction),
        serverState: newServerState,
        server: newServerState.server, // Keep computed property in sync
      };
    }
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
    ? {
        ...initialState,
        ...providedInitialState,
        ui: { ...initialState.ui, ...(providedInitialState.ui ?? {}) },
        navigation: {
          ...initialState.navigation,
          ...(providedInitialState.navigation ?? {}),
        },
        selections: {
          ...initialState.selections,
          ...(providedInitialState.selections ?? {}),
        },
        modals: {
          ...initialState.modals,
          ...(providedInitialState.modals ?? {}),
        },
        serverState: {
          ...initialState.serverState,
          ...(providedInitialState.serverState ?? {}),
        },
        server: providedInitialState.server ?? initialState.server,
      }
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
  return state.serverState.server;
};

export const useAppsData = () => {
  const { state } = useAppState();
  return { apps: state.apps, apiVersion: state.apiVersion };
};
