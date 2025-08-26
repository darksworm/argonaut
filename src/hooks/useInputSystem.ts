import { useInput } from "ink";
import type { CommandRegistry } from "../commands/registry";
import type { CommandContext, StatusLogger } from "../commands/types";
import { useAppState } from "../contexts/AppStateContext";

export const useInputSystem = (
  registry: CommandRegistry,
  cleanupAndExit: () => void,
  statusLog: StatusLogger,
  navigationActions?: {
    drillDown: () => void;
    toggleSelection: () => void;
  },
) => {
  const { state, dispatch } = useAppState();

  // Create a mutable command context that can be passed to handlers
  const context: CommandContext = {
    state,
    dispatch,
    statusLog,
    cleanupAndExit,
    navigationActions,
    // This will be overwritten below to avoid circular dependency issues
    executeCommand: async () => {},
  };

  // Command execution helper
  const executeCommand = async (command: string, ...args: string[]) => {
    const success = await registry.executeCommand(command, context, ...args);
    if (!success) {
      statusLog.warn(`Unknown command: ${command}`, "command");
    }
  };

  // Now that executeCommand is defined, attach it to the context
  context.executeCommand = executeCommand;

  // Handle keyboard input
  useInput((input, key) => {
    // Let the registry handle all input
    const handled = registry.handleInput(input, key, context);

    if (!handled) {
      // If no handler processed the input, we could log it or handle it as unknown
      // For now, we just ignore unhandled input
    }
  });

  return {
    executeCommand,
    context,
  };
};
