import { Box, Text } from "ink";
import TextInput from "ink-text-input";
import type React from "react";
import type { CommandRegistry } from "../../commands";
import { getCommandAutocomplete } from "../../commands/autocomplete";
import { useAppState } from "../../contexts/AppStateContext";

interface CommandBarProps {
  commandRegistry: CommandRegistry;
  onExecuteCommand: (command: string, ...args: string[]) => void;
}

export const CommandBar: React.FC<CommandBarProps> = ({
  commandRegistry,
  onExecuteCommand,
}) => {
  const { state, dispatch } = useAppState();

  if (state.mode !== "command") {
    return null;
  }

  const handleSubmit = (val: string) => {
    const line = `:${val}`;
    const auto = getCommandAutocomplete(line, state, commandRegistry);
    const completed = auto ? auto.completed : line;

    dispatch({ type: "SET_MODE", payload: "normal" });

    const { command, args } = commandRegistry.parseCommandLine(completed);
    if (command) {
      onExecuteCommand(command, ...args);
    }

    dispatch({ type: "SET_COMMAND", payload: "" });
  };

  return (
    <Box borderStyle="round" borderColor="yellow" paddingX={1}>
      <Text bold color="cyan">
        CMD
      </Text>
      <Box width={1} />
      <Text color="white">:</Text>
      <TextInput
        key={state.ui.commandInputKey}
        value={state.ui.command}
        onChange={(value) =>
          dispatch({
            type: "SET_COMMAND",
            payload: value,
          })
        }
        onSubmit={handleSubmit}
        showCursor={false}
      />
      {(() => {
        const auto = getCommandAutocomplete(
          `:${state.ui.command}`,
          state,
          commandRegistry,
        );
        return auto ? <Text dimColor>{auto.suggestion}</Text> : null;
      })()}
      <Box width={2} />
      <Text dimColor>(Enter to run, Esc to cancel)</Text>
    </Box>
  );
};
