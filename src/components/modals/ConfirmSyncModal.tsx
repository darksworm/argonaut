import { useInput } from "ink";
import type React from "react";
import { syncApp } from "../../api/applications.command";
import { useAppState } from "../../contexts/AppStateContext";
import { useStatus } from "../../hooks/useStatus";
import ConfirmationBox from "../ConfirmationBox";

export const ConfirmSyncModal: React.FC = () => {
  const { state, dispatch } = useAppState();
  const [, statusLog] = useStatus("Ready");

  const { server, apps, modals, selections } = state;
  const { confirmTarget, confirmSyncPrune, confirmSyncWatch } = modals;
  const { selectedApps } = selections;

  // Handle input for the confirm modal
  useInput((input, key) => {
    if (state.mode !== "confirm-sync") return;

    // Esc or 'q' aborts immediately
    if (key.escape || input.toLowerCase() === "q") {
      handleConfirm(false);
      return;
    }

    // Toggle prune
    if (input.toLowerCase() === "p") {
      dispatch({ type: "SET_CONFIRM_SYNC_PRUNE", payload: !confirmSyncPrune });
      return;
    }

    // Toggle watch (only when not multi)
    if (input.toLowerCase() === "w") {
      if (confirmTarget !== "__MULTI__") {
        dispatch({
          type: "SET_CONFIRM_SYNC_WATCH",
          payload: !confirmSyncWatch,
        });
      }
      return;
    }

    // All other handling is done via the ConfirmationBox component
  });

  const handleConfirm = async (yes: boolean) => {
    dispatch({ type: "SET_MODE", payload: "normal" });
    const isMulti = confirmTarget === "__MULTI__";
    const names = isMulti
      ? Array.from(selectedApps)
      : confirmTarget
        ? [confirmTarget]
        : [];

    dispatch({ type: "SET_CONFIRM_TARGET", payload: null });

    if (!yes) {
      statusLog.info("Sync cancelled.", "sync");
      return;
    }

    if (!server) {
      statusLog.error("Not authenticated.", "auth");
      return;
    }

    try {
      statusLog.info(
        `Syncing ${isMulti ? `${names.length} app(s)` : names[0]}…`,
        "sync",
      );

      for (const n of names) {
        const app = apps.find((a) => a.name === n);
        syncApp(server, n, {
          prune: confirmSyncPrune,
          appNamespace: app?.appNamespace,
        });
      }

      statusLog.info(
        `Sync initiated for ${isMulti ? `${names.length} app(s)` : names[0]}.`,
        "sync",
      );

      // Show resource stream only for single-app syncs and when watch is enabled
      if (!isMulti && confirmSyncWatch) {
        dispatch({ type: "SET_SYNC_VIEW_APP", payload: names[0] });
        dispatch({ type: "SET_MODE", payload: "resources" });
      } else {
        // After syncing multiple apps, clear the selection
        if (isMulti) {
          dispatch({ type: "SET_SELECTED_APPS", payload: new Set() });
        }
      }
    } catch (e: any) {
      statusLog.error(`Sync failed: ${e.message}`, "sync");
    }
  };

  if (state.mode !== "confirm-sync" || !confirmTarget) {
    return null;
  }

  return (
    <ConfirmationBox
      title={
        confirmTarget === "__MULTI__"
          ? "Sync applications?"
          : "Sync application?"
      }
      message={
        confirmTarget === "__MULTI__"
          ? "Do you want to sync"
          : "Do you want to sync"
      }
      target={
        confirmTarget === "__MULTI__"
          ? String(selectedApps.size)
          : confirmTarget
      }
      isMulti={confirmTarget === "__MULTI__"}
      options={[
        {
          key: "p",
          label: "Prune",
          value: confirmSyncPrune,
        },
        {
          key: "w",
          label: "Watch",
          value: confirmSyncWatch,
          disabled: confirmTarget === "__MULTI__",
        },
      ]}
      onConfirm={handleConfirm}
    />
  );
};
