import { useEffect } from "react";
import { getApiVersion } from "../api/version";
import type { StatusLogger } from "../commands";
import { useAppState } from "../contexts/AppStateContext";
import type { AppOrchestrator } from "../services/app-orchestrator";
import { useApps } from "./useApps";

export const useLiveData = (
  orchestrator: AppOrchestrator,
  statusLog: StatusLogger,
) => {
  const { state, dispatch } = useAppState();
  const { server, mode } = state;

  // Live data via useApps hook
  const { apps: liveApps, status: appsStatus } = useApps(
    server,
    mode === "external" || mode === "loading",
    () => {
      // On auth error from background data flow
      orchestrator.handleAuthError(dispatch, statusLog);
    },
  );

  // Sync live apps data to state
  useEffect(() => {
    if (!server) return;
    if (mode === "external" || mode === "auth-required") return; // pause syncing state while in external/diff mode or auth required

    dispatch({ type: "SET_APPS", payload: liveApps });
    statusLog.set(appsStatus);
  }, [server, liveApps, appsStatus, mode, dispatch, statusLog.set]); // Remove statusLog from deps

  // Periodically refresh API version
  useEffect(() => {
    if (!server) return;

    const abortController = new AbortController();
    getApiVersion(server, { signal: abortController.signal, timeout: 5000 })
      .then((version) =>
        dispatch({ type: "SET_API_VERSION", payload: version }),
      )
      .catch((error) => {
        if (!abortController.signal.aborted) {
          console.warn("Failed to get API version:", error.message);
        }
      });

    return () => {
      abortController.abort();
    };
  }, [server, dispatch]);

  return {
    apps: liveApps,
    status: appsStatus,
  };
};
