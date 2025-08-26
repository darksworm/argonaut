# Comprehensive UI Testing Guide for TUI Applications

This guide demonstrates how to implement robust UI testing for your Ink-based TUI application using `ink-testing-library`.

## Overview

The UI testing suite provides comprehensive coverage of:
- **User interactions** - Keyboard navigation, commands, search
- **Visual output** - Screen rendering, modals, states
- **Workflows** - End-to-end user journeys
- **Regression detection** - Snapshot testing for visual changes

## Installation and Setup

The testing infrastructure is already installed and configured:

```bash
# ink-testing-library is installed
bun add --dev ink-testing-library

# Tests are configured in jest.config.js
# UI tests are located in src/__tests__/ui/
```

## Core Testing Patterns

### 1. Basic Component Rendering

```typescript
import { render } from 'ink-testing-library';
import React from 'react';

it('renders component correctly', () => {
  const { lastFrame } = render(<MyComponent />);
  expect(lastFrame()).toContain('Expected Text');
});
```

### 2. User Input Simulation

```typescript
it('handles keyboard navigation', () => {
  const { lastFrame, stdin } = render(<MyComponent />);
  
  // Simulate key presses
  stdin.write('j'); // Down
  stdin.write('k'); // Up
  stdin.write('\r'); // Enter
  stdin.write('\u001b'); // Escape
  
  expect(lastFrame()).toContain('Expected Result');
});
```

### 3. Key Mapping for TUI Applications

```typescript
const keyMap = {
  'enter': '\r',
  'escape': '\u001b',
  'tab': '\t',
  'space': ' ',
  'up': '\u001b[A',
  'down': '\u001b[B',
  'left': '\u001b[D',
  'right': '\u001b[C',
  'ctrl+c': '\u0003',
  'backspace': '\u007f',
};

// Usage
stdin.write(keyMap.down);
stdin.write(keyMap.enter);
```

### 4. Testing Search and Command Workflows

```typescript
it('handles search workflow', () => {
  const { lastFrame, stdin } = render(<AppComponent />);
  
  // Enter search mode
  stdin.write('/');
  
  // Type search query
  stdin.write('my-query');
  
  // Confirm search
  stdin.write('\r');
  
  expect(lastFrame()).toContain('Search Results');
});
```

## Test Structure and Organization

### Recommended File Structure

```
src/__tests__/ui/
├── components/           # Individual component tests
│   ├── Modal.test.tsx
│   ├── ListView.test.tsx
│   └── SearchBar.test.tsx
├── workflows/           # End-to-end workflow tests
│   ├── sync-workflow.test.tsx
│   ├── navigation.test.tsx
│   └── search-workflow.test.tsx
├── snapshots/          # Visual regression tests
│   └── screen-snapshots.test.tsx
├── utils/              # Test utilities
│   ├── test-helpers.tsx
│   ├── mock-factories.tsx
│   └── key-simulation.tsx
└── integration/        # Full app integration tests
    └── app-integration.test.tsx
```

### Test Categories

1. **Component Tests** - Individual component behavior
2. **Integration Tests** - Component interaction
3. **Workflow Tests** - Complete user journeys
4. **Snapshot Tests** - Visual regression detection
5. **Performance Tests** - Large dataset handling

## Working Examples

### Example 1: Navigation Testing

```typescript
describe('App Navigation', () => {
  const mockApps = [
    { name: 'app1', status: 'healthy' },
    { name: 'app2', status: 'degraded' },
    { name: 'app3', status: 'synced' },
  ];

  it('navigates through app list', () => {
    const { lastFrame, stdin } = render(
      <AppList apps={mockApps} />
    );

    // Start at first item
    expect(lastFrame()).toMatch(/.*app1.*/);

    // Navigate down
    stdin.write('j');
    expect(lastFrame()).toMatch(/.*app2.*/);

    // Navigate up
    stdin.write('k');
    expect(lastFrame()).toMatch(/.*app1.*/);

    // Jump to end
    stdin.write('G');
    expect(lastFrame()).toMatch(/.*app3.*/);

    // Jump to beginning
    stdin.write('g');
    stdin.write('g');
    expect(lastFrame()).toMatch(/.*app1.*/);
  });
});
```

### Example 2: Modal Interaction Testing

```typescript
describe('Confirmation Modal', () => {
  it('shows and handles sync confirmation', async () => {
    const { lastFrame, stdin } = render(<SyncModal appName="test-app" />);
    
    // Modal should be visible
    expect(lastFrame()).toContain('Confirm sync for test-app');
    expect(lastFrame()).toMatch(/[yY]es.*[nN]o/);
    
    // Test confirmation
    stdin.write('y');
    
    // Modal should handle confirmation
    // (exact behavior depends on implementation)
    expect(lastFrame()).toBeDefined();
  });

  it('cancels on escape or no', () => {
    const { lastFrame, stdin } = render(<SyncModal appName="test-app" />);
    
    stdin.write('n');
    
    // Should handle cancellation
    expect(lastFrame()).toBeDefined();
  });
});
```

### Example 3: Search Functionality Testing

```typescript
describe('Search Integration', () => {
  const apps = [
    { name: 'frontend-app', namespace: 'prod' },
    { name: 'backend-service', namespace: 'prod' },
    { name: 'database', namespace: 'staging' },
  ];

  it('filters results based on search query', () => {
    const { lastFrame, stdin } = render(<AppList apps={apps} />);
    
    // Enter search mode
    stdin.write('/');
    expect(lastFrame()).toContain('/');
    
    // Type search query
    stdin.write('frontend');
    
    // Should show filtered results
    expect(lastFrame()).toContain('frontend-app');
    expect(lastFrame()).not.toContain('backend-service');
    
    // Clear search
    stdin.write('\u001b'); // Escape
    
    // Should show all results again
    expect(lastFrame()).toContain('backend-service');
  });
});
```

