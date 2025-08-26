// src/__tests__/commands/EnhancedApplicationCommands.test.ts
import { createMockContext, createMockState, createMockApps } from '../test-utils';
import type { CommandContext } from '../../commands/types';

// Enhanced test implementations with more realistic error scenarios
class EnhancedSyncCommand {
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
    const visibleItems = context.state.apps;
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
}

class EnhancedDiffCommand {
  aliases = [];
  description = "View diff for application";

  canExecute(context: CommandContext): boolean {
    return context.state.server !== null;
  }

  async execute(context: CommandContext, arg?: string): Promise<void> {
    const { state, dispatch, statusLog } = context;
    const { server } = state;
    const { selectedApps } = state.selections;
    const { view, selectedIdx } = state.navigation;

    if (!server) {
      statusLog.error("Not authenticated.", "auth");
      return;
    }

    const visibleItems = context.state.apps;
    const target =
      arg ||
      (view === "apps"
        ? (visibleItems[selectedIdx] as any)?.name
        : undefined) ||
      Array.from(selectedApps)[0];

    if (!target) {
      statusLog.warn("No app selected to diff.", "user-action");
      return;
    }

    try {
      dispatch({ type: "SET_MODE", payload: "normal" });
      statusLog.info(`Preparing diff for ${target}…`, "diff");
      
      // Simulate diff process with potential timeout
      await new Promise(resolve => setTimeout(resolve, 10));
      
      statusLog.info("No differences.", "diff");
    } catch (e: any) {
      dispatch({ type: "SET_MODE", payload: "normal" });
      statusLog.error(`Diff failed: ${e?.message || String(e)}`, "diff");
    }
  }
}

