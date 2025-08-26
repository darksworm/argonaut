import type React from "react";
import packageJson from "../../package.json";
import { createDefaultCommandRegistry } from "../commands";
import { hostFromUrl } from "../config/paths";
import { AppStateProvider, useAppState } from "../contexts/AppStateContext";
import { useAppLifecycle } from "../hooks/useAppLifecycle";
import { useInputSystem } from "../hooks/useInputSystem";
import { useLiveData } from "../hooks/useLiveData";
import { useNavigationLogic } from "../hooks/useNavigationLogic";
import { useStatus } from "../hooks/useStatus";
import { DefaultAppOrchestrator } from "../services/app-orchestrator";
import { fmtScope } from "../utils";
import AuthRequiredView from "./AuthRequiredView";
import { ConfirmSyncModal, HelpModal, RollbackModal } from "./modals";
import { LoadingView, MainLayout } from "./views";

// Create singleton instances
const orchestrator = new DefaultAppOrchestrator();
const commandRegistry = createDefaultCommandRegistry();

// Main App component using our clean architecture
const AppContent: React.FC = () => {
  const { state } = useAppState();

  // Initialize lifecycle management
  const { cleanupAndExit } = useAppLifecycle(orchestrator);

  // Global status management (like legacy app)
  const [status, statusLog] = useStatus("Starting…");

  // Handle navigation logic
  const { drillDown, toggleSelection, visibleItems } = useNavigationLogic();

  // Set up input system with command registry
  const { executeCommand } = useInputSystem(
    commandRegistry,
    cleanupAndExit,
    statusLog,
    {
      drillDown,
      toggleSelection,
    },
  );

  // Manage live data synchronization
  useLiveData(orchestrator, statusLog);

  // Render based on current mode
  switch (state.mode) {
    case "loading":
      return <LoadingView />;

    case "auth-required":
      return (
        <AuthRequiredView
          server={
            state.server ? hostFromUrl(state.server.config.baseUrl) : null
          }
          apiVersion={state.apiVersion}
          termCols={state.terminal.cols}
          termRows={state.terminal.rows}
          clusterScope={fmtScope(state.selections.scopeClusters)}
          namespaceScope={fmtScope(state.selections.scopeNamespaces)}
          projectScope={fmtScope(state.selections.scopeProjects)}
          argonautVersion={packageJson.version}
          message="Authentication required"
          status={status}
        />
      );

    default:
      // Handle modal states directly in the main layout
      if (state.mode === "help") {
        return <HelpModal />;
      }

      return (
        <MainLayout
          visibleItems={visibleItems}
          onDrillDown={drillDown}
          commandRegistry={commandRegistry}
          onExecuteCommand={executeCommand}
          status={status}
          modal={
            state.mode === "confirm-sync" ? (
              <ConfirmSyncModal />
            ) : state.mode === "rollback" ? (
              <RollbackModal />
            ) : null
          }
        />
      );
  }
};

// Root App component with providers
export const App: React.FC = () => {
  return (
    <AppStateProvider>
      <AppContent />
    </AppStateProvider>
  );
};
