import { Box } from "ink";
import type React from "react";
import packageJson from "../../../package.json";
import { hostFromUrl } from "../../config/paths";
import { useAppState } from "../../contexts/AppStateContext";
import { fmtScope } from "../../utils";
import ArgoNautBanner from "../Banner";
import Rollback from "../Rollback";

export const RollbackModal: React.FC = () => {
  const { state, dispatch } = useAppState();

  const { mode, terminal, modals, server, apps, apiVersion, selections } =
    state;
  const { rollbackAppName } = modals;
  const { scopeClusters, scopeNamespaces, scopeProjects } = selections;

  if (mode !== "rollback" || !rollbackAppName) {
    return null;
  }

  const availableRows = terminal.rows - 8; // Estimate for header overhead

  const handleClose = () => {
    dispatch({ type: "SET_MODE", payload: "normal" });
    dispatch({ type: "SET_ROLLBACK_APP_NAME", payload: null });
  };

  const handleStartWatching = (appName: string) => {
    dispatch({ type: "SET_SYNC_VIEW_APP", payload: appName });
    dispatch({ type: "SET_MODE", payload: "resources" });
    dispatch({ type: "SET_ROLLBACK_APP_NAME", payload: null });
  };

  return (
    <Box flexDirection="column" paddingX={1} height={terminal.rows - 1}>
      <ArgoNautBanner
        server={server ? hostFromUrl(server.config.baseUrl) : null}
        clusterScope={fmtScope(scopeClusters)}
        namespaceScope={fmtScope(scopeNamespaces)}
        projectScope={fmtScope(scopeProjects)}
        termCols={terminal.cols}
        termRows={availableRows}
        apiVersion={apiVersion}
        argonautVersion={packageJson.version}
      />
      <Rollback
        app={rollbackAppName}
        server={server}
        appNamespace={
          apps.find((a) => a.name === rollbackAppName)?.appNamespace
        }
        onClose={handleClose}
        onStartWatching={handleStartWatching}
      />
    </Box>
  );
};