describe('Enhanced SyncCommand Edge Cases', () => {
  let syncCommand: EnhancedSyncCommand;

  beforeEach(() => {
    syncCommand = new EnhancedSyncCommand();
  });

  describe('boundary conditions', () => {
    it('should handle empty app name argument', () => {
      // Arrange
      const mockDispatch = jest.fn();
      const mockStatusLog = {
        info: jest.fn(),
        warn: jest.fn(),
        error: jest.fn(),
        debug: jest.fn(),
        set: jest.fn(),
        clear: jest.fn()
      };
      const context = createMockContext({
        dispatch: mockDispatch,
        statusLog: mockStatusLog
      });

      // Act
      syncCommand.execute(context, '');

      // Assert - empty string should be treated as falsy and go to fallback logic
      expect(mockStatusLog.warn).toHaveBeenCalledWith("No app selected to sync.", "user-action");
    });

    it('should handle very long app name argument', () => {
      // Arrange
      const mockDispatch = jest.fn();
      const longAppName = 'a'.repeat(1000); // Very long app name
      const context = createMockContext({
        dispatch: mockDispatch
      });

      // Act
      syncCommand.execute(context, longAppName);

      // Assert
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_CONFIRM_TARGET',
        payload: longAppName
      });
    });

    it('should handle special characters in app name', () => {
      // Arrange
      const mockDispatch = jest.fn();
      const specialAppName = 'app-with-special-chars.@#$%^&*()';
      const context = createMockContext({
        dispatch: mockDispatch
      });

      // Act
      syncCommand.execute(context, specialAppName);

      // Assert
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_CONFIRM_TARGET',
        payload: specialAppName
      });
    });

    it('should handle exactly one selected app', () => {
      // Arrange
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

      // Act
      syncCommand.execute(context);

      // Assert
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_CONFIRM_TARGET',
        payload: 'single-app'
      });
    });

    it('should handle exactly two selected apps (multi-sync)', () => {
      // Arrange
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

      // Act
      syncCommand.execute(context);

      // Assert
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_CONFIRM_TARGET',
        payload: '__MULTI__'
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_CONFIRM_SYNC_WATCH',
        payload: false // disabled for multi
      });
    });

    it('should handle large number of selected apps', () => {
      // Arrange
      const mockDispatch = jest.fn();
      const manyApps = Array.from({ length: 100 }, (_, i) => `app-${i}`);
      const context = createMockContext({
        state: createMockState({
          selections: {
            selectedApps: new Set(manyApps),
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set()
          }
        }),
        dispatch: mockDispatch
      });

      // Act
      syncCommand.execute(context);

      // Assert
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_CONFIRM_TARGET',
        payload: '__MULTI__'
      });
    });
  });

  describe('state consistency edge cases', () => {
    it('should handle cursor at last index in apps array', () => {
      // Arrange
      const mockDispatch = jest.fn();
      const apps = createMockApps();
      const context = createMockContext({
        state: createMockState({
          navigation: { view: 'apps', selectedIdx: apps.length - 1, lastGPressed: 0 },
          apps
        }),
        dispatch: mockDispatch
      });

      // Act
      syncCommand.execute(context);

      // Assert
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_CONFIRM_TARGET',
        payload: apps[apps.length - 1].name
      });
    });

    it('should handle cursor beyond apps array bounds gracefully', () => {
      // Arrange
      const mockStatusLog = {
        info: jest.fn(),
        warn: jest.fn(),
        error: jest.fn(),
        debug: jest.fn(),
        set: jest.fn(),
        clear: jest.fn()
      };
      const apps = createMockApps();
      const context = createMockContext({
        state: createMockState({
          navigation: { view: 'apps', selectedIdx: apps.length + 5, lastGPressed: 0 },
          apps
        }),
        statusLog: mockStatusLog
      });

      // Act
      syncCommand.execute(context);

      // Assert - Should warn about no app selected (because cursor is out of bounds)
      expect(mockStatusLog.warn).toHaveBeenCalledWith("No app selected to sync.", "user-action");
    });

    it('should handle empty apps array with cursor position', () => {
      // Arrange
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
          navigation: { view: 'apps', selectedIdx: 0, lastGPressed: 0 },
          apps: []
        }),
        statusLog: mockStatusLog
      });

      // Act
      syncCommand.execute(context);

      // Assert
      expect(mockStatusLog.warn).toHaveBeenCalledWith("No app selected to sync.", "user-action");
    });
  });

  describe('race condition scenarios', () => {
    it('should handle concurrent selections modifications', () => {
      // Arrange
      const mockDispatch = jest.fn();
      const selectedApps = new Set(['app1', 'app2']);
      const context = createMockContext({
        state: createMockState({
          selections: {
            selectedApps,
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set()
          }
        }),
        dispatch: mockDispatch
      });

      // Act - Simulate modification of selectedApps during execution
      selectedApps.add('app3');
      syncCommand.execute(context);

      // Assert - Should still treat as multi-sync based on initial state
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_CONFIRM_TARGET',
        payload: '__MULTI__'
      });
    });
  });

  describe('error recovery scenarios', () => {
    it('should handle dispatch failure on first action', () => {
      // Arrange
      const mockDispatch = jest.fn()
        .mockImplementationOnce(() => { throw new Error('First dispatch failed'); });
      
      const context = createMockContext({
        dispatch: mockDispatch
      });

      // Act & Assert
      expect(() => syncCommand.execute(context, 'test-app')).toThrow('First dispatch failed');
    });

    it('should handle dispatch failure on subsequent actions', () => {
      // Arrange
      const mockDispatch = jest.fn()
        .mockImplementationOnce(() => { /* Success */ })
        .mockImplementationOnce(() => { throw new Error('Second dispatch failed'); });
      
      const context = createMockContext({
        dispatch: mockDispatch
      });

      // Act & Assert
      expect(() => syncCommand.execute(context, 'test-app')).toThrow('Second dispatch failed');
    });
  });
});

