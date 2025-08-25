// @ts-nocheck

import chalk from "chalk";
import { Box, Text } from "ink";
import type React from "react";
import ArgoNautBanner from "./Banner";

interface Props {
  server: string | null;
  apiVersion: string;
  termCols: number;
  termRows: number;
  clusterScope: string;
  namespaceScope: string;
  projectScope: string;
  argonautVersion: string;
  message?: string;
  status: string;
}

const AuthRequiredView: React.FC<Props> = ({
  server,
  apiVersion,
  termCols,
  termRows,
  clusterScope,
  namespaceScope,
  projectScope,
  argonautVersion,
  message,
  status,
}) => {
  const headerMsg = `${chalk.bold("View:")} ${chalk.red("AUTH REQUIRED")} • ${chalk.bold("Context:")} ${chalk.cyan(server || "—")}`;
  const instructions = [
    "1. Run: argocd login <your-argocd-server>",
    "2. Follow prompts to authenticate",
    "3. Re-run argonaut",
  ];

  return (
    <Box flexDirection="column" paddingX={1} height={termRows - 1}>
      <ArgoNautBanner
        server={server}
        clusterScope={clusterScope}
        namespaceScope={namespaceScope}
        projectScope={projectScope}
        termCols={termCols}
        termRows={termRows}
        apiVersion={apiVersion}
        argonautVersion={argonautVersion}
      />

      <Box
        flexDirection="column"
        flexGrow={1}
        borderStyle="round"
        borderColor="red"
        paddingX={2}
        paddingY={1}
      >
        <Box
          flexGrow={1}
          alignItems="center"
          justifyContent="center"
          flexDirection="column"
        >
          <Text>{chalk.bgRed.white.bold(" AUTHENTICATION REQUIRED ")}</Text>
          <Box height={1} />
          <Text color="redBright" bold>
            {message || "Please login to ArgoCD before running argonaut."}
          </Text>
          <Box height={1} />
          {instructions.map((line) => (
            <Text key={line} dimColor>
              - {line}
            </Text>
          ))}
          <Box height={1} />
          <Text dimColor>
            Current context: {server ? chalk.cyan(server) : "unknown"}
          </Text>
          <Text dimColor>
            Press {chalk.yellow("l")} to view logs, {chalk.yellow("q")} to quit.
          </Text>
        </Box>
      </Box>

      <Box justifyContent="space-between">
        <Box>
          <Text dimColor>{headerMsg}</Text>
        </Box>
        <Box>
          <Text dimColor>{status}</Text>
        </Box>
      </Box>
    </Box>
  );
};

export default AuthRequiredView;
