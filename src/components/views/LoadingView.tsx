import chalk from "chalk";
import { Box, Text } from "ink";
import React, { useEffect, useState } from "react";
import { hostFromUrl } from "../../config/paths";
import { useAppState } from "../../contexts/AppStateContext";

export const LoadingView: React.FC = () => {
  const { state } = useAppState();
  const { server, terminal, loadingMessage } = state;

  if (state.mode !== "loading") {
    return null;
  }

  const frames = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"];
  const [frame, setFrame] = useState(0);

  useEffect(() => {
    const timer = setInterval(
      () => setFrame((f) => (f + 1) % frames.length),
      80,
    );
    return () => clearInterval(timer);
  }, []);

  const spinChar = frames[frame];
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
          {spinChar} {loadingMessage}
        </Text>
      </Box>
    </Box>
  );
};
