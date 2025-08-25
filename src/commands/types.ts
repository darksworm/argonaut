import type { AppAction, AppState } from "../contexts/AppStateContext";

export interface StatusLogger {
  info: (message: string, context?: string) => void;
  warn: (message: string, context?: string) => void;
  error: (message: string, context?: string) => void;
  debug: (message: string, context?: string) => void;
  set: (message: string) => void;
}

export interface CommandContext {
  state: AppState;
  dispatch: React.Dispatch<AppAction>;
  statusLog: StatusLogger;
  cleanupAndExit: () => void;
  navigationActions?: {
    drillDown: () => void;
    toggleSelection: () => void;
  };
  executeCommand: (command: string, ...args: string[]) => Promise<void>;
}

export interface Command {
  execute(context: CommandContext, ...args: string[]): void | Promise<void>;
  canExecute?(context: CommandContext): boolean;
  description?: string;
  aliases?: string[];
}

export interface InputHandler {
  handleInput(input: string, key: any, context: CommandContext): boolean;
  canHandle(context: CommandContext): boolean;
  priority?: number; // Higher priority handlers are checked first
}

export type KeyBinding = {
  key: string;
  command: string;
  args?: string[];
  modes?: string[];
};
