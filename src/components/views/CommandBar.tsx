import { Box, Text } from "ink";
import TextInput from "ink-text-input";
import type React from "react";
import { useState } from "react";
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
  const [error, setError] = useState<string | null>(null);

  if (state.mode !== "command") {
    return null;
  }

  const handleSubmit = (val: string) => {
    const line = `:${val}`;
    const auto = getCommandAutocomplete(line, state, commandRegistry);
    const completed = auto ? auto.completed : line;

    const { command, args } = commandRegistry.parseCommandLine(completed);

    if (!command) {
      dispatch({ type: "SET_MODE", payload: "normal" });
      dispatch({ type: "SET_COMMAND", payload: "" });
      setError(null);
      return;
    }

    if (!commandRegistry.getCommand(command)) {
      setError("Unknown command");
      process.stdout.write("\x07");
      return;
    }

    dispatch({ type: "SET_MODE", payload: "normal" });
    onExecuteCommand(command, ...args);
    dispatch({ type: "SET_COMMAND", payload: "" });
    setError(null);
  };

  const auto = getCommandAutocomplete(
    `:${state.ui.command}`,
    state,
    commandRegistry,
  );

  const hintText = (() => {
    if (!state.ui.command) {
      return "(Enter to run, Esc to cancel)";
    }
    const completedLine = auto ? auto.completed : `:${state.ui.command}`;
    const parsed = commandRegistry.parseCommandLine(completedLine);
    const command = parsed?.command ?? "";
    if (!command) {
      return "(Unknown command)";
    }
    const cmd = commandRegistry.getCommand(command);
    return `(${cmd?.description ?? "Unknown command"})`;
  })();

  return (
    <Box
      borderStyle="round"
      borderColor={error ? "red" : "yellow"}
      paddingX={1}
    >
      <Text bold color="cyan">
        CMD
      </Text>
      <Box width={1} />
      <Text color="white">:</Text>
      <TextInput
        key={state.ui.commandInputKey}
        value={state.ui.command}
        onChange={(value) => {
          dispatch({
            type: "SET_COMMAND",
            payload: value,
          });
          if (error) {
            setError(null);
          }
        }}
        onSubmit={handleSubmit}
        showCursor={false}
      />
      {!error && auto ? <Text dimColor>{auto.suggestion}</Text> : null}
      <Box width={2} />
      {error ? (
        <Text color="red">{error}</Text>
      ) : (
        <Text dimColor>{hintText}</Text>
      )}
    </Box>
  );
};
