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
    const sanitized = val.replace(/^:+/, "");
    const line = `:${sanitized}`;
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

  const userCommand = state.ui.command.replace(/^:+/, "");
  const auto = getCommandAutocomplete(
    `:${userCommand}`,
    state,
    commandRegistry,
  );
  const hint = (() => {
    if (!userCommand) {
      return <Text dimColor>(Enter to run, Esc to cancel)</Text>;
    }
    const completedLine = auto ? auto.completed : `:${userCommand}`;
    const parsed = commandRegistry.parseCommandLine(completedLine);
    const command = parsed?.command ?? "";
    if (!command) {
      return <Text dimColor>(Unknown command)</Text>;
    }
    const cmd = commandRegistry.getCommand(command);

    const scopeMap: Record<string, string | null> = {
      cluster: "namespaces",
      namespace: "projects",
      project: "apps",
      app: null,
    };

    const arg = parsed.args[0];
    const nextScope = scopeMap[command];
    if (arg && nextScope !== undefined) {
      return nextScope ? (
        <Text dimColor>
          (display{" "}
          <Text bold dimColor={false}>
            {nextScope}
          </Text>{" "}
          in{" "}
          <Text color="white" dimColor={false}>
            {arg}
          </Text>{" "}
          {command})
        </Text>
      ) : (
        <Text dimColor>
          (go to app{" "}
          <Text color="white" dimColor={false}>
            {arg}
          </Text>
          )
        </Text>
      );
    }

    return <Text dimColor>({cmd?.description ?? "Unknown command"})</Text>;
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
          const sanitized = value.replace(/^:+/, "");
          dispatch({
            type: "SET_COMMAND",
            payload: sanitized,
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
      {error ? <Text color="red">{error}</Text> : hint}
    </Box>
  );
};