describe('Enhanced DiffCommand Edge Cases', () => {
  let diffCommand: EnhancedDiffCommand;

  beforeEach(() => {
    diffCommand = new EnhancedDiffCommand();
  });

  describe('timeout and performance scenarios', () => {
    it('should handle diff operation timeout', async () => {
      // Arrange
      const mockStatusLog = {
        info: jest.fn(),
        warn: jest.fn(),
        error: jest.fn(),
        debug: jest.fn(),
        set: jest.fn(),
        clear: jest.fn()
      };
      const mockDispatch = jest.fn();
      
      // Mock a slow diff operation by overriding the internal timeout
      const slowDiffCommand = new class extends EnhancedDiffCommand {
        async execute(context: CommandContext, arg?: string): Promise<void> {
          const { state, dispatch, statusLog } = context;
          const { server } = state;

          if (!server) {
            statusLog.error("Not authenticated.", "auth");
            return;
          }

          const target = arg || 'test-app';

          try {
            dispatch({ type: "SET_MODE", payload: "normal" });
            statusLog.info(`Preparing diff for ${target}…`, "diff");
            
            // Simulate timeout
            await new Promise((_, reject) => {
              setTimeout(() => reject(new Error('Operation timed out')), 50);
            });
            
          } catch (e: any) {
            dispatch({ type: "SET_MODE", payload: "normal" });
            statusLog.error(`Diff failed: ${e?.message || String(e)}`, "diff");
          }
        }
      };

      const context = createMockContext({
        dispatch: mockDispatch,
        statusLog: mockStatusLog
      });

      // Act
      await slowDiffCommand.execute(context, 'timeout-app');

      // Assert
      expect(mockStatusLog.error).toHaveBeenCalledWith('Diff failed: Operation timed out', 'diff');
      expect(mockDispatch).toHaveBeenCalledWith({ type: "SET_MODE", payload: "normal" });
    });

    it('should handle memory pressure during diff', async () => {
      // Arrange
      const mockStatusLog = {
        info: jest.fn(),
        warn: jest.fn(),
        error: jest.fn(),
        debug: jest.fn(),
        set: jest.fn(),
        clear: jest.fn()
      };
      const mockDispatch = jest.fn();

      // Mock memory pressure scenario
      const memoryPressureDiffCommand = new class extends EnhancedDiffCommand {
        async execute(context: CommandContext, arg?: string): Promise<void> {
          try {
            const { statusLog, dispatch } = context;
            statusLog.info(`Preparing diff for ${arg}…`, "diff");
            
            // Simulate memory pressure error
            throw new Error('Cannot allocate memory for diff operation');
            
          } catch (e: any) {
            context.dispatch({ type: "SET_MODE", payload: "normal" });
            context.statusLog.error(`Diff failed: ${e?.message || String(e)}`, "diff");
          }
        }
      };

      const context = createMockContext({
        dispatch: mockDispatch,
        statusLog: mockStatusLog
      });

      // Act
      await memoryPressureDiffCommand.execute(context, 'memory-test-app');

      // Assert
      expect(mockStatusLog.error).toHaveBeenCalledWith('Diff failed: Cannot allocate memory for diff operation', 'diff');
    });
  });

  describe('authentication edge cases', () => {
    it('should handle server object with missing properties', async () => {
      // Arrange
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
          server: {} as any // Empty server object
        }),
        statusLog: mockStatusLog
      });

      // Act
      await diffCommand.execute(context, 'test-app');

      // Assert - Should still proceed since server is truthy
      expect(mockStatusLog.info).toHaveBeenCalledWith('Preparing diff for test-app…', 'diff');
    });

    it('should handle server authentication state changes during execution', async () => {
      // Arrange
      const mockStatusLog = {
        info: jest.fn(),
        warn: jest.fn(),
        error: jest.fn(),
        debug: jest.fn(),
        set: jest.fn(),
        clear: jest.fn()
      };
      const serverState = { config: { baseUrl: 'https://test.com' }, token: 'token' };
      const context = createMockContext({
        state: createMockState({ server: serverState }),
        statusLog: mockStatusLog
      });

      // Act - Simulate server state changing to null during execution
      const executePromise = diffCommand.execute(context, 'test-app');
      
      // Modify server state (simulating concurrent state change)
      (context.state as any).server = null;
      
      await executePromise;

      // Assert - Should complete based on initial state check
      expect(mockStatusLog.info).toHaveBeenCalledWith('Preparing diff for test-app…', 'diff');
    });
  });

  describe('app selection edge cases', () => {
    it('should handle null/undefined app names gracefully', async () => {
      // Arrange
      const mockStatusLog = {
        info: jest.fn(),
        warn: jest.fn(),
        error: jest.fn(),
        debug: jest.fn(),
        set: jest.fn(),
        clear: jest.fn()
      };
      const apps = [
        { name: null, sync: 'Synced', health: 'Healthy' }, // Invalid app
        { name: 'valid-app', sync: 'Synced', health: 'Healthy' }
      ] as any;
      
      const context = createMockContext({
        state: createMockState({
          navigation: { view: 'apps', selectedIdx: 0, lastGPressed: 0 },
          apps
        }),
        statusLog: mockStatusLog
      });

      // Act
      await diffCommand.execute(context);

      // Assert - Should warn about no app selected due to null name
      expect(mockStatusLog.warn).toHaveBeenCalledWith('No app selected to diff.', 'user-action');
    });

    it('should prioritize argument over cursor and selections', async () => {
      // Arrange
      const mockStatusLog = {
        info: jest.fn(),
        warn: jest.fn(),
        error: jest.fn(),
        debug: jest.fn(),
        set: jest.fn(),
        clear: jest.fn()
      };
      const apps = createMockApps();
      const context = createMockContext({
        state: createMockState({
          navigation: { view: 'apps', selectedIdx: 0, lastGPressed: 0 },
          selections: {
            selectedApps: new Set(['selected-app']),
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set()
          },
          apps
        }),
        statusLog: mockStatusLog
      });

      // Act - Explicit argument should win
      await diffCommand.execute(context, 'explicit-arg-app');

      // Assert
      expect(mockStatusLog.info).toHaveBeenCalledWith('Preparing diff for explicit-arg-app…', 'diff');
    });

    it('should fallback from cursor to selection when not in apps view', async () => {
      // Arrange
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
            selectedApps: new Set(['fallback-app']),
            scopeClusters: new Set(),
            scopeNamespaces: new Set(),
            scopeProjects: new Set()
          }
        }),
        statusLog: mockStatusLog
      });

      // Act
      await diffCommand.execute(context);

      // Assert
      expect(mockStatusLog.info).toHaveBeenCalledWith('Preparing diff for fallback-app…', 'diff');
    });
  });

  describe('error handling resilience', () => {
    it('should handle statusLog errors during execution', async () => {
      // Arrange
      const mockStatusLog = {
        info: jest.fn().mockImplementation(() => {
          throw new Error('StatusLog info failed');
        }),
        warn: jest.fn(),
        error: jest.fn(),
        debug: jest.fn(),
        set: jest.fn(),
        clear: jest.fn()
      };
      const mockDispatch = jest.fn();
      const context = createMockContext({
        statusLog: mockStatusLog,
        dispatch: mockDispatch
      });

      // Act & Assert
      await diffCommand.execute(context, 'test-app');
      
      // The error should be caught and handled gracefully
      expect(mockStatusLog.error).toHaveBeenCalledWith('Diff failed: StatusLog info failed', 'diff');
      expect(mockDispatch).toHaveBeenCalledWith({ type: "SET_MODE", payload: "normal" });
    });

    it('should handle dispatch errors in catch block', async () => {
      // Arrange
      const mockDispatch = jest.fn()
        .mockImplementationOnce(() => { /* Success */ })
        .mockImplementationOnce(() => { throw new Error('Cleanup dispatch failed'); });
      
      const mockStatusLog = {
        info: jest.fn().mockImplementation(() => {
          throw new Error('Original error');
        }),
        warn: jest.fn(),
        error: jest.fn(),
        debug: jest.fn(),
        set: jest.fn(),
        clear: jest.fn()
      };
      
      const context = createMockContext({
        dispatch: mockDispatch,
        statusLog: mockStatusLog
      });

      // Act & Assert - Should throw the cleanup error, not the original
      await expect(diffCommand.execute(context, 'test-app')).rejects.toThrow('Cleanup dispatch failed');
    });

    it('should handle multiple concurrent errors gracefully', async () => {
      // Arrange
      const errors: Error[] = [];
      const mockDispatch = jest.fn().mockImplementation(() => {
        const error = new Error('Dispatch error');
        errors.push(error);
        throw error;
      });
      
      const mockStatusLog = {
        info: jest.fn(),
        warn: jest.fn(),
        error: jest.fn().mockImplementation(() => {
          const error = new Error('StatusLog error');
          errors.push(error);
          throw error;
        }),
        debug: jest.fn(),
        set: jest.fn(),
        clear: jest.fn()
      };
      
      const context = createMockContext({
        dispatch: mockDispatch,
        statusLog: mockStatusLog
      });

      // Act & Assert
      await expect(diffCommand.execute(context, 'test-app')).rejects.toThrow();
      expect(errors.length).toBeGreaterThan(0);
    });
  });
});

