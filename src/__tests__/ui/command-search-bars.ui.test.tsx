import { beforeEach, describe, expect, it, mock } from "bun:test";
import { render } from "ink-testing-library";
import React, { useEffect } from "react";
import { NavigationCommand } from "../../commands/navigation";
import { CommandRegistry } from "../../commands/registry";
import { CommandBar } from "../../components/views/CommandBar";
import { SearchBar } from "../../components/views/SearchBar";
import { AppStateProvider, useAppState } from "../../contexts/AppStateContext";
import { createMockCommand, stripAnsi } from "../test-utils";

// Test CommandBar and SearchBar components
describe("CommandBar and SearchBar UI Tests", () => {
  let mockCommandRegistry: {
    parseCommandLine: ReturnType<typeof mock>;
    getCommands: ReturnType<typeof mock>;
    getAllCommands: ReturnType<typeof mock>;
    getCommand: ReturnType<typeof mock>;
    executeCommand: ReturnType<typeof mock>;
    registerCommand: ReturnType<typeof mock>;
    registerInputHandler: ReturnType<typeof mock>;
  };

  let mockOnExecuteCommand: ReturnType<typeof mock>;
  let mockOnSubmit: ReturnType<typeof mock>;

  beforeEach(() => {
    mockCommandRegistry = {
      parseCommandLine: mock().mockReturnValue({ command: "", args: [] }),
      getCommands: mock().mockReturnValue([]),
      getAllCommands: mock().mockReturnValue(new Map([["cluster", {}]])),
      getCommand: mock(),
      executeCommand: mock(),
      registerCommand: mock(),
      registerInputHandler: mock(),
    };

    mockOnExecuteCommand = mock();
    mockOnSubmit = mock();
  });

  describe("CommandBar Component", () => {
    describe("Visibility States", () => {
      it("renders when mode is 'command'", () => {
        const commandModeState = {
          mode: "command" as const,
          ui: {
            command: "",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={commandModeState}>
            <CommandBar
              commandRegistry={mockCommandRegistry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        const frame = lastFrame();

        // Should show command bar elements
        expect(frame).toContain("CMD");
        expect(frame).toContain("Enter to run, Esc to cancel");
      });

      it("does not render when mode is not 'command'", () => {
        const normalModeState = {
          mode: "normal" as const,
          ui: {
            command: "sync",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={normalModeState}>
            <CommandBar
              commandRegistry={mockCommandRegistry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        const frame = lastFrame();

        // Should be empty/null when not in command mode
        expect(frame).toBe("");
      });

      it("does not render in other modes", () => {
        const searchModeState = {
          mode: "search" as const,
          ui: {
            command: "sync",
            searchQuery: "test",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={searchModeState}>
            <CommandBar
              commandRegistry={mockCommandRegistry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        const frame = lastFrame();
        expect(frame).toBe("");
      });

      it("renders correctly after switching to command mode", async () => {
        const initialState = {
          mode: "normal" as const,
          ui: {
            command: "",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const ModeSwitcher = () => {
          const { dispatch } = useAppState();
          useEffect(() => {
            dispatch({ type: "SET_MODE", payload: "command" });
          }, [dispatch]);
          return null;
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={initialState}>
            <CommandBar
              commandRegistry={mockCommandRegistry}
              onExecuteCommand={mockOnExecuteCommand}
            />
            <ModeSwitcher />
          </AppStateProvider>,
        );

        await new Promise((resolve) => setTimeout(resolve, 0));

        const frame = lastFrame();
        expect(frame).toContain("CMD");
      });
    });

    describe("Command Input Display", () => {
      it("displays current command value", () => {
        const commandState = {
          mode: "command" as const,
          ui: {
            command: "sync frontend-app",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={commandState}>
            <CommandBar
              commandRegistry={mockCommandRegistry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        const frame = lastFrame();

        // Should show the command text
        expect(frame).toContain("CMD");
        expect(frame).toContain("sync frontend-app");
      });

      it("displays empty command correctly", () => {
        const emptyCommandState = {
          mode: "command" as const,
          ui: {
            command: "",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={emptyCommandState}>
            <CommandBar
              commandRegistry={mockCommandRegistry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        const frame = lastFrame();

        expect(frame).toContain("CMD");
        expect(frame).toContain("Enter to run, Esc to cancel");
        expect(frame).toContain(":");
      });

      it("swallows extra leading colon input", async () => {
        const state = {
          mode: "command" as const,
          ui: {
            command: "",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { stdin, lastFrame } = render(
          <AppStateProvider initialState={state}>
            <CommandBar
              commandRegistry={mockCommandRegistry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        stdin.write(":");
        await new Promise((resolve) => setTimeout(resolve, 0));

        const frame = stripAnsi(lastFrame());
        const colonCount = (frame.match(/:/g) ?? []).length;
        expect(colonCount).toBe(1);
        expect(frame).toContain("Enter to run, Esc to cancel");
      });
    });

    describe("Command Execution", () => {
      it("executes valid command on submit", () => {
        mockCommandRegistry.parseCommandLine.mockReturnValue({
          command: "sync",
          args: ["frontend-app"],
        });
        mockCommandRegistry.getCommand.mockReturnValue({});

        const commandState = {
          mode: "command" as const,
          ui: {
            command: "sync frontend-app",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { stdin } = render(
          <AppStateProvider initialState={commandState}>
            <CommandBar
              commandRegistry={mockCommandRegistry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        // Simulate Enter key press
        stdin.write("\r");

        // Should parse the command
        expect(mockCommandRegistry.parseCommandLine).toHaveBeenCalledWith(
          ":sync frontend-app",
        );

        // Should execute the command
        expect(mockOnExecuteCommand).toHaveBeenCalledWith(
          "sync",
          "frontend-app",
        );
      });

      it("handles empty command submission", () => {
        mockCommandRegistry.parseCommandLine.mockReturnValue({
          command: "",
          args: [],
        });

        const commandState = {
          mode: "command" as const,
          ui: {
            command: "",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { stdin } = render(
          <AppStateProvider initialState={commandState}>
            <CommandBar
              commandRegistry={mockCommandRegistry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        // Simulate Enter key press
        stdin.write("\r");

        // Should parse the command
        expect(mockCommandRegistry.parseCommandLine).toHaveBeenCalledWith(":");

        // Should NOT execute any command
        expect(mockOnExecuteCommand).not.toHaveBeenCalled();
      });

      it("handles complex commands with multiple arguments", () => {
        mockCommandRegistry.parseCommandLine.mockReturnValue({
          command: "rollback",
          args: ["myapp", "v1.2.3", "--force"],
        });
        mockCommandRegistry.getCommand.mockReturnValue({});

        const commandState = {
          mode: "command" as const,
          ui: {
            command: "rollback myapp v1.2.3 --force",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { stdin } = render(
          <AppStateProvider initialState={commandState}>
            <CommandBar
              commandRegistry={mockCommandRegistry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        // Simulate Enter key press
        stdin.write("\r");

        expect(mockCommandRegistry.parseCommandLine).toHaveBeenCalledWith(
          ":rollback myapp v1.2.3 --force",
        );
        expect(mockOnExecuteCommand).toHaveBeenCalledWith(
          "rollback",
          "myapp",
          "v1.2.3",
          "--force",
        );
      });

      it("shows error for unknown command", async () => {
        mockCommandRegistry.parseCommandLine.mockReturnValue({
          command: "nosuch",
          args: [],
        });
        mockCommandRegistry.getCommand.mockReturnValue(undefined);

        const commandState = {
          mode: "command" as const,
          ui: {
            command: "nosuch",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { stdin, lastFrame } = render(
          <AppStateProvider initialState={commandState}>
            <CommandBar
              commandRegistry={mockCommandRegistry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        stdin.write("\r");
        await new Promise((resolve) => setTimeout(resolve, 0));

        expect(mockOnExecuteCommand).not.toHaveBeenCalled();
        const frame = stripAnsi(lastFrame());
        expect(frame).toContain("Unknown command");
        expect(frame).toContain(":nosuch  Unknown command");
        expect(frame).not.toContain("Unknown command:");
        expect(frame).not.toContain("(Enter to run, Esc to cancel)");
      });
    });

    describe("Autocomplete", () => {
      it("shows suggestion for partial input", () => {
        const commandState = {
          mode: "command" as const,
          ui: {
            command: "cluster pro",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
          apps: [
            {
              name: "app1",
              sync: "Synced",
              health: "Healthy",
              clusterLabel: "production",
              namespace: "default",
              project: "proj1",
            },
          ],
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={commandState}>
            <CommandBar
              commandRegistry={mockCommandRegistry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        const frame = stripAnsi(lastFrame());
        expect(frame).toContain("cluster production");
      });

      it("applies suggestion on submit", () => {
        mockCommandRegistry.parseCommandLine.mockReturnValue({
          command: "cluster",
          args: ["production"],
        });
        mockCommandRegistry.getCommand.mockReturnValue({});

        const commandState = {
          mode: "command" as const,
          ui: {
            command: "cluster pro",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
          apps: [
            {
              name: "app1",
              sync: "Synced",
              health: "Healthy",
              clusterLabel: "production",
              namespace: "default",
              project: "proj1",
            },
          ],
        };

        const { stdin } = render(
          <AppStateProvider initialState={commandState}>
            <CommandBar
              commandRegistry={mockCommandRegistry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        stdin.write("\r");

        expect(mockCommandRegistry.parseCommandLine).toHaveBeenCalledWith(
          ":cluster production",
        );
        expect(mockOnExecuteCommand).toHaveBeenCalledWith(
          "cluster",
          "production",
        );
      });

      it("completes command names on submit", () => {
        mockCommandRegistry.parseCommandLine.mockReturnValue({
          command: "cluster",
          args: [],
        });
        mockCommandRegistry.getCommand.mockReturnValue({});

        const commandState = {
          mode: "command" as const,
          ui: {
            command: "clu",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { stdin } = render(
          <AppStateProvider initialState={commandState}>
            <CommandBar
              commandRegistry={mockCommandRegistry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        stdin.write("\r");

        expect(mockCommandRegistry.parseCommandLine).toHaveBeenCalledWith(
          ":cluster",
        );
      });

      it("allows deleting autocompleted text", () => {
        const commandState = {
          mode: "command" as const,
          ui: {
            command: "cluster production",
            commandInputKey: 1,
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { stdin, lastFrame } = render(
          <AppStateProvider initialState={commandState}>
            <CommandBar
              commandRegistry={mockCommandRegistry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        stdin.write("\u0008");

        const frame = stripAnsi(lastFrame());
        expect(frame).toContain(":cluster productio");
      });
    });

    describe("Command hints", () => {
      it("shows description for known commands", () => {
        const registry = new CommandRegistry();
        const clustersCmd = createMockCommand({
          description: "Switch to clusters view",
          aliases: ["clusters"],
        });
        registry.registerCommand("cluster", clustersCmd);

        const commandState = {
          mode: "command" as const,
          ui: {
            command: "clu",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={commandState}>
            <CommandBar
              commandRegistry={registry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        const frame = stripAnsi(lastFrame());
        expect(frame).toContain("(Switch to clusters view)");
        expect(frame).not.toContain("Enter to run, Esc to cancel");
      });

      it("shows unknown command hint for invalid commands", () => {
        const registry = new CommandRegistry();

        const commandState = {
          mode: "command" as const,
          ui: {
            command: "asdfasd",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={commandState}>
            <CommandBar
              commandRegistry={registry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        const frame = stripAnsi(lastFrame());
        expect(frame).toContain("(Unknown command)");
        expect(frame).not.toContain("Enter to run, Esc to cancel");
      });

      it("shows scoped hint for cluster command with entity", () => {
        const registry = new CommandRegistry();
        registry.registerCommand(
          "cluster",
          new NavigationCommand("clusters", "cluster"),
        );

        const commandState = {
          mode: "command" as const,
          ui: {
            command: "cluster enigma-us",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={commandState}>
            <CommandBar
              commandRegistry={registry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        const frame = stripAnsi(lastFrame());
        expect(frame).toContain("(display namespaces in enigma-us cluster)");
        expect(frame).not.toContain("Switch to clusters view");
      });

      it("shows scoped hint for namespace command with entity", () => {
        const registry = new CommandRegistry();
        registry.registerCommand(
          "namespace",
          new NavigationCommand("namespaces", "namespace"),
        );

        const commandState = {
          mode: "command" as const,
          ui: {
            command: "namespace api",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={commandState}>
            <CommandBar
              commandRegistry={registry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        const frame = stripAnsi(lastFrame());
        expect(frame).toContain("(display projects in api namespace)");
        expect(frame).not.toContain("Switch to namespaces view");
      });

      it("shows scoped hint for project command with entity", () => {
        const registry = new CommandRegistry();
        registry.registerCommand(
          "project",
          new NavigationCommand("projects", "project"),
        );

        const commandState = {
          mode: "command" as const,
          ui: {
            command: "project payments",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={commandState}>
            <CommandBar
              commandRegistry={registry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        const frame = stripAnsi(lastFrame());
        expect(frame).toContain("(display apps in payments project)");
        expect(frame).not.toContain("Switch to projects view");
      });

      it("shows scoped hint for app command with entity", () => {
        const registry = new CommandRegistry();
        registry.registerCommand("app", new NavigationCommand("apps", "app"));

        const commandState = {
          mode: "command" as const,
          ui: {
            command: "app frontend",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={commandState}>
            <CommandBar
              commandRegistry={registry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        const frame = stripAnsi(lastFrame());
        expect(frame).toContain("(go to app frontend)");
        expect(frame).not.toContain("Switch to apps view");
      });

      it("supports aliases for scoped hints", () => {
        const registry = new CommandRegistry();
        registry.registerCommand(
          "cluster",
          new NavigationCommand("clusters", "cluster", ["clusters", "cls"]),
        );

        const commandState = {
          mode: "command" as const,
          ui: {
            command: "cls enigma-us",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={commandState}>
            <CommandBar
              commandRegistry={registry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        const frame = stripAnsi(lastFrame());
        expect(frame).toContain("(display namespaces in enigma-us cluster)");
      });
    });

    describe("Autocomplete", () => {
      it("completes command names on Tab", async () => {
        const registry = new CommandRegistry();
        registry.registerCommand(
          "cluster",
          new NavigationCommand("clusters", "cluster"),
        );

        const state = {
          mode: "command" as const,
          ui: {
            command: "",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { stdin, lastFrame } = render(
          <AppStateProvider initialState={state}>
            <CommandBar
              commandRegistry={registry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        stdin.write("clu");
        await new Promise((r) => setTimeout(r, 0));
        stdin.write("\t");
        await new Promise((r) => setTimeout(r, 0));

        const frame = stripAnsi(lastFrame());
        expect(frame).toContain("cluster");
      });

      it("completes cluster names on Tab", async () => {
        const registry = new CommandRegistry();
        registry.registerCommand(
          "cluster",
          new NavigationCommand("clusters", "cluster"),
        );

        const state = {
          mode: "command" as const,
          apps: [
            {
              name: "a",
              sync: "Synced",
              health: "Healthy",
              clusterLabel: "production",
              namespace: "default",
              project: "proj1",
            },
          ],
          ui: {
            command: "",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { stdin, lastFrame } = render(
          <AppStateProvider initialState={state}>
            <CommandBar
              commandRegistry={registry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        stdin.write("cluster pro");
        await new Promise((r) => setTimeout(r, 0));
        stdin.write("\t");
        await new Promise((r) => setTimeout(r, 0));

        const frame = stripAnsi(lastFrame());
        expect(frame).toContain("cluster production");
      });
    });

    describe("UI Styling and Layout", () => {
      it("displays with proper styling elements", () => {
        const commandState = {
          mode: "command" as const,
          ui: {
            command: "help",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={commandState}>
            <CommandBar
              commandRegistry={mockCommandRegistry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        const frame = lastFrame();

        // Should have CMD label
        expect(frame).toContain("CMD");

        // Should show hint about command
        expect(frame).toContain("(Unknown command)");

        // Should contain the command
        expect(frame).toContain("help");
      });
    });
  });

  describe("SearchBar Component", () => {
    describe("Visibility States", () => {
      it("renders when mode is 'search'", () => {
        const searchModeState = {
          mode: "search" as const,
          navigation: {
            view: "apps" as const,
            selectedIdx: 0,
            lastGPressed: 0,
          },
          ui: {
            searchQuery: "frontend",
            command: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={searchModeState}>
            <SearchBar onSubmit={mockOnSubmit} />
          </AppStateProvider>,
        );

        const frame = lastFrame();

        // Should show search bar elements
        expect(frame).toContain("Search");
        expect(frame).toContain("Enter");
        expect(frame).toContain("Esc cancels");
      });

      it("does not render when mode is not 'search'", () => {
        const normalModeState = {
          mode: "normal" as const,
          navigation: {
            view: "apps" as const,
            selectedIdx: 0,
            lastGPressed: 0,
          },
          ui: {
            searchQuery: "frontend",
            command: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={normalModeState}>
            <SearchBar onSubmit={mockOnSubmit} />
          </AppStateProvider>,
        );

        const frame = lastFrame();
        expect(frame).toBe("");
      });
    });

    describe("Search Query Display", () => {
      it("displays current search query", () => {
        const searchState = {
          mode: "search" as const,
          navigation: {
            view: "apps" as const,
            selectedIdx: 0,
            lastGPressed: 0,
          },
          ui: {
            searchQuery: "frontend-web",
            command: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={searchState}>
            <SearchBar onSubmit={mockOnSubmit} />
          </AppStateProvider>,
        );

        const frame = lastFrame();

        expect(frame).toContain("Search");
        expect(frame).toContain("frontend-web");
      });

      it("displays empty search query", () => {
        const emptySearchState = {
          mode: "search" as const,
          navigation: {
            view: "clusters" as const,
            selectedIdx: 0,
            lastGPressed: 0,
          },
          ui: {
            searchQuery: "",
            command: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={emptySearchState}>
            <SearchBar onSubmit={mockOnSubmit} />
          </AppStateProvider>,
        );

        const frame = lastFrame();
        expect(frame).toContain("Search");
      });
    });

    describe("Context-Aware Help Text", () => {
      it("shows 'keeps filter' help text for apps view", () => {
        const appsSearchState = {
          mode: "search" as const,
          navigation: {
            view: "apps" as const,
            selectedIdx: 0,
            lastGPressed: 0,
          },
          ui: {
            searchQuery: "api",
            command: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={appsSearchState}>
            <SearchBar onSubmit={mockOnSubmit} />
          </AppStateProvider>,
        );

        const frame = lastFrame();

        expect(frame).toContain("Enter keeps filter");
        expect(frame).toContain("Esc cancels");
      });

      it("shows 'opens first result' help text for non-apps views", () => {
        const clustersSearchState = {
          mode: "search" as const,
          navigation: {
            view: "clusters" as const,
            selectedIdx: 0,
            lastGPressed: 0,
          },
          ui: {
            searchQuery: "prod",
            command: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={clustersSearchState}>
            <SearchBar onSubmit={mockOnSubmit} />
          </AppStateProvider>,
        );

        const frame = lastFrame();

        expect(frame).toContain("Enter opens first result");
        expect(frame).toContain("Esc cancels");
      });

      it("shows correct help text for different views", () => {
        const namespacesSearchState = {
          mode: "search" as const,
          navigation: {
            view: "namespaces" as const,
            selectedIdx: 0,
            lastGPressed: 0,
          },
          ui: {
            searchQuery: "frontend",
            command: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={namespacesSearchState}>
            <SearchBar onSubmit={mockOnSubmit} />
          </AppStateProvider>,
        );

        const frame = lastFrame();

        expect(frame).toContain("Enter opens first result");
        expect(frame).toContain("Esc cancels");
      });
    });

    describe("Search Submission Behavior", () => {
      it("calls onSubmit for non-apps views", () => {
        const clustersSearchState = {
          mode: "search" as const,
          navigation: {
            view: "clusters" as const,
            selectedIdx: 0,
            lastGPressed: 0,
          },
          ui: {
            searchQuery: "production",
            command: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { stdin } = render(
          <AppStateProvider initialState={clustersSearchState}>
            <SearchBar onSubmit={mockOnSubmit} />
          </AppStateProvider>,
        );

        // Simulate Enter key press
        stdin.write("\r");

        // Should call onSubmit for drill-down
        expect(mockOnSubmit).toHaveBeenCalled();
      });

      it("sets active filter for apps view without calling onSubmit", () => {
        const appsSearchState = {
          mode: "search" as const,
          navigation: {
            view: "apps" as const,
            selectedIdx: 0,
            lastGPressed: 0,
          },
          ui: {
            searchQuery: "frontend",
            command: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { stdin } = render(
          <AppStateProvider initialState={appsSearchState}>
            <SearchBar onSubmit={mockOnSubmit} />
          </AppStateProvider>,
        );

        // Simulate Enter key press
        stdin.write("\r");

        // Should NOT call onSubmit for apps view (filter is set instead)
        expect(mockOnSubmit).not.toHaveBeenCalled();
      });
    });

    describe("UI Styling and Layout", () => {
      it("displays with proper styling elements", () => {
        const searchState = {
          mode: "search" as const,
          navigation: {
            view: "projects" as const,
            selectedIdx: 0,
            lastGPressed: 0,
          },
          ui: {
            searchQuery: "web-platform",
            command: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={searchState}>
            <SearchBar onSubmit={mockOnSubmit} />
          </AppStateProvider>,
        );

        const frame = lastFrame();

        // Should have Search label
        expect(frame).toContain("Search");

        // Should have help text
        expect(frame).toContain("Enter opens first result");
        expect(frame).toContain("Esc cancels");

        // Should contain the search query
        expect(frame).toContain("web-platform");
      });
    });
  });

  describe("Input Interaction Tests", () => {
    describe("CommandBar Input", () => {
      it("handles typing in command input", () => {
        const commandState = {
          mode: "command" as const,
          ui: {
            command: "",
            searchQuery: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={commandState}>
            <CommandBar
              commandRegistry={mockCommandRegistry}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );

        // Should start with just ":"
        expect(lastFrame()).toContain("CMD");

        // Note: TextInput component handles internal state,
        // but we can test that the component renders properly
        expect(lastFrame()).toBeDefined();
        expect(lastFrame()).toContain("Enter to run");
      });
    });

    describe("SearchBar Input", () => {
      it("handles typing in search input", () => {
        const searchState = {
          mode: "search" as const,
          navigation: {
            view: "apps" as const,
            selectedIdx: 0,
            lastGPressed: 0,
          },
          ui: {
            searchQuery: "",
            command: "",
            activeFilter: "",
            isVersionOutdated: false,
          },
        };

        const { lastFrame } = render(
          <AppStateProvider initialState={searchState}>
            <SearchBar onSubmit={mockOnSubmit} />
          </AppStateProvider>,
        );

        // Should render search bar
        expect(lastFrame()).toContain("Search");
        expect(lastFrame()).toContain("keeps filter");
      });
    });
  });

  describe("Edge Cases and Error Handling", () => {
    it("handles CommandBar with null command registry gracefully", () => {
      const commandState = {
        mode: "command" as const,
        ui: {
          command: "help",
          searchQuery: "",
          activeFilter: "",
          isVersionOutdated: false,
        },
      };

      // This might cause issues in real code, but we test it doesn't crash
      expect(() => {
        render(
          <AppStateProvider initialState={commandState}>
            <CommandBar
              commandRegistry={null as any}
              onExecuteCommand={mockOnExecuteCommand}
            />
          </AppStateProvider>,
        );
      }).not.toThrow();
    });

    it("handles SearchBar with missing navigation gracefully", () => {
      const searchState = {
        mode: "search" as const,
        // Missing navigation property
        ui: {
          searchQuery: "test",
          command: "",
          activeFilter: "",
          isVersionOutdated: false,
        },
      };

      // Should handle gracefully
      expect(() => {
        render(
          <AppStateProvider initialState={searchState}>
            <SearchBar onSubmit={mockOnSubmit} />
          </AppStateProvider>,
        );
      }).not.toThrow();
    });

    it("handles very long command input", () => {
      const longCommand = `:sync ${"very-long-app-name-".repeat(10)}`;
      const commandState = {
        mode: "command" as const,
        ui: {
          command: longCommand,
          searchQuery: "",
          activeFilter: "",
          isVersionOutdated: false,
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={commandState}>
          <CommandBar
            commandRegistry={mockCommandRegistry}
            onExecuteCommand={mockOnExecuteCommand}
          />
        </AppStateProvider>,
      );

      // Should render without issues (text might be wrapped due to length)
      expect(lastFrame()).toBeDefined();
      expect(lastFrame()).toMatch(/CM[D]?/); // Might be wrapped as "CM" + "D"
    });

    it("handles very long search query", () => {
      const longQuery = "frontend-web-application-with-very-long-name-".repeat(
        5,
      );
      const searchState = {
        mode: "search" as const,
        navigation: {
          view: "apps" as const,
          selectedIdx: 0,
          lastGPressed: 0,
        },
        ui: {
          searchQuery: longQuery,
          command: "",
          activeFilter: "",
          isVersionOutdated: false,
        },
      };

      const { lastFrame } = render(
        <AppStateProvider initialState={searchState}>
          <SearchBar onSubmit={mockOnSubmit} />
        </AppStateProvider>,
      );

      // Should render without issues (text might be wrapped due to length)
      expect(lastFrame()).toBeDefined();
      expect(lastFrame()).toMatch(/Sear/); // Might be wrapped, so just check for beginning
    });
  });
});
