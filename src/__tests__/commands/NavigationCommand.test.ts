// src/__tests__/commands/NavigationCommand.test.ts
import { NavigationCommand, ClearCommand, ClearAllCommand } from '../../commands/navigation';
import { createMockContext, createMockState } from '../test-utils';
import type { View } from '../../types/domain';

describe('NavigationCommand', () => {
  describe('construction', () => {
    it('should set target view and aliases correctly', () => {
      const command = new NavigationCommand('apps' as View, 'apps', ['a', 'applications']);
      
      expect(command.description).toBe('Switch to apps view');
      expect(command.aliases).toEqual(['a', 'applications']);
    });
  });

  describe('canExecute', () => {
    it('should only allow execution in normal mode', () => {
      const command = new NavigationCommand('apps' as View, 'apps');
      
      const normalContext = createMockContext({
        state: createMockState({ mode: 'normal' })
      });
      expect(command.canExecute(normalContext)).toBe(true);
      
      const loadingContext = createMockContext({
        state: createMockState({ mode: 'loading' })
      });
      expect(command.canExecute(loadingContext)).toBe(false);
    });
  });

  describe('execute', () => {
    it('should handle cluster navigation with argument', () => {
      const command = new NavigationCommand('clusters' as View, 'clusters');
      const mockDispatch = jest.fn();
      const context = createMockContext({ dispatch: mockDispatch });

      command.execute(context, 'production');

      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'RESET_NAVIGATION',
        payload: { view: 'clusters' }
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_MODE',
        payload: 'normal'
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_SCOPE_CLUSTERS',
        payload: new Set(['production'])
      });
    });

    it('should handle cluster navigation without argument', () => {
      const command = new NavigationCommand('clusters' as View, 'clusters');
      const mockDispatch = jest.fn();
      const context = createMockContext({ dispatch: mockDispatch });

      command.execute(context);

      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'RESET_NAVIGATION',
        payload: { view: 'clusters' }
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_SCOPE_CLUSTERS',
        payload: new Set()
      });
    });

    it('should handle namespace navigation with argument', () => {
      const command = new NavigationCommand('namespaces' as View, 'namespaces');
      const mockDispatch = jest.fn();
      const context = createMockContext({ dispatch: mockDispatch });

      command.execute(context, 'kube-system');

      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'RESET_NAVIGATION',
        payload: { view: 'namespaces' }
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_SCOPE_NAMESPACES',
        payload: new Set(['kube-system'])
      });
    });

    it('should handle namespace navigation without argument', () => {
      const command = new NavigationCommand('namespaces' as View, 'namespaces');
      const mockDispatch = jest.fn();
      const context = createMockContext({ dispatch: mockDispatch });

      command.execute(context);

      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_SCOPE_NAMESPACES',
        payload: new Set()
      });
    });

    it('should handle project navigation with argument', () => {
      const command = new NavigationCommand('projects' as View, 'projects');
      const mockDispatch = jest.fn();
      const context = createMockContext({ dispatch: mockDispatch });

      command.execute(context, 'team-a');

      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'RESET_NAVIGATION',
        payload: { view: 'projects' }
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_SCOPE_PROJECTS',
        payload: new Set(['team-a'])
      });
    });

    it('should handle project navigation without argument', () => {
      const command = new NavigationCommand('projects' as View, 'projects');
      const mockDispatch = jest.fn();
      const context = createMockContext({ dispatch: mockDispatch });

      command.execute(context);

      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_SCOPE_PROJECTS',
        payload: new Set()
      });
    });

    it('should handle app navigation with argument', () => {
      const command = new NavigationCommand('apps' as View, 'apps');
      const mockDispatch = jest.fn();
      const context = createMockContext({ dispatch: mockDispatch });

      command.execute(context, 'my-app');

      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'RESET_NAVIGATION',
        payload: { view: 'apps' }
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_SELECTED_APPS',
        payload: new Set(['my-app'])
      });
    });

    it('should handle app navigation without argument', () => {
      const command = new NavigationCommand('apps' as View, 'apps');
      const mockDispatch = jest.fn();
      const context = createMockContext({ dispatch: mockDispatch });

      command.execute(context);

      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_SELECTED_APPS',
        payload: new Set()
      });
    });

    it('should clear selections when switching views', () => {
      const command = new NavigationCommand('apps' as View, 'apps');
      const mockDispatch = jest.fn();
      const context = createMockContext({ dispatch: mockDispatch });

      command.execute(context);

      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'RESET_NAVIGATION',
        payload: { view: 'apps' }
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_MODE',
        payload: 'normal'
      });
    });
  });
});

