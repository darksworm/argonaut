import chalk from "chalk";
import { Box, Text } from "ink";
import type React from "react";
import { hostFromUrl } from "../../config/paths";
import { useAppState } from "../../contexts/AppStateContext";

export const LoadingView: React.FC = () => {
  const { state } = useAppState();
  const { server, terminal } = state;

  if (state.mode !== "loading") {
    return null;
  }

  const spinChar = "⠋";
  const loadingHeader = `${chalk.bold("View:")} ${chalk.yellow("LOADING")} • ${chalk.bold("Context:")} ${chalk.cyan(server ? hostFromUrl(server.config.baseUrl) : "—")}`;

  return (
    <Box
      flexDirection="column"
      borderStyle="round"
      borderColor="magenta"
      paddingX={1}
      height={terminal.rows - 1}
    >
      <Box>
        <Text>{loadingHeader}</Text>
      </Box>
      <Box flexGrow={1} alignItems="center" justifyContent="center">
        <Text color="yellow">
          {spinChar} Connecting & fetching applications…
        </Text>
      </Box>
      <Box>
        <Text dimColor>Starting…</Text>
      </Box>
    </Box>
  );
};
