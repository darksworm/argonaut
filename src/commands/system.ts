import { rulerLineMode } from "../components/OfficeSupplyManager";
import type { Command, CommandContext } from "./types";

export class QuitCommand implements Command {
  aliases = ["quit", "exit", "q!"];
  description = "Exit the application";

  execute(context: CommandContext): void {
    context.cleanupAndExit();
  }
}

export class HelpCommand implements Command {
  aliases = ["?"];
  description = "Show help";

  execute(context: CommandContext): void {
    context.dispatch({ type: "SET_MODE", payload: "help" });
  }
}

export class RulerCommand implements Command {
  aliases = [];
  description = "Open ruler line mode";

  execute(context: CommandContext): void {
    context.dispatch({ type: "SET_MODE", payload: "rulerline" });
  }
}

// Register ruler command by its special name
export const createRulerCommand = () => {
  const command = new RulerCommand();
  return { name: rulerLineMode, command };
};
