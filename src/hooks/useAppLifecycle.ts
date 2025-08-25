import { useApp } from "ink";
import { useEffect } from "react";
import { useAppState } from "../contexts/AppStateContext";
import type { AppOrchestrator } from "../services/app-orchestrator";
import { useStatus } from "./useStatus";

export const useAppLifecycle = (orchestrator: AppOrchestrator) => {
  const { exit } = useApp();
  const { state, dispatch } = useAppState();
  const [, statusLog] = useStatus("Starting…");

  // Initialize app on mount
  useEffect(() => {
    orchestrator.initializeApp(dispatch, statusLog);
  }, [dispatch, orchestrator, statusLog]); // Remove statusLog from deps as it changes frequently

  // Handle Ctrl+C (SIGINT) since Ink has exitOnCtrlC: false
  useEffect(() => {
    const handleSigInt = () => {
      orchestrator.cleanupAndExit(exit);
    };

    process.on("SIGINT", handleSigInt);

    return () => {
      process.off("SIGINT", handleSigInt);
    };
  }, [orchestrator, exit]);

  // Handle terminal resize
  useEffect(() => {
    return orchestrator.handleTerminalResize(dispatch);
  }, [orchestrator, dispatch]);

  // Track view changes in logger
  useEffect(() => {
    orchestrator.updateView(state.mode, state.navigation.view);
  }, [state.mode, state.navigation.view, orchestrator]);

  return {
    cleanupAndExit: () => orchestrator.cleanupAndExit(exit),
  };
};
