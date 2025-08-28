import {
  DiffCommand,
  LicenseCommand,
  LogsCommand,
  ResourcesCommand,
  RollbackCommand,
  SyncCommand,
} from "./application";
import {
  CommandInputHandler,
  GlobalInputHandler,
  ModeInputHandler,
  NavigationInputHandler,
  SearchInputHandler,
} from "./handlers/keyboard";
import {
  ClearAllCommand,
  ClearCommand,
  NavigationCommand,
  UpCommand,
} from "./navigation";
import { CommandRegistry } from "./registry";
import {
  createRulerCommand,
  HelpCommand,
  LoginCommand,
  QuitCommand,
} from "./system";

export function createDefaultCommandRegistry(): CommandRegistry {
  const registry = new CommandRegistry();

  // System commands
  registry.registerCommand("q", new QuitCommand());
  registry.registerCommand("help", new HelpCommand());
  registry.registerCommand("login", new LoginCommand());

  // Special ruler command
  const { name: rulerName, command: rulerCommand } = createRulerCommand();
  registry.registerCommand(rulerName, rulerCommand);

  // Navigation commands
  registry.registerCommand(
    "cluster",
    new NavigationCommand("clusters", "cluster", ["clusters", "cls"]),
  );
  registry.registerCommand(
    "namespace",
    new NavigationCommand("namespaces", "namespace", ["namespaces", "ns"]),
  );
  registry.registerCommand(
    "project",
    new NavigationCommand("projects", "project", ["projects", "proj"]),
  );
  registry.registerCommand(
    "app",
    new NavigationCommand("apps", "app", ["apps"]),
  );

  // Clear commands
  registry.registerCommand("clear", new ClearCommand());
  registry.registerCommand("all", new ClearAllCommand());

  // Navigation commands
  registry.registerCommand("up", new UpCommand());

  // Application commands
  registry.registerCommand("sync", new SyncCommand());
  registry.registerCommand("diff", new DiffCommand());
  registry.registerCommand("rollback", new RollbackCommand());
  registry.registerCommand("resources", new ResourcesCommand());
  registry.registerCommand("logs", new LogsCommand());
  registry.registerCommand("license", new LicenseCommand());

  // Input handlers (order matters - higher priority first)
  registry.registerInputHandler(new SearchInputHandler());
  registry.registerInputHandler(new CommandInputHandler());
  registry.registerInputHandler(new ModeInputHandler());
  registry.registerInputHandler(new NavigationInputHandler());
  registry.registerInputHandler(new GlobalInputHandler());

  return registry;
}

export * from "./registry";
export * from "./types";
export { CommandRegistry };
