export interface ModalState {
  confirmTarget: string | null;
  confirmSyncPrune: boolean;
  confirmSyncWatch: boolean;
  rollbackAppName: string | null;
  syncViewApp: string | null;
}

export type ModalAction =
  | { type: "SET_CONFIRM_TARGET"; payload: string | null }
  | { type: "SET_CONFIRM_SYNC_PRUNE"; payload: boolean }
  | { type: "SET_CONFIRM_SYNC_WATCH"; payload: boolean }
  | { type: "SET_ROLLBACK_APP_NAME"; payload: string | null }
  | { type: "SET_SYNC_VIEW_APP"; payload: string | null };

export const initialModalState: ModalState = {
  confirmTarget: null,
  confirmSyncPrune: false,
  confirmSyncWatch: true,
  rollbackAppName: null,
  syncViewApp: null,
};

/**
 * Pure reducer for modal state
 * Handles all modal dialog states
 */
export function modalReducer(
  state: ModalState,
  action: ModalAction,
): ModalState {
  switch (action.type) {
    case "SET_CONFIRM_TARGET":
      return {
        ...state,
        confirmTarget: action.payload,
      };

    case "SET_CONFIRM_SYNC_PRUNE":
      return {
        ...state,
        confirmSyncPrune: action.payload,
      };

    case "SET_CONFIRM_SYNC_WATCH":
      return {
        ...state,
        confirmSyncWatch: action.payload,
      };

    case "SET_ROLLBACK_APP_NAME":
      return {
        ...state,
        rollbackAppName: action.payload,
      };

    case "SET_SYNC_VIEW_APP":
      return {
        ...state,
        syncViewApp: action.payload,
      };

    default:
      return state;
  }
}
