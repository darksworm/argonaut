# UI Testing Suite for TUI App

This directory contains UI tests for the TUI application using `ink-testing-library`. These tests focus on simulating user interactions and verifying visual output.

## Test Structure

### Test Files

- **`basic-ui.test.tsx`** - Basic component rendering and interaction tests
- **`working-ui-tests.test.tsx`** - Comprehensive working UI tests with examples
- **`minimal.ui.test.tsx`** - Minimal integration tests with app context
- **`simple-test.tsx`** - Simple ink-testing-library verification

## Key Testing Patterns

### 1. Component Rendering Tests
```typescript
const { lastFrame } = renderApp(testScenarios.withApps(mockApps));
await waitForNextFrame();
expectToContainText(lastFrame(), 'expected content');
```

### 2. User Interaction Simulation
```typescript
simulateKey(stdin, 'j'); // Navigate down
simulateKey(stdin, 'enter'); // Confirm action
simulateSearch(stdin, 'query'); // Search workflow
simulateCommand(stdin, 'sync'); // Command execution
```

### 3. Visual State Assertions
```typescript
expect(isElementSelected(lastFrame(), 'app-name')).toBe(true);
expectToContainText(frame, 'Loading');
expectToNotContainText(frame, 'Hidden Element');
```

### 4. Snapshot Testing
```typescript
expect(lastFrame()).toMatchSnapshot('test-scenario-name');
```

## Test Categories

### Basic Functionality
- Application startup and loading states
- Authentication screens
- Main navigation and list display
- Modal dialogs and confirmations

### User Interactions
- Keyboard navigation (vim-style and arrow keys)
- Search mode entry and filtering
- Command mode and command execution
- Modal interactions and confirmations

### Visual Regression
- Snapshot tests for all major screens
- Different terminal size adaptations
- Status indicators and themes
- Error states and edge cases

### End-to-End Workflows
- Complete sync workflows
- Rollback procedures
- Multi-app selection and batch operations
- Search -> filter -> action workflows
- Error recovery scenarios

## Running Tests

```bash
# Run all UI tests
bun test ui/

# Run specific test file
bun test ui/App.ui.test.tsx

# Run with coverage
bun test --coverage ui/

# Update snapshots
bun test --updateSnapshot ui/Snapshots.ui.test.tsx
```

## Mock Data and Scenarios

The test suite uses factory functions to create consistent test data:

```typescript
// Create mock applications with different states
const healthyApp = createMockApp({
  name: 'my-app',
  health: { status: 'Healthy' },
  sync: { status: 'Synced', revision: 'abc123' }
});

// Use predefined test scenarios
const { lastFrame } = renderApp(testScenarios.withLoadingState());
```

## Best Practices

### 1. Wait for Async Updates
Always use `await waitForNextFrame()` after simulating user input to ensure the UI has time to update.

### 2. Use Semantic Assertions
Prefer descriptive assertion helpers over direct string matching:
```typescript
// Good
expectToContainText(frame, 'App Name');
expect(isElementSelected(frame, 'App Name')).toBe(true);

// For text with ANSI codes, strip them first
import { stripAnsi } from '../test-utils';
const cleanFrame = stripAnsi(frame);
expect(cleanFrame).toContain('Expected text without colors');

// Avoid
expect(frame.includes('App Name')).toBe(true);
```

### 3. Test User Workflows
Focus on complete user journeys rather than isolated component behavior:
```typescript
// Test complete workflow
simulateSearch(stdin, 'production');
await waitForNextFrame();
simulateKey(stdin, 'j'); // navigate
await waitForNextFrame();
simulateCommand(stdin, 'sync'); // action
await waitForNextFrame();
simulateKey(stdin, 'y'); // confirm
```

### 4. Mock External Dependencies
All external API calls, file system access, and side effects are mocked to keep tests deterministic and fast.

### 5. Test Edge Cases
Include tests for:
- Empty states
- Large datasets
- Error conditions
- Rapid user input
- Terminal size constraints

## Snapshot Management

Snapshots are used to detect visual regressions. Update them when UI changes are intentional:

```bash
# Update specific snapshot
bun test --updateSnapshot ui/Snapshots.ui.test.tsx -t "loading screen"

# Update all snapshots (use carefully)
bun test --updateSnapshot ui/Snapshots.ui.test.tsx
```

## Troubleshooting

### Common Issues

1. **Flaky Tests**: Usually caused by not waiting for async updates. Always use `waitForNextFrame()`.

2. **Snapshot Mismatches**: Check if the UI change is intentional. Update snapshots if needed.

3. **Mock Issues**: Ensure all external dependencies are properly mocked in `ui-test-utils.tsx`.

4. **Terminal Size Issues**: Some tests may be sensitive to terminal dimensions. Use consistent sizes in test scenarios.

### Debugging Tips

```typescript
// Log current frame content for debugging
console.log('Current frame:', lastFrame());

// Strip ANSI codes for cleaner debugging
import { stripAnsi } from '../test-utils';
console.log('Clean frame:', stripAnsi(lastFrame()));

// Check visible lines
console.log('Visible lines:', getVisibleLines(lastFrame()));

// Verify element selection
console.log('Is selected:', isElementSelected(lastFrame(), 'element-name'));
```

## Contributing

When adding new UI tests:

1. Use existing utilities and patterns
2. Add new helpers to `ui-test-utils.tsx` if needed
3. Follow the existing file organization
4. Include both positive and negative test cases
5. Add snapshot tests for new UI components
6. Test complete user workflows, not just component isolation
7. Ensure tests are deterministic and fast

## Coverage Goals

- **Navigation**: All keyboard shortcuts and navigation patterns
- **Commands**: All supported commands and their workflows
- **Modals**: All modal states and interactions
- **Search**: All search and filtering scenarios
- **Edge Cases**: Empty states, errors, large datasets
- **Workflows**: Complete end-to-end user journeys
- **Visual**: Snapshot coverage of all major screens and states