import type { Command, CommandContext, InputHandler } from "./types";

export class CommandRegistry {
  private commands = new Map<string, Command>();
  private inputHandlers: InputHandler[] = [];

  registerCommand(name: string, command: Command) {
    this.commands.set(name.toLowerCase(), command);

    // Register aliases
    if (command.aliases) {
      for (const alias of command.aliases) {
        this.commands.set(alias.toLowerCase(), command);
      }
    }
  }

  registerInputHandler(handler: InputHandler) {
    this.inputHandlers.push(handler);
    // Sort by priority (higher first)
    this.inputHandlers.sort((a, b) => (b.priority || 0) - (a.priority || 0));
  }

  async executeCommand(
    name: string,
    context: CommandContext,
    ...args: string[]
  ): Promise<boolean> {
    const command = this.commands.get(name.toLowerCase());
    if (!command) {
      return false;
    }

    if (command.canExecute && !command.canExecute(context)) {
      return false;
    }

    try {
      await command.execute(context, ...args);
      return true;
    } catch (error) {
      context.statusLog.error(
        `Command '${name}' failed: ${error instanceof Error ? error.message : String(error)}`,
        "command",
      );
      return false;
    }
  }

  handleInput(input: string, key: any, context: CommandContext): boolean {
    // Try input handlers in priority order
    for (const handler of this.inputHandlers) {
      if (handler.canHandle(context)) {
        if (handler.handleInput(input, key, context)) {
          return true; // Input was handled
        }
      }
    }
    return false; // Input not handled
  }

  getCommand(name: string): Command | undefined {
    return this.commands.get(name.toLowerCase());
  }

  getAllCommands(): Map<string, Command> {
    return new Map(this.commands);
  }

  parseCommandLine(line: string): { command: string; args: string[] } {
    const raw = line.trim();
    if (!raw.startsWith(":")) {
      return { command: "", args: [] };
    }

    const parts = raw.slice(1).trim().split(/\s+/);
    const command = (parts[0] || "").toLowerCase();
    const args = parts.slice(1);

    return { command, args };
  }
}
