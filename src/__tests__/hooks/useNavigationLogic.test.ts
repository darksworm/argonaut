// biome-ignore lint/correctness/useHookAtTopLevel: Testing React hooks
import { beforeEach, describe, expect, mock, test } from "bun:test";
import type { AppItem, View } from "../../types/domain";
import { createMockState } from "../test-utils";

// Mock the dependencies
const mockUseAppState = mock();
const mockUseVisibleItems = mock();
const mockUseEffect = mock();
const mockUseCallback = mock();

// Mock React hooks
mock.module("react", () => ({
  useCallback: mockUseCallback,
  useEffect: mockUseEffect,
}));

mock.module("../../contexts/AppStateContext", () => ({
  useAppState: mockUseAppState,
}));

mock.module("../../hooks/useVisibleItems", () => ({
  useVisibleItems: mockUseVisibleItems,
}));

describe("useNavigationLogic", () => {
  let mockDispatch: ReturnType<typeof mock>;
  let mockVisibleItems: any[];
  let mockState: ReturnType<typeof createMockState>;

  beforeEach(() => {
    mockDispatch = mock();
    mockVisibleItems = [];
    mockState = createMockState();

    mockUseAppState.mockReturnValue({
      state: mockState,
      dispatch: mockDispatch,
    });

    mockUseVisibleItems.mockReturnValue({
      visibleItems: mockVisibleItems,
    });

    // Reset React hook mocks
    mockUseEffect.mockReset();
    mockUseCallback.mockReset();

    // Mock useEffect to call the effect immediately
    mockUseEffect.mockImplementation((effect: () => void) => {
      effect();
    });

    // Mock useCallback to return the callback function directly
    mockUseCallback.mockImplementation((callback: (...args: any[]) => any) => {
      return callback;
    });
  });

  describe("bounds checking logic", () => {
    test("should trigger selectedIdx update when out of bounds", async () => {
      mockState.navigation.selectedIdx = 5;
      mockVisibleItems = ["item1", "item2"]; // Only 2 items

      mockUseVisibleItems.mockReturnValue({
        visibleItems: mockVisibleItems,
      });

      // Dynamic import to test the hook
      const { useNavigationLogic } = await import(
        "../../hooks/useNavigationLogic"
      );

      // Call the hook - this will trigger useEffect
      useNavigationLogic();

      // Check that useEffect was called with the bounds checking logic
      expect(mockUseEffect).toHaveBeenCalled();

      // The effect should have been triggered, leading to dispatch call
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_IDX",
        payload: 1, // Should be clamped to 1 (length - 1)
      });
    });

    test("should not update selectedIdx when within bounds", async () => {
      mockState.navigation.selectedIdx = 1;
      mockVisibleItems = ["item1", "item2", "item3"];

      mockUseVisibleItems.mockReturnValue({
        visibleItems: mockVisibleItems,
      });

      const { useNavigationLogic } = await import(
        "../../hooks/useNavigationLogic"
      );
      useNavigationLogic();

      expect(mockUseEffect).toHaveBeenCalled();

      // Should not dispatch since index is within bounds
      expect(mockDispatch).not.toHaveBeenCalledWith({
        type: "SET_SELECTED_IDX",
        payload: expect.any(Number),
      });
    });

    test("should handle empty array gracefully", async () => {
      mockState.navigation.selectedIdx = 5;
      mockVisibleItems = [];

      mockUseVisibleItems.mockReturnValue({
        visibleItems: mockVisibleItems,
      });

      const { useNavigationLogic } = await import(
        "../../hooks/useNavigationLogic"
      );
      useNavigationLogic();

      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_IDX",
        payload: 0, // Should default to 0 for empty array
      });
    });
  });

  describe("drillDown callback logic", () => {
    test("should create drillDown callback for clusters view", async () => {
      mockState.navigation.view = "clusters" as View;
      mockState.navigation.selectedIdx = 0;
      mockVisibleItems = ["cluster1", "cluster2"];

      mockUseVisibleItems.mockReturnValue({
        visibleItems: mockVisibleItems,
      });

      const { useNavigationLogic } = await import(
        "../../hooks/useNavigationLogic"
      );
      const result = useNavigationLogic();

      expect(mockUseCallback).toHaveBeenCalled();

      // Get the drillDown callback and test it
      expect(result.drillDown).toBeDefined();
      expect(typeof result.drillDown).toBe("function");

      // Execute the drillDown function
      result.drillDown();

      expect(mockDispatch).toHaveBeenCalledWith({ type: "RESET_NAVIGATION" });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "CLEAR_LOWER_LEVEL_SELECTIONS",
        payload: "clusters",
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_CLUSTERS",
        payload: new Set(["cluster1"]),
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_VIEW",
        payload: "namespaces",
      });
    });

    test("should create drillDown callback for namespaces view", async () => {
      mockState.navigation.view = "namespaces" as View;
      mockState.navigation.selectedIdx = 1;
      mockVisibleItems = ["namespace1", "namespace2", "namespace3"];

      mockUseVisibleItems.mockReturnValue({
        visibleItems: mockVisibleItems,
      });

      const { useNavigationLogic } = await import(
        "../../hooks/useNavigationLogic"
      );
      const result = useNavigationLogic();

      result.drillDown();

      expect(mockDispatch).toHaveBeenCalledWith({ type: "RESET_NAVIGATION" });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "CLEAR_LOWER_LEVEL_SELECTIONS",
        payload: "namespaces",
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_NAMESPACES",
        payload: new Set(["namespace2"]),
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_VIEW",
        payload: "projects",
      });
    });

    test("should create drillDown callback for projects view", async () => {
      mockState.navigation.view = "projects" as View;
      mockState.navigation.selectedIdx = 0;
      mockVisibleItems = ["project1"];

      mockUseVisibleItems.mockReturnValue({
        visibleItems: mockVisibleItems,
      });

      const { useNavigationLogic } = await import(
        "../../hooks/useNavigationLogic"
      );
      const result = useNavigationLogic();

      result.drillDown();

      expect(mockDispatch).toHaveBeenCalledWith({ type: "RESET_NAVIGATION" });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "CLEAR_LOWER_LEVEL_SELECTIONS",
        payload: "projects",
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_PROJECTS",
        payload: new Set(["project1"]),
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_VIEW",
        payload: "apps",
      });
    });

    test("should not drill down when no item is selected", async () => {
      mockState.navigation.view = "clusters" as View;
      mockState.navigation.selectedIdx = 0;
      mockVisibleItems = [];

      mockUseVisibleItems.mockReturnValue({
        visibleItems: mockVisibleItems,
      });

      const { useNavigationLogic } = await import(
        "../../hooks/useNavigationLogic"
      );
      const result = useNavigationLogic();

      result.drillDown();

      expect(mockDispatch).not.toHaveBeenCalledWith({
        type: "RESET_NAVIGATION",
      });
    });

    test("should handle items that convert to strings properly", async () => {
      mockState.navigation.view = "clusters" as View;
      mockState.navigation.selectedIdx = 0;
      mockVisibleItems = [42]; // Number

      mockUseVisibleItems.mockReturnValue({
        visibleItems: mockVisibleItems,
      });

      const { useNavigationLogic } = await import(
        "../../hooks/useNavigationLogic"
      );
      const result = useNavigationLogic();

      result.drillDown();

      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SCOPE_CLUSTERS",
        payload: new Set(["42"]), // Should be converted to string
      });
    });
  });

  describe("toggleSelection callback logic", () => {
    test("should create toggleSelection callback for apps view", async () => {
      mockState.navigation.view = "apps" as View;
      mockState.navigation.selectedIdx = 0;
      mockState.selections.selectedApps = new Set(["existing-app"]);
      mockVisibleItems = [{ name: "test-app" } as AppItem];

      mockUseVisibleItems.mockReturnValue({
        visibleItems: mockVisibleItems,
      });

      const { useNavigationLogic } = await import(
        "../../hooks/useNavigationLogic"
      );
      const result = useNavigationLogic();

      result.toggleSelection();

      expect(mockDispatch).toHaveBeenCalledWith({
        type: "CLEAR_LOWER_LEVEL_SELECTIONS",
        payload: "apps",
      });
      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_APPS",
        payload: new Set(["existing-app", "test-app"]),
      });
    });

    test("should remove app from selection if already selected", async () => {
      mockState.navigation.view = "apps" as View;
      mockState.navigation.selectedIdx = 0;
      mockState.selections.selectedApps = new Set(["test-app", "other-app"]);
      mockVisibleItems = [{ name: "test-app" } as AppItem];

      mockUseVisibleItems.mockReturnValue({
        visibleItems: mockVisibleItems,
      });

      const { useNavigationLogic } = await import(
        "../../hooks/useNavigationLogic"
      );
      const result = useNavigationLogic();

      result.toggleSelection();

      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_APPS",
        payload: new Set(["other-app"]),
      });
    });

    test("should not toggle selection for non-apps views", async () => {
      const views: View[] = ["clusters", "namespaces", "projects"];

      for (const view of views) {
        mockDispatch.mockReset();
        mockState.navigation.view = view;
        mockState.navigation.selectedIdx = 0;
        mockVisibleItems = ["item1"];

        mockUseVisibleItems.mockReturnValue({
          visibleItems: mockVisibleItems,
        });

        const { useNavigationLogic } = await import(
          "../../hooks/useNavigationLogic"
        );
        const result = useNavigationLogic();

        result.toggleSelection();

        expect(mockDispatch).not.toHaveBeenCalledWith({
          type: "SET_SELECTED_APPS",
          payload: expect.any(Set),
        });
      }
    });

    test("should not toggle when no item selected", async () => {
      mockState.navigation.view = "apps" as View;
      mockState.navigation.selectedIdx = 0;
      mockVisibleItems = [];

      mockUseVisibleItems.mockReturnValue({
        visibleItems: mockVisibleItems,
      });

      const { useNavigationLogic } = await import(
        "../../hooks/useNavigationLogic"
      );
      const result = useNavigationLogic();

      result.toggleSelection();

      expect(mockDispatch).not.toHaveBeenCalledWith({
        type: "SET_SELECTED_APPS",
        payload: expect.any(Set),
      });
    });

    test("should handle app without name property", async () => {
      mockState.navigation.view = "apps" as View;
      mockState.navigation.selectedIdx = 0;
      mockState.selections.selectedApps = new Set();
      mockVisibleItems = [{ sync: "Synced" } as AppItem]; // Missing name

      mockUseVisibleItems.mockReturnValue({
        visibleItems: mockVisibleItems,
      });

      const { useNavigationLogic } = await import(
        "../../hooks/useNavigationLogic"
      );
      const result = useNavigationLogic();

      result.toggleSelection();

      expect(mockDispatch).toHaveBeenCalledWith({
        type: "SET_SELECTED_APPS",
        payload: new Set([undefined]), // Should handle undefined name
      });
    });
  });

  describe("return values", () => {
    test("should return expected interface", async () => {
      mockVisibleItems = ["item1", "item2"];

      mockUseVisibleItems.mockReturnValue({
        visibleItems: mockVisibleItems,
      });

      const { useNavigationLogic } = await import(
        "../../hooks/useNavigationLogic"
      );
      const result = useNavigationLogic();

      expect(result).toEqual({
        drillDown: expect.any(Function),
        toggleSelection: expect.any(Function),
        visibleItems: mockVisibleItems,
      });
    });

    test("should return the actual visibleItems from useVisibleItems", async () => {
      const testItems = [
        { name: "app1" } as AppItem,
        { name: "app2" } as AppItem,
      ];

      mockUseVisibleItems.mockReturnValue({
        visibleItems: testItems,
      });

      const { useNavigationLogic } = await import(
        "../../hooks/useNavigationLogic"
      );
      const result = useNavigationLogic();

      expect(result.visibleItems).toBe(testItems);
    });
  });

  describe("dependency arrays", () => {
    test("should pass correct dependencies to useEffect", async () => {
      mockVisibleItems = ["item1"];
      mockState.navigation.selectedIdx = 0;

      mockUseVisibleItems.mockReturnValue({
        visibleItems: mockVisibleItems,
      });

      const { useNavigationLogic } = await import(
        "../../hooks/useNavigationLogic"
      );
      useNavigationLogic();

      // Check that useEffect was called with dependency array
      const effectCall = mockUseEffect.mock.calls[0];
      expect(effectCall).toBeDefined();
      expect(effectCall[1]).toEqual([1, 0, mockDispatch]); // [visibleItems.length, selectedIdx, dispatch]
    });

    test("should pass correct dependencies to useCallback for drillDown", async () => {
      mockVisibleItems = ["item1"];
      mockState.navigation.selectedIdx = 0;
      mockState.navigation.view = "clusters" as View;

      mockUseVisibleItems.mockReturnValue({
        visibleItems: mockVisibleItems,
      });

      const { useNavigationLogic } = await import(
        "../../hooks/useNavigationLogic"
      );
      useNavigationLogic();

      // Check that useCallback was called for drillDown (first call)
      const callbackCall = mockUseCallback.mock.calls[0];
      expect(callbackCall).toBeDefined();
      expect(callbackCall[1]).toEqual([
        mockVisibleItems,
        mockState.navigation,
        mockDispatch,
      ]);
    });

    test("should pass correct dependencies to useCallback for toggleSelection", async () => {
      mockVisibleItems = ["item1"];
      mockState.navigation.selectedIdx = 0;

      mockUseVisibleItems.mockReturnValue({
        visibleItems: mockVisibleItems,
      });

      const { useNavigationLogic } = await import(
        "../../hooks/useNavigationLogic"
      );
      useNavigationLogic();

      // Check that useCallback was called for toggleSelection (second call)
      const callbackCall = mockUseCallback.mock.calls[1];
      expect(callbackCall).toBeDefined();
      expect(callbackCall[1]).toEqual([
        mockVisibleItems,
        mockState.navigation,
        mockState.selections,
        mockDispatch,
      ]);
    });
  });
});
