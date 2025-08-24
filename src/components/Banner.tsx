import chalk from "chalk";
import { Box, Text } from "ink";
import type React from "react";

interface ArgoNautBannerProps {
  server?: string | null;
  clusterScope?: string;
  namespaceScope?: string;
  projectScope?: string;
  termCols?: number;
  termRows?: number;
  apiVersion?: string;
  argonautVersion?: string;
}

interface ContextProps {
  paddingBottom: number;
  paddingTop: number;
  server?: string | null;
  clusterScope?: string;
  namespaceScope?: string;
  projectScope?: string;
  apiVersion?: string;
  isNarrow: boolean;
}

interface LogoProps {
  align: "center" | "flex-end";
  paddingBottom: number;
  argonautVersion?: string;
}

const Context: React.FC<ContextProps> = ({
  paddingBottom,
  paddingTop,
  server,
  clusterScope,
  namespaceScope,
  projectScope,
  apiVersion,
  isNarrow,
}) => (
  <Box
    flexDirection="column"
    paddingRight={2}
    paddingBottom={paddingBottom || 0}
    paddingTop={paddingTop || 0}
    alignSelf={isNarrow ? undefined : "flex-end"}
  >
    {server && (
      <>
        <Text>
          <Text bold>Context:</Text> <Text color="cyan">{server || "—"}</Text>
        </Text>
        {clusterScope && clusterScope !== "—" && (
          <Text>
            <Text bold>Cluster:</Text> {clusterScope}
          </Text>
        )}
        {namespaceScope && namespaceScope !== "—" && (
          <Text>
            <Text bold>Namespace:</Text> {namespaceScope}
          </Text>
        )}
        {projectScope && projectScope !== "—" && (
          <Text>
            <Text bold>Project:</Text> {projectScope}
          </Text>
        )}
        {!isNarrow && apiVersion && (
          <Text>
            <Text bold>ArgoCD:</Text> <Text color="green">{apiVersion}</Text>
          </Text>
        )}
      </>
    )}
  </Box>
);

const Logo: React.FC<LogoProps> = ({
  align,
  paddingBottom,
  argonautVersion,
}) => (
  <Box
    flexDirection="column"
    paddingBottom={paddingBottom || 0}
    alignItems={align}
  >
    <Text>
      {chalk.cyan("     _____")}
      &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
      {chalk.whiteBright(" __   ")}
    </Text>
    <Text>
      {chalk.cyan("  /  _  \\_______  ____   ____") +
        chalk.whiteBright("   ____ _____   __ ___/  |_ ")}
    </Text>
    <Text>
      {chalk.cyan(" /  /_\\  \\_  __ \\/ ___\\ /  _ \\ ") +
        chalk.whiteBright("/    \\\\__  \\ |  |  \\   __\\")}
    </Text>
    <Text>
      {chalk.cyan("/    |    \\  | \\/ /_/  >  <_> )  ") +
        chalk.whiteBright(" |  \\/ __ \\|  |  /|  |  ")}
    </Text>
    <Text>
      {chalk.cyan("\\____|__  /__|  \\___  / \\____/") +
        chalk.whiteBright("|___|  (____  /____/ |__|  ")}
    </Text>
    <Text>
      {chalk.cyan("        \\/     /_____/             ") +
        chalk.whiteBright(
          `\\/     \\/${chalk.dim((argonautVersion ?? "").padStart(13))}`,
        )}
    </Text>
  </Box>
);

const ArgoNautBanner: React.FC<ArgoNautBannerProps> = ({
  server,
  clusterScope,
  namespaceScope,
  projectScope,
  termCols = 80,
  apiVersion,
  argonautVersion,
}) => {
  const isNarrow = termCols <= 100; // stack vertically

  // Text-only for tiny terminals
  if (isNarrow) {
    return (
      <Box flexDirection="column" paddingTop={1}>
        <Box>
          <Text backgroundColor="cyan" color="white" bold>
            {" "}
            Argonaut {argonautVersion && `${argonautVersion}`}
          </Text>
        </Box>
        <Context
          paddingBottom={1}
          paddingTop={1}
          server={server}
          clusterScope={clusterScope}
          namespaceScope={namespaceScope}
          projectScope={projectScope}
          apiVersion={apiVersion}
          isNarrow={isNarrow}
        />
      </Box>
    );
  }

  // Wide: side-by-side, bottom-aligned
  return (
    <Box justifyContent="space-between" alignItems="flex-end">
      <Context
        paddingBottom={0}
        paddingTop={0}
        server={server}
        clusterScope={clusterScope}
        namespaceScope={namespaceScope}
        projectScope={projectScope}
        apiVersion={apiVersion}
        isNarrow={isNarrow}
      />
      <Logo
        paddingBottom={0}
        align="flex-end"
        argonautVersion={argonautVersion}
      />
    </Box>
  );
};

export default ArgoNautBanner;