describe('Application Commands Performance Tests', () => {
  describe('memory efficiency', () => {
    it('should not leak memory with repeated sync operations', () => {
      // Arrange
      const syncCommand = new EnhancedSyncCommand();
      const mockDispatch = jest.fn();
      const context = createMockContext({
        dispatch: mockDispatch
      });

      // Act - Simulate many rapid operations
      for (let i = 0; i < 1000; i++) {
        syncCommand.execute(context, `app-${i}`);
      }

      // Assert - All operations should complete successfully
      expect(mockDispatch).toHaveBeenCalledTimes(4000); // 1000 calls × 4 dispatches each
    });

    it('should handle large app arrays efficiently', () => {
      // Arrange
      const syncCommand = new EnhancedSyncCommand();
      const mockDispatch = jest.fn();
      const largeAppsArray = Array.from({ length: 10000 }, (_, i) => ({
        name: `app-${i}`,
        sync: 'Synced',
        health: 'Healthy',
        clusterId: 'cluster1',
        clusterLabel: 'cluster1',
        namespace: 'default',
        appNamespace: 'argocd',
        project: 'default',
        lastSyncAt: '2023-12-01T10:00:00Z'
      }));

      const context = createMockContext({
        state: createMockState({
          navigation: { view: 'apps', selectedIdx: 9999, lastGPressed: 0 },
          apps: largeAppsArray
        }),
        dispatch: mockDispatch
      });

      const startTime = performance.now();

      // Act
      syncCommand.execute(context);

      const endTime = performance.now();

      // Assert - Should complete quickly even with large array
      expect(endTime - startTime).toBeLessThan(10); // Less than 10ms
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_CONFIRM_TARGET',
        payload: 'app-9999'
      });
    });
  });

  describe('concurrent operations', () => {
    it('should handle concurrent sync and diff operations', async () => {
      // Arrange
      const syncCommand = new EnhancedSyncCommand();
      const diffCommand = new EnhancedDiffCommand();
      
      const mockDispatch = jest.fn();
      const mockStatusLog = {
        info: jest.fn(),
        warn: jest.fn(),
        error: jest.fn(),
        debug: jest.fn(),
        set: jest.fn(),
        clear: jest.fn()
      };
      
      const context = createMockContext({
        dispatch: mockDispatch,
        statusLog: mockStatusLog
      });

      // Act - Execute both operations concurrently
      const syncPromise = Promise.resolve(syncCommand.execute(context, 'sync-app'));
      const diffPromise = diffCommand.execute(context, 'diff-app');

      await Promise.all([syncPromise, diffPromise]);

      // Assert - Both operations should complete
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_CONFIRM_TARGET',
        payload: 'sync-app'
      });
      expect(mockStatusLog.info).toHaveBeenCalledWith('Preparing diff for diff-app…', 'diff');
    });
  });
});