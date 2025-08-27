import { Box, Text, useStdout } from "ink";
import type React from "react";

const HelpSection: React.FC<{
  title: string;
  children: React.ReactNode;
  isWide: boolean;
}> = ({ title, children, isWide }) => (
  <Box marginTop={1} flexDirection={isWide ? "row" : "column"}>
    <Box width={isWide ? 12 : undefined} marginBottom={isWide ? 0 : 0}>
      <Text color="green" bold>
        {title}
      </Text>
    </Box>
    <Box flexShrink={1} flexWrap="wrap">
      {children}
    </Box>
  </Box>
);

const Help: React.FC = () => {
  const { stdout } = useStdout();
  const terminalWidth = stdout?.columns || 80;
  const isWide = terminalWidth >= 60;

  return (
    <Box flexDirection="column" paddingX={1} paddingY={1}>
      <HelpSection title="GENERAL" isWide={isWide}>
        <Text>
          <Text color="cyan">:</Text> command • <Text color="cyan">/</Text>{" "}
          search • <Text color="cyan">?</Text> help
        </Text>
      </HelpSection>

      <HelpSection title="NAV" isWide={isWide}>
        <Text>
          <Text color="cyan">j/k</Text> up/down •{" "}
          <Text color="cyan">Space</Text> select •{" "}
          <Text color="cyan">Enter</Text> drill down •{" "}
          <Text color="cyan">Esc</Text> clear/up
        </Text>
      </HelpSection>

      <HelpSection title="VIEWS" isWide={isWide}>
        <Box flexDirection="column">
          <Text>
            <Text color="cyan">:cls</Text>|<Text color="cyan">:clusters</Text>|
            <Text color="cyan">:cluster</Text> • <Text color="cyan">:ns</Text>|
            <Text color="cyan">:namespaces</Text>|
            <Text color="cyan">:namespace</Text>
          </Text>
          <Text>
            <Text color="cyan">:proj</Text>|<Text color="cyan">:projects</Text>|
            <Text color="cyan">:project</Text> • <Text color="cyan">:apps</Text>
          </Text>
        </Box>
      </HelpSection>

      <HelpSection title="ACTIONS" isWide={isWide}>
        <Box flexDirection="column">
          <Text>
            <Text color="cyan">:diff</Text> [app] •{" "}
            <Text color="cyan">:sync</Text> [app] •{" "}
            <Text color="cyan">:rollback</Text> [app]
          </Text>
          <Text>
            <Text color="cyan">:up</Text> go up level
          </Text>
        </Box>
      </HelpSection>

      <HelpSection title="MISC" isWide={isWide}>
        <Box flexDirection="column">
          <Text>
            <Text color="cyan">:login</Text> • <Text color="cyan">:clear</Text>{" "}
            • <Text color="cyan">:all</Text> •{" "}
            <Text color="cyan">:licenses</Text>
          </Text>
          <Text>
            <Text color="cyan">:logs</Text> • <Text color="cyan">:q</Text>
          </Text>
        </Box>
      </HelpSection>

      <Box marginTop={1} justifyContent="center">
        <Text dimColor>Press ?, q or Esc to close</Text>
      </Box>
    </Box>
  );
};

export default Help;
