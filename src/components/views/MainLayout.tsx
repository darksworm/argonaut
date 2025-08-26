import { Box, Text } from "ink";
import type React from "react";
import packageJson from "../../../package.json";
import { hostFromUrl } from "../../config/paths";
import { useAppState } from "../../contexts/AppStateContext";
import { fmtScope } from "../../utils";
import ArgoNautBanner from "../Banner";
import OfficeSupplyManager from "../OfficeSupplyManager";
import { ResourceStream } from "../ResourceStream";
import { CommandBar } from "./CommandBar";
import { ListView } from "./ListView";
import { SearchBar } from "./SearchBar";

interface MainLayoutProps {
  visibleItems: any[];
  onDrillDown: () => void;
  commandRegistry: any;
  onExecuteCommand: (command: string, ...args: string[]) => void;
  status: string;
  modal?: React.ReactNode;
}

export const MainLayout: React.FC<MainLayoutProps> = ({
  visibleItems,
  onDrillDown,
  commandRegistry,
  onExecuteCommand,
  status,
  modal,
}) => {
  const { state, dispatch } = useAppState();
  const { mode, terminal, server, apiVersion, selections, modals, ui } = state;

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

      {/* Input bars between header and main content */}
      <SearchBar onSubmit={onDrillDown} />
      <CommandBar
        commandRegistry={commandRegistry}
        onExecuteCommand={onExecuteCommand}
      />

      {/* Modal appears here if present */}
      {modal}

      <Box
        flexDirection="column"
        flexGrow={1}
        borderStyle="round"
        borderColor="magenta"
        paddingX={1}
        flexWrap="nowrap"
      >
        {mode === "resources" && server && syncViewApp ? (
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

      {/* Status line outside the main box */}
      <Box justifyContent="space-between">
        <Box>
          <Text dimColor>
            {ui.activeFilter && state.navigation.view === "apps"
              ? `<${state.navigation.view}:${ui.activeFilter}>`
              : `<${state.navigation.view}>`}
          </Text>
        </Box>
        <Box>
          <Text dimColor>
            {status} •{" "}
            {visibleItems.length
              ? `${state.navigation.selectedIdx + 1}/${visibleItems.length}`
              : "0/0"}
            {state.ui.isVersionOutdated && (
              <Text color="yellow"> • Update available!</Text>
            )}
          </Text>
        </Box>
      </Box>
    </Box>
  );
};
