import { render } from "ink-testing-library";
import AuthRequiredView from "../../components/AuthRequiredView";
import { LoadingView } from "../../components/views";
import { AppStateProvider } from "../../contexts/AppStateContext";

// Integration tests for app states and transitions
// Since we can't easily mock the full app initialization with Bun test,
// we'll test the main components and state transitions directly

describe("App Integration UI Tests", () => {
  describe("App State Rendering", () => {
    it("renders loading view correctly", () => {
      const { lastFrame } = render(
        <AppStateProvider>
          <LoadingView />
        </AppStateProvider>,
      );

      const frame = lastFrame();
      expect(frame.toLowerCase()).toContain("loading");
    });

    it("renders auth required view with app state context", () => {
      const mockServer = "argocd.example.com";
      const props = {
        server: mockServer,
        apiVersion: "v2.8.0",
        termCols: 80,
        termRows: 24,
        clusterScope: "all",
        namespaceScope: "default",
        projectScope: "default",
        argonautVersion: "1.12.0",
        message: "Authentication required",
        status: "Config not found",
      };

      const { lastFrame } = render(
        <AppStateProvider>
          <AuthRequiredView {...props} />
        </AppStateProvider>,
      );

      const frame = lastFrame();
      expect(frame).toContain("AUTHENTICATION REQUIRED");
      expect(frame).toContain(mockServer);
      expect(frame).toContain("Config not found");
    });

    it("handles auth required view with no server", () => {
      const props = {
        server: null,
        apiVersion: "v2.8.0",
        termCols: 80,
        termRows: 24,
        clusterScope: "all",
        namespaceScope: "default",
        projectScope: "default",
        argonautVersion: "1.12.0",
        message: "No configuration found",
        status: "Please authenticate",
      };

      const { lastFrame } = render(
        <AppStateProvider>
          <AuthRequiredView {...props} />
        </AppStateProvider>,
      );

      const frame = lastFrame();
      expect(frame).toContain("AUTHENTICATION REQUIRED");
      expect(frame).toContain("No configuration found");
      expect(frame).toContain("Current context: unknown");
    });
  });

  describe("Real-world Scenarios", () => {
    it("displays appropriate message for missing config file", () => {
      const props = {
        server: null,
        apiVersion: "",
        termCols: 80,
        termRows: 24,
        clusterScope: "",
        namespaceScope: "",
        projectScope: "",
        argonautVersion: "1.12.0",
        message:
          "Could not load Argo CD config. Please run 'argocd login' to configure and authenticate. Config file not found",
        status: "Config file not found",
      };

      const { lastFrame } = render(<AuthRequiredView {...props} />);

      const frame = lastFrame();
      expect(frame).toContain("AUTHENTICATION REQUIRED");
      expect(frame).toContain("Could not load Argo CD config");
      expect(frame).toContain("argocd login");
    });

    it("displays appropriate message for expired token", () => {
      const props = {
        server: "argocd.company.com",
        apiVersion: "v2.8.0",
        termCols: 80,
        termRows: 24,
        clusterScope: "prod",
        namespaceScope: "default",
        projectScope: "web-app",
        argonautVersion: "1.12.0",
        message:
          "No auth token found for user 'admin'. Please run 'argocd login' to authenticate.",
        status: "Token expired",
      };

      const { lastFrame } = render(<AuthRequiredView {...props} />);

      const frame = lastFrame();
      expect(frame).toContain("AUTHENTICATION REQUIRED");
      expect(frame).toContain("No auth token found");
      expect(frame).toContain("argocd.company.com");
    });

    it("displays appropriate message for invalid token", () => {
      const props = {
        server: "argocd.internal.com",
        apiVersion: "v2.9.1",
        termCols: 120,
        termRows: 30,
        clusterScope: "staging",
        namespaceScope: "test",
        projectScope: "api",
        argonautVersion: "1.12.0",
        message: "user info fetch fail: Invalid token",
        status: "Authentication failed",
      };

      const { lastFrame } = render(<AuthRequiredView {...props} />);

      const frame = lastFrame();
      expect(frame).toContain("AUTHENTICATION REQUIRED");
      expect(frame).toContain("user info fetch fail: Invalid token");
      expect(frame).toContain("argocd.internal.com");
    });

    it("displays appropriate message for connection timeout", () => {
      const props = {
        server: "argocd.remote.com",
        apiVersion: "",
        termCols: 80,
        termRows: 24,
        clusterScope: "prod",
        namespaceScope: "default",
        projectScope: "frontend",
        argonautVersion: "1.12.0",
        message: "Could not connect to Argo CD! Connection timed out.",
        status: "Connection timeout",
      };

      const { lastFrame } = render(<AuthRequiredView {...props} />);

      const frame = lastFrame();
      expect(frame).toContain("AUTHENTICATION REQUIRED");
      expect(frame).toContain("Could not connect to Argo CD");
      expect(frame).toContain("Connection timed out");
    });
  });

  describe("UI Layout and Responsiveness", () => {
    it("handles wide terminal gracefully", () => {
      const props = {
        server: "argocd.example.com",
        apiVersion: "v2.8.0",
        termCols: 160,
        termRows: 40,
        clusterScope: "production",
        namespaceScope: "web-services",
        projectScope: "ecommerce-platform",
        argonautVersion: "1.12.0",
        message: "Authentication required",
        status: "Please authenticate",
      };

      const { lastFrame } = render(<AuthRequiredView {...props} />);

      const frame = lastFrame();
      expect(frame).toBeDefined();
      expect(frame).toContain("AUTHENTICATION REQUIRED");
      expect(frame).toContain("argocd.example.com");
    });

    it("handles narrow terminal gracefully", () => {
      const props = {
        server: "argocd.dev",
        apiVersion: "v2.8.0",
        termCols: 60,
        termRows: 15,
        clusterScope: "dev",
        namespaceScope: "test",
        projectScope: "api",
        argonautVersion: "1.12.0",
        message: "Auth required",
        status: "No token",
      };

      const { lastFrame } = render(<AuthRequiredView {...props} />);

      const frame = lastFrame();
      expect(frame).toBeDefined();
      expect(frame.length).toBeGreaterThan(0);
      // Don't check for specific text as it might be truncated
    });

    it("handles tall terminal gracefully", () => {
      const props = {
        server: "argocd.example.com",
        apiVersion: "v2.8.0",
        termCols: 80,
        termRows: 50,
        clusterScope: "all",
        namespaceScope: "default",
        projectScope: "default",
        argonautVersion: "1.12.0",
        message: "Authentication required",
        status: "Please authenticate",
      };

      const { lastFrame } = render(<AuthRequiredView {...props} />);

      const frame = lastFrame();
      expect(frame).toBeDefined();
      expect(frame).toContain("AUTHENTICATION REQUIRED");
    });
  });

  describe("Content Validation", () => {
    it("includes all required elements", () => {
      const props = {
        server: "test.argocd.com",
        apiVersion: "v2.8.0",
        termCols: 80,
        termRows: 24,
        clusterScope: "test",
        namespaceScope: "default",
        projectScope: "test-app",
        argonautVersion: "1.12.0",
        message: "Please authenticate to continue",
        status: "Auth required",
      };

      const { lastFrame } = render(<AuthRequiredView {...props} />);

      const frame = lastFrame();

      // Check for all required UI elements
      expect(frame).toContain("AUTHENTICATION REQUIRED");
      expect(frame).toContain("Please authenticate to continue");
      expect(frame).toContain("1. Run: argocd login");
      expect(frame).toContain("2. Follow prompts to authenticate");
      expect(frame).toContain("3. Re-run argonaut");
      // Check for server context - the "Current context:" text has ANSI codes
      expect(frame).toContain("Current context:");
      expect(frame).toContain("test.argocd.com");
      // Check for keyboard shortcuts - may have ANSI color codes around 'l' and 'q'
      expect(frame).toContain("Press");
      expect(frame).toContain("to view logs");
      expect(frame).toContain("to quit");
      expect(frame).toContain("Auth required"); // status
      expect(frame).toContain("1.12.0"); // version
    });

    it("properly formats header information", () => {
      const props = {
        server: "prod.argocd.io",
        apiVersion: "v2.9.0",
        termCols: 80,
        termRows: 24,
        clusterScope: "production",
        namespaceScope: "web-app",
        projectScope: "frontend",
        argonautVersion: "2.0.0",
        message: "Token expired",
        status: "Please re-authenticate",
      };

      const { lastFrame } = render(<AuthRequiredView {...props} />);

      const frame = lastFrame();

      // Check header format (from the banner)
      expect(frame).toContain("AUTH REQUIRED");
      expect(frame).toContain("prod.argocd.io");
    });
  });
});
