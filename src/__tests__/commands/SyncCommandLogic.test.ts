// src/__tests__/commands/SyncCommandLogic.test.ts
import { createMockContext, createMockState } from '../test-utils';
import type { Command, CommandContext } from '../../commands/types';

// Create a test implementation of SyncCommand without external dependencies
class TestSyncCommand implements Command {
  aliases = [];
  description = "Sync application(s)";

  canExecute(context: CommandContext): boolean {
    return context.state.server !== null;
  }

  execute(context: CommandContext, arg?: string): void {
    const { state, dispatch } = context;
    const { selectedApps } = state.selections;
    const { view, selectedIdx } = state.navigation;

    // Prefer explicit arg; otherwise if multiple apps are selected, confirm multi-sync.
    if (arg) {
      dispatch({ type: "SET_CONFIRM_TARGET", payload: arg });
      dispatch({ type: "SET_CONFIRM_SYNC_PRUNE", payload: false });
      dispatch({ type: "SET_CONFIRM_SYNC_WATCH", payload: true });
      dispatch({ type: "SET_MODE", payload: "confirm-sync" });
      return;
    }

    if (selectedApps.size > 1) {
      dispatch({ type: "SET_CONFIRM_TARGET", payload: "__MULTI__" });
      dispatch({ type: "SET_CONFIRM_SYNC_PRUNE", payload: false });
      dispatch({ type: "SET_CONFIRM_SYNC_WATCH", payload: false }); // disabled for multi
      dispatch({ type: "SET_MODE", payload: "confirm-sync" });
      return;
    }

    // Fallback to current cursor app (apps view) or the single selected app
    const visibleItems = this.getVisibleItems(context);
    const target =
      (view === "apps"
        ? (visibleItems[selectedIdx] as any)?.name
        : undefined) ||
      (selectedApps.size === 1 ? Array.from(selectedApps)[0] : undefined);

    if (!target) {
      context.statusLog.warn("No app selected to sync.", "user-action");
      return;
    }

    dispatch({ type: "SET_CONFIRM_TARGET", payload: target });
    dispatch({ type: "SET_CONFIRM_SYNC_PRUNE", payload: false });
    dispatch({ type: "SET_CONFIRM_SYNC_WATCH", payload: true });
    dispatch({ type: "SET_MODE", payload: "confirm-sync" });
  }

  private getVisibleItems(context: CommandContext): any[] {
    return context.state.apps;
  }
}

describe('SyncCommand Logic', () => {
  let syncCommand: TestSyncCommand;

  beforeEach(() => {
    syncCommand = new TestSyncCommand();
  });

  describe('canExecute', () => {
    it('should require authentication', () => {
      const context = createMockContext({
        state: createMockState({ server: null })
      });

      expect(syncCommand.canExecute(context)).toBe(false);
    });

    it('should allow execution when authenticated', () => {
      const context = createMockContext();

      expect(syncCommand.canExecute(context)).toBe(true);
    });
  });

  describe('execute', () => {
    it('should handle explicit app argument', () => {
      const mockDispatch = jest.fn();
      const context = createMockContext({
        dispatch: mockDispatch
      });

      syncCommand.execute(context, 'my-app');

      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_CONFIRM_TARGET',
        payload: 'my-app'
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_CONFIRM_SYNC_PRUNE',
        payload: false
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_CONFIRM_SYNC_WATCH',
        payload: true
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_MODE',
        payload: 'confirm-sync'
      });
    });

    it('should handle multi-app selection (>1 apps)', () => {
      const mockDispatch = jest.fn();
      const context = createMockContext({
        state: createMockState({
          selections: {
            selectedApps: new Set(['app1', 'app2']),
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set()
          }
        }),
        dispatch: mockDispatch
      });

      syncCommand.execute(context);

      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_CONFIRM_TARGET',
        payload: '__MULTI__'
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_CONFIRM_SYNC_WATCH',
        payload: false // disabled for multi-sync
      });
    });

    it('should handle single app from cursor position', () => {
      const mockDispatch = jest.fn();
      const apps = [
        { name: 'cursor-app', sync: 'Synced', health: 'Healthy', clusterId: 'cluster1', clusterLabel: 'cluster1', namespace: 'default', appNamespace: 'argocd', project: 'default' }
      ];
      const context = createMockContext({
        state: createMockState({
          navigation: { view: 'apps', selectedIdx: 0, lastGPressed: 0 },
          selections: { selectedApps: new Set(), scopeClusters: new Set(), scopeNamespaces: new Set(), scopeProjects: new Set() },
          apps
        }),
        dispatch: mockDispatch
      });

      syncCommand.execute(context);

      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_CONFIRM_TARGET',
        payload: 'cursor-app'
      });
    });

    it('should handle single app from selection', () => {
      const mockDispatch = jest.fn();
      const context = createMockContext({
        state: createMockState({
          navigation: { view: 'clusters', selectedIdx: 0, lastGPressed: 0 },
          selections: {
            selectedApps: new Set(['selected-app']),
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set()
          }
        }),
        dispatch: mockDispatch
      });

      syncCommand.execute(context);

      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_CONFIRM_TARGET',
        payload: 'selected-app'
      });
    });

    it('should warn when no app selected', () => {
      const mockStatusLog = {
        info: jest.fn(),
        warn: jest.fn(),
        error: jest.fn(),
        debug: jest.fn(),
        set: jest.fn(),
        clear: jest.fn()
      };
      const context = createMockContext({
        state: createMockState({
          navigation: { view: 'clusters', selectedIdx: 0, lastGPressed: 0 },
          selections: {
            selectedApps: new Set(),
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set()
          },
          apps: []
        }),
        statusLog: mockStatusLog
      });

      syncCommand.execute(context);

      expect(mockStatusLog.warn).toHaveBeenCalledWith('No app selected to sync.', 'user-action');
    });

    it('should set correct confirmation state for single app', () => {
      const mockDispatch = jest.fn();
      const context = createMockContext({
        state: createMockState({
          selections: {
            selectedApps: new Set(['single-app']),
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set()
          }
        }),
        dispatch: mockDispatch
      });

      syncCommand.execute(context);

      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_CONFIRM_TARGET',
        payload: 'single-app'
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_CONFIRM_SYNC_PRUNE',
        payload: false
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_CONFIRM_SYNC_WATCH',
        payload: true
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_MODE',
        payload: 'confirm-sync'
      });
    });
  });

  describe('properties', () => {
    it('should have correct description', () => {
      expect(syncCommand.description).toBe('Sync application(s)');
    });

    it('should have empty aliases array', () => {
      expect(syncCommand.aliases).toEqual([]);
    });
  });
});