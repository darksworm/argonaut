import { describe, expect, test } from "bun:test";
import { NavigationService } from "../../services/navigation-service";

describe("NavigationService", () => {
  describe("drillDown", () => {
    test("should drill down from clusters to namespaces", () => {
      const result = NavigationService.drillDown(
        "clusters",
        "test-cluster",
        ["test-cluster", "other-cluster"],
        0,
      );

      expect(result).toEqual({
        newView: "namespaces",
        scopeClusters: new Set(["test-cluster"]),
        shouldResetNavigation: true,
        shouldClearLowerLevelSelections: true,
      });
    });

    test("should drill down from namespaces to projects", () => {
      const result = NavigationService.drillDown(
        "namespaces",
        "test-namespace",
        ["test-namespace", "other-namespace"],
        0,
      );

      expect(result).toEqual({
        newView: "projects",
        scopeNamespaces: new Set(["test-namespace"]),
        shouldResetNavigation: true,
        shouldClearLowerLevelSelections: true,
      });
    });

    test("should drill down from projects to apps", () => {
      const result = NavigationService.drillDown(
        "projects",
        "test-project",
        ["test-project", "other-project"],
        0,
      );

      expect(result).toEqual({
        newView: "apps",
        scopeProjects: new Set(["test-project"]),
        shouldResetNavigation: true,
        shouldClearLowerLevelSelections: true,
      });
    });

    test("should return null when trying to drill down from apps", () => {
      const result = NavigationService.drillDown(
        "apps",
        "test-app",
        [{ name: "test-app" }],
        0,
      );

      expect(result).toBe(null);
    });

    test("should return null when selectedIdx is out of bounds", () => {
      const result = NavigationService.drillDown(
        "clusters",
        "test-cluster",
        ["test-cluster"],
        5, // out of bounds
      );

      expect(result).toBe(null);
    });
  });

  describe("toggleSelection", () => {
    test("should add app to selection when not selected", () => {
      const currentSelections = new Set<string>();
      const result = NavigationService.toggleSelection(
        "apps",
        { name: "test-app" },
        [{ name: "test-app" }],
        0,
        currentSelections,
      );

      expect(result).toEqual({
        selectedApps: new Set(["test-app"]),
      });
    });

    test("should remove app from selection when already selected", () => {
      const currentSelections = new Set(["test-app"]);
      const result = NavigationService.toggleSelection(
        "apps",
        { name: "test-app" },
        [{ name: "test-app" }],
        0,
        currentSelections,
      );

      expect(result).toEqual({
        selectedApps: new Set(),
      });
    });

    test("should return null when not in apps view", () => {
      const result = NavigationService.toggleSelection(
        "clusters",
        "test-cluster",
        ["test-cluster"],
        0,
        new Set(),
      );

      expect(result).toBe(null);
    });

    test("should return null when selectedIdx is out of bounds", () => {
      const result = NavigationService.toggleSelection(
        "apps",
        { name: "test-app" },
        [{ name: "test-app" }],
        5, // out of bounds
        new Set(),
      );

      expect(result).toBe(null);
    });
  });

  describe("validateBounds", () => {
    test("should return selectedIdx when within bounds", () => {
      const result = NavigationService.validateBounds(2, 5);
      expect(result).toBe(2);
    });

    test("should return max index when selectedIdx is too high", () => {
      const result = NavigationService.validateBounds(10, 5);
      expect(result).toBe(4); // max index for 5 items is 4
    });

    test("should return 0 when selectedIdx is negative", () => {
      const result = NavigationService.validateBounds(-1, 5);
      expect(result).toBe(0);
    });

    test("should return 0 when item count is 0", () => {
      const result = NavigationService.validateBounds(5, 0);
      expect(result).toBe(0);
    });
  });

  describe("helper methods", () => {
    test("canDrillDown should return correct values", () => {
      expect(NavigationService.canDrillDown("clusters")).toBe(true);
      expect(NavigationService.canDrillDown("namespaces")).toBe(true);
      expect(NavigationService.canDrillDown("projects")).toBe(true);
      expect(NavigationService.canDrillDown("apps")).toBe(false);
    });

    test("canToggleSelection should return correct values", () => {
      expect(NavigationService.canToggleSelection("clusters")).toBe(false);
      expect(NavigationService.canToggleSelection("namespaces")).toBe(false);
      expect(NavigationService.canToggleSelection("projects")).toBe(false);
      expect(NavigationService.canToggleSelection("apps")).toBe(true);
    });

    test("getNextView should return correct next view", () => {
      expect(NavigationService.getNextView("clusters")).toBe("namespaces");
      expect(NavigationService.getNextView("namespaces")).toBe("projects");
      expect(NavigationService.getNextView("projects")).toBe("apps");
      expect(NavigationService.getNextView("apps")).toBe(null);
    });

    test("getPreviousView should return correct previous view", () => {
      expect(NavigationService.getPreviousView("apps")).toBe("projects");
      expect(NavigationService.getPreviousView("projects")).toBe("namespaces");
      expect(NavigationService.getPreviousView("namespaces")).toBe("clusters");
      expect(NavigationService.getPreviousView("clusters")).toBe(null);
    });

    test("clearAllSelections should return empty sets", () => {
      const result = NavigationService.clearAllSelections();
      expect(result).toEqual({
        scopeClusters: new Set(),
        scopeNamespaces: new Set(),
        scopeProjects: new Set(),
        selectedApps: new Set(),
      });
    });

    test("clearFilters should return empty strings", () => {
      const result = NavigationService.clearFilters();
      expect(result).toEqual({
        activeFilter: "",
        searchQuery: "",
      });
    });
  });
});
