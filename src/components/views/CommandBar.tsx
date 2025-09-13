import { Box, Text, useInput } from "ink";
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
  const [input, setInput] = useState(state.ui.command);
  const [error, setError] = useState<string | null>(null);
  useInput(
    (_, key) => {
      if (key.escape) {
        dispatch({ type: "SET_MODE", payload: "normal" });
        dispatch({ type: "SET_COMMAND", payload: "" });
        setInput("");
        setError(null);
      }

      if (key.tab) {
        const autoComplete = getCommandAutocomplete(
          `:${input}`,
          state,
          commandRegistry,
        );
        if (autoComplete) {
          setInput(autoComplete.completed.slice(1));
        }
      }
    },
    { isActive: state.mode === "command" },
  );

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
    setInput("");
    setError(null);
  };

  const userCommand = input.replace(/^:+/, "");
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

  const fullWidth = Math.max(0, (state.terminal?.cols ?? 0) - 2);

  return (
    <Box
      borderStyle="round"
      borderColor={error ? "red" : "yellow"}
      paddingX={1}
      width={fullWidth || undefined}
    >
      <Text bold color="cyan">
        CMD
      </Text>
      <Box width={1} />
      <Text color="white">:</Text>
      <TextInput
        value={input}
        onChange={(value) => {
          const sanitized = value.replace(/^:+/, "");
          setInput(sanitized);
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