### Example 4: Snapshot Testing

```typescript
describe('Visual Regression Tests', () => {
  it('renders main screen consistently', () => {
    const { lastFrame } = render(<MainApp />);
    expect(lastFrame()).toMatchSnapshot('main-screen');
  });

  it('renders loading state consistently', () => {
    const { lastFrame } = render(<MainApp loading={true} />);
    expect(lastFrame()).toMatchSnapshot('loading-screen');
  });

  it('renders error state consistently', () => {
    const { lastFrame } = render(<MainApp error="Connection failed" />);
    expect(lastFrame()).toMatchSnapshot('error-screen');
  });
});
```

## Advanced Testing Techniques

### 1. Async State Testing

```typescript
it('handles async operations', async () => {
  const { lastFrame, stdin, rerender } = render(<AsyncComponent />);
  
  // Trigger async operation
  stdin.write('\r');
  
  // Wait for state changes
  await new Promise(resolve => setTimeout(resolve, 100));
  
  expect(lastFrame()).toContain('Async Result');
});
```

### 2. Context and State Management

```typescript
it('works with context providers', () => {
  const TestWrapper = ({ children }) => (
    <AppStateProvider initialState={{ mode: 'test' }}>
      {children}
    </AppStateProvider>
  );

  const { lastFrame } = render(<MyComponent />, { wrapper: TestWrapper });
  expect(lastFrame()).toBeDefined();
});
```

### 3. Large Dataset Testing

```typescript
it('handles large datasets efficiently', () => {
  const manyApps = Array.from({ length: 1000 }, (_, i) => ({
    name: `app-${i}`,
    status: i % 2 === 0 ? 'healthy' : 'degraded',
  }));

  const { lastFrame, stdin } = render(<AppList apps={manyApps} />);
  
  // Should render without performance issues
  expect(lastFrame()).toBeDefined();
  
  // Navigation should work with large datasets
  stdin.write('G'); // Jump to end
  stdin.write('g'); stdin.write('g'); // Jump to beginning
  
  expect(lastFrame()).toContain('app-0');
});
```

## Best Practices

### 1. Test User Workflows, Not Implementation Details

```typescript
// Good: Tests user workflow
it('completes sync workflow', () => {
  const { stdin } = render(<App />);
  
  // User selects app and syncs
  stdin.write('j'); // Navigate to app
  stdin.write(':'); // Enter command mode
  stdin.write('sync'); // Type sync command
  stdin.write('\r'); // Execute
  stdin.write('y'); // Confirm
  
  // Verify result from user perspective
});

// Avoid: Testing internal implementation
it('calls syncApp method', () => {
  // This tests implementation, not user experience
});
```

### 2. Use Descriptive Test Names

```typescript
describe('Search Functionality', () => {
  it('filters apps by name when typing in search mode');
  it('clears filter when exiting search mode with escape');
  it('maintains navigation state when switching between filtered and unfiltered views');
  it('shows "no results" message when search query matches no apps');
});
```

### 3. Mock External Dependencies

```typescript
// Mock API calls and side effects
const mockApi = {
  syncApp: jest.fn().mockResolvedValue({ success: true }),
  getApps: jest.fn().mockResolvedValue(mockApps),
};

it('handles API operations', async () => {
  // Test UI with mocked backend
});
```

### 4. Test Edge Cases

```typescript
describe('Edge Cases', () => {
  it('handles empty app list gracefully');
  it('handles very long app names');
  it('handles rapid key input');
  it('handles terminal resize during operation');
  it('recovers from network errors');
});
```

## Running Tests

```bash
# Run all UI tests
bun test ui/

# Run specific test file
bun test navigation.test.tsx

# Run with watch mode
bun test --watch ui/

# Run with coverage
bun test --coverage ui/

# Update snapshots
bun test --updateSnapshot ui/snapshots/
```

## Debugging Tips

### 1. Inspect Frame Output

```typescript
it('debug test output', () => {
  const { lastFrame } = render(<Component />);
  console.log('Current frame:', lastFrame());
  
  // Use this to understand what's actually rendered
});
```

### 2. Step Through Interactions

```typescript
it('step through user interaction', () => {
  const { lastFrame, stdin } = render(<Component />);
  
  console.log('Initial:', lastFrame());
  
  stdin.write('j');
  console.log('After j:', lastFrame());
  
  stdin.write('\r');
  console.log('After enter:', lastFrame());
});
```

### 3. Test Individual Key Mappings

```typescript
it('verifies key mapping', () => {
  const { stdin } = render(<Component />);
  
  // Test each key individually to isolate issues
  stdin.write('\u001b[A'); // Up arrow
  stdin.write('\u001b[B'); // Down arrow
  // etc.
});
```

## Integration with CI/CD

Add UI tests to your CI pipeline:

```yaml
# GitHub Actions example
- name: Run UI Tests
  run: bun test ui/ --ci --coverage

- name: Upload Coverage
  uses: codecov/codecov-action@v3
  with:
    files: ./coverage/lcov.info
```

## Conclusion

This UI testing approach provides:

✅ **Comprehensive Coverage** - User interactions, visual output, workflows  
✅ **Regression Detection** - Snapshot testing catches visual changes  
✅ **Fast Feedback** - Tests run quickly and provide immediate results  
✅ **Maintainability** - Clear patterns and utilities make tests easy to write  
✅ **Real User Simulation** - Tests actual key presses and terminal output  

The key to effective TUI testing is focusing on the user experience rather than implementation details, using realistic input simulation, and maintaining comprehensive coverage of user workflows.