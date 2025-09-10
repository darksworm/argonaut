import type { AppItem, Mode } from "../types/domain";
import type { Server } from "../types/server";

export interface TerminalState {
  rows: number;
  cols: number;
}

export interface ServerState {
  mode: Mode;
  terminal: TerminalState;
  server: Server | null;
  apps: AppItem[];
  apiVersion: string;
  loadingAbortController: AbortController | null;
}

export type ServerAction =
  | { type: "SET_MODE"; payload: Mode }
  | { type: "SET_TERMINAL_SIZE"; payload: { rows: number; cols: number } }
  | { type: "SET_SERVER"; payload: Server | null }
  | { type: "SET_APPS"; payload: AppItem[] }
  | { type: "SET_API_VERSION"; payload: string }
  | { type: "SET_LOADING_ABORT_CONTROLLER"; payload: AbortController | null };

export const initialServerState: ServerState = {
  mode: "loading",
  terminal: {
    rows: process.stdout.rows || 24,
    cols: process.stdout.columns || 80,
  },
  server: null,
  apps: [],
  apiVersion: "",
  loadingAbortController: null,
};

/**
 * Pure reducer for server/core app state
 * Handles connection, data loading, and terminal state
 */
export function serverReducer(
  state: ServerState,
  action: ServerAction,
): ServerState {
  switch (action.type) {
    case "SET_MODE":
      return { ...state, mode: action.payload };

    case "SET_TERMINAL_SIZE":
      return {
        ...state,
        terminal: { ...state.terminal, ...action.payload },
      };

    case "SET_SERVER":
      return { ...state, server: action.payload };

    case "SET_APPS":
      return { ...state, apps: action.payload };

    case "SET_API_VERSION":
      return { ...state, apiVersion: action.payload };

    case "SET_LOADING_ABORT_CONTROLLER":
      return { ...state, loadingAbortController: action.payload };

    default:
      return state;
  }
}
