import React from 'react';
import {Box, Text} from 'ink';

export type HelpProps = {
  version: string;
  isOutdated?: boolean;
  latestVersion?: string;
};

const Help: React.FC<HelpProps> = ({version, isOutdated, latestVersion}) => (
  <Box flexDirection="column" paddingX={2} paddingY={1}>
    <Box justifyContent="center">
      <Text color="magentaBright" bold>
        Argonaut {version}
        {isOutdated && latestVersion && (
          <Text color="yellow"> (latest: {latestVersion})</Text>
        )}
      </Text>
    </Box>
    <Box marginTop={1}>
      <Box width={24}>
        <Text color="green" bold>
          GENERAL
        </Text>
      </Box>
      <Box>
        <Text>
          <Text color="cyan">:</Text> command • <Text color="cyan">/</Text> search • <Text color="cyan">?</Text> help
        </Text>
      </Box>
    </Box>
    <Box marginTop={1}>
      <Box width={24}>
        <Text color="green" bold>
          NAV
        </Text>
      </Box>
      <Box>
        <Text>
          <Text color="cyan">j/k</Text> up/down • <Text color="cyan">Space</Text> select • <Text color="cyan">Enter</Text> drill down
        </Text>
      </Box>
    </Box>
    <Box marginTop={1}>
      <Box width={24}>
        <Text color="green" bold>
          VIEWS
        </Text>
      </Box>
      <Box>
        <Text>
          :cls|:clusters|:cluster • :ns|:namespaces|:namespace • :proj|:projects|:project • :apps
        </Text>
      </Box>
    </Box>
    <Box marginTop={1}>
      <Box width={24}>
        <Text color="green" bold>
          ACTIONS
        </Text>
      </Box>
      <Box>
        <Text>
          :sync [app] • :rollback [app]
        </Text>
      </Box>
    </Box>
    <Box marginTop={1}>
      <Box width={24}>
        <Text color="green" bold>
          MISC
        </Text>
      </Box>
      <Box>
        <Text>
          :login • :clear • :all • :q
        </Text>
      </Box>
    </Box>
    <Box marginTop={1}>
      <Text dimColor>Press ? or Esc to close</Text>
    </Box>
  </Box>
);

export default Help;
