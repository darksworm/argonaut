import type { Server } from "../types/server";

export interface ServerState {
  server: Server | null;
}

export type ServerAction = { type: "SET_SERVER"; payload: Server | null };

export const initialServerState: ServerState = {
  server: null,
};

/**
 * Pure reducer for server state
 * Handles server connection information
 */
export function serverReducer(
  state: ServerState,
  action: ServerAction,
): ServerState {
  switch (action.type) {
    case "SET_SERVER":
      return { ...state, server: action.payload };

    default:
      return state;
  }
}
