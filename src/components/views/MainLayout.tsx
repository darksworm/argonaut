import { Box } from "ink";
import type React from "react";
import packageJson from "../../../package.json";
import { hostFromUrl } from "../../config/paths";
import { useAppState } from "../../contexts/AppStateContext";
import { fmtScope, uniqueSorted } from "../../utils";
import ArgoNautBanner from "../Banner";
import Help from "../Help";
import OfficeSupplyManager from "../OfficeSupplyManager";
import { ResourceStream } from "../ResourceStream";
import { ListView } from "./ListView";

interface MainLayoutProps {
  visibleItems: any[];
  onDrillDown: () => void;
}

export const MainLayout: React.FC<MainLayoutProps> = ({
  visibleItems,
  onDrillDown,
}) => {
  const { state, dispatch } = useAppState();
  const { mode, terminal, server, apiVersion, selections, modals } = state;

  const { scopeClusters, scopeNamespaces, scopeProjects } = selections;
  const { syncViewApp } = modals;

  // Height calculations
  const BORDER_LINES = 2;
  const HEADER_CONTEXT = 6;
  const SEARCH_LINES = mode === "search" ? 1 : 0;
  const TABLE_HEADER_LINES = 1;
  const TAG_LINE = 1;
  const STATUS_LINES = 1;
  const COMMAND_LINES = mode === "command" ? 1 : 0;

  const OVERHEAD =
    BORDER_LINES +
    HEADER_CONTEXT +
    SEARCH_LINES +
    TABLE_HEADER_LINES +
    TAG_LINE +
    STATUS_LINES +
    COMMAND_LINES;

  const availableRows = Math.max(0, terminal.rows - OVERHEAD);
  const barOpenExtra = mode === "search" || mode === "command" ? 1 : 0;
  const listRows = Math.max(0, availableRows - barOpenExtra);

  // Special view modes
  if (mode === "external") {
    return null;
  }

  if (mode === "rulerline") {
    return (
      <OfficeSupplyManager
        onExit={() => dispatch({ type: "SET_MODE", payload: "normal" })}
      />
    );
  }

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

      <Box
        flexDirection="column"
        flexGrow={1}
        borderStyle="round"
        borderColor="magenta"
        paddingX={1}
        flexWrap="nowrap"
      >
        {mode === "help" ? (
          <Box flexDirection="column" marginTop={1} flexGrow={1}>
            <Help />
          </Box>
        ) : mode === "resources" && server && syncViewApp ? (
          <Box flexDirection="column" flexGrow={1}>
            <ResourceStream
              serverConfig={server.config}
              token={server.token}
              appName={syncViewApp}
              appNamespace={
                state.apps.find((a) => a.name === syncViewApp)?.appNamespace
              }
              onExit={() => {
                dispatch({ type: "SET_MODE", payload: "normal" });
                dispatch({ type: "SET_SYNC_VIEW_APP", payload: null });
              }}
            />
          </Box>
        ) : (
          <ListView visibleItems={visibleItems} availableRows={listRows} />
        )}
      </Box>
    </Box>
  );
};