describe('ClearCommand', () => {
  let clearCommand: ClearCommand;

  beforeEach(() => {
    clearCommand = new ClearCommand();
  });

  describe('canExecute', () => {
    it('should only allow execution in normal mode', () => {
      const normalContext = createMockContext({
        state: createMockState({ mode: 'normal' })
      });
      expect(clearCommand.canExecute(normalContext)).toBe(true);

      const loadingContext = createMockContext({
        state: createMockState({ mode: 'loading' })
      });
      expect(clearCommand.canExecute(loadingContext)).toBe(false);
    });
  });

  describe('execute', () => {
    it('should clear clusters selection in clusters view', () => {
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
        state: createMockState({
          navigation: { view: 'clusters', selectedIdx: 0, lastGPressed: 0 }
        }),
        dispatch: mockDispatch,
        statusLog: mockStatusLog
      });

      clearCommand.execute(context);

      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_SCOPE_CLUSTERS',
        payload: new Set()
      });
      expect(mockStatusLog.info).toHaveBeenCalledWith('Selection cleared.', 'user-action');
    });

    it('should clear namespaces selection in namespaces view', () => {
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
        state: createMockState({
          navigation: { view: 'namespaces', selectedIdx: 0, lastGPressed: 0 }
        }),
        dispatch: mockDispatch,
        statusLog: mockStatusLog
      });

      clearCommand.execute(context);

      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_SCOPE_NAMESPACES',
        payload: new Set()
      });
      expect(mockStatusLog.info).toHaveBeenCalledWith('Selection cleared.', 'user-action');
    });

    it('should clear projects selection in projects view', () => {
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
        state: createMockState({
          navigation: { view: 'projects', selectedIdx: 0, lastGPressed: 0 }
        }),
        dispatch: mockDispatch,
        statusLog: mockStatusLog
      });

      clearCommand.execute(context);

      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_SCOPE_PROJECTS',
        payload: new Set()
      });
      expect(mockStatusLog.info).toHaveBeenCalledWith('Selection cleared.', 'user-action');
    });

    it('should clear apps selection in apps view', () => {
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
        state: createMockState({
          navigation: { view: 'apps', selectedIdx: 0, lastGPressed: 0 }
        }),
        dispatch: mockDispatch,
        statusLog: mockStatusLog
      });

      clearCommand.execute(context);

      expect(mockDispatch).toHaveBeenCalledWith({
        type: 'SET_SELECTED_APPS',
        payload: new Set()
      });
      expect(mockStatusLog.info).toHaveBeenCalledWith('Selection cleared.', 'user-action');
    });
  });

  describe('properties', () => {
    it('should have correct description', () => {
      expect(clearCommand.description).toBe('Clear current view selection');
    });

    it('should have empty aliases array', () => {
      expect(clearCommand.aliases).toEqual([]);
    });
  });
});

describe('ClearAllCommand', () => {
  let clearAllCommand: ClearAllCommand;

  beforeEach(() => {
    clearAllCommand = new ClearAllCommand();
  });

  describe('execute', () => {
    it('should clear all selections and filters', () => {
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

      clearAllCommand.execute(context);

      expect(mockDispatch).toHaveBeenCalledWith({ type: 'CLEAR_ALL_SELECTIONS' });
      expect(mockDispatch).toHaveBeenCalledWith({ type: 'CLEAR_FILTERS' });
      expect(mockStatusLog.info).toHaveBeenCalledWith('All filtering cleared.', 'user-action');
    });
  });

  describe('properties', () => {
    it('should have correct description', () => {
      expect(clearAllCommand.description).toBe('Clear all selections and filters');
    });

    it('should have empty aliases array', () => {
      expect(clearAllCommand.aliases).toEqual([]);
    });
  });
});