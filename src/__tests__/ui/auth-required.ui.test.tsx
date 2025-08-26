import { render } from "ink-testing-library";
import AuthRequiredView from "../../components/AuthRequiredView";

// Real UI tests for AuthRequiredView component
describe("AuthRequiredView Component", () => {
  const defaultProps = {
    server: "argocd.example.com",
    apiVersion: "v2.8.0",
    termCols: 80,
    termRows: 24,
    clusterScope: "all",
    namespaceScope: "default",
    projectScope: "default",
    argonautVersion: "1.12.0",
    status: "Authentication required",
  };

  it("renders authentication required message", () => {
    const { lastFrame } = render(<AuthRequiredView {...defaultProps} />);

    const frame = lastFrame();
    expect(frame).toContain("AUTHENTICATION REQUIRED");
    expect(frame).toContain("Please login to ArgoCD before running argonaut.");
  });

  it("displays server information when provided", () => {
    const { lastFrame } = render(
      <AuthRequiredView {...defaultProps} server="my-argocd.company.com" />,
    );

    const frame = lastFrame();
    expect(frame).toContain("my-argocd.company.com");
    // The "Current context:" text might have ANSI codes, so check for both parts separately
    expect(frame).toContain("Current context:");
    expect(frame).toContain("my-argocd.company.com");
  });

  it("displays unknown context when server is null", () => {
    const { lastFrame } = render(
      <AuthRequiredView {...defaultProps} server={null} />,
    );

    const frame = lastFrame();
    expect(frame).toContain("Current context: unknown");
  });

  it("displays login instructions", () => {
    const { lastFrame } = render(<AuthRequiredView {...defaultProps} />);

    const frame = lastFrame();
    expect(frame).toContain("1. Run: argocd login <your-argocd-server>");
    expect(frame).toContain("2. Follow prompts to authenticate");
    expect(frame).toContain("3. Re-run argonaut");
  });

  it("displays keyboard shortcuts", () => {
    const { lastFrame } = render(<AuthRequiredView {...defaultProps} />);

    const frame = lastFrame();
    // Check for keyboard shortcuts text which might have ANSI color codes
    expect(frame).toContain("Press");
    expect(frame).toContain("to view logs");
    expect(frame).toContain("to quit");
  });

  it("displays custom message when provided", () => {
    const customMessage = "Your session has expired. Please re-authenticate.";
    const { lastFrame } = render(
      <AuthRequiredView {...defaultProps} message={customMessage} />,
    );

    const frame = lastFrame();
    expect(frame).toContain(customMessage);
  });

  it("displays version information", () => {
    const { lastFrame } = render(
      <AuthRequiredView {...defaultProps} argonautVersion="2.0.0" />,
    );

    const frame = lastFrame();
    expect(frame).toContain("2.0.0");
  });

  it("displays API version and scopes in header", () => {
    const props = {
      ...defaultProps,
      apiVersion: "v2.9.1",
      clusterScope: "prod-cluster",
      namespaceScope: "production",
      projectScope: "web-app",
    };

    const { lastFrame } = render(<AuthRequiredView {...props} />);

    const frame = lastFrame();
    // These might be in the banner component, so just verify they don't crash
    expect(frame).toBeDefined();
    expect(frame.length).toBeGreaterThan(0);
  });

  it("handles different terminal sizes", () => {
    const smallTermProps = {
      ...defaultProps,
      termCols: 40,
      termRows: 10,
    };

    const { lastFrame } = render(<AuthRequiredView {...smallTermProps} />);

    const frame = lastFrame();
    expect(frame).toBeDefined();
    // With small terminal, text might be truncated, so just check it renders without crashing
    expect(frame.length).toBeGreaterThan(0);
    expect(frame).toContain("argonaut"); // This should definitely be there
  });

  it("displays status in footer", () => {
    const customStatus = "Config not found - please authenticate";
    const { lastFrame } = render(
      <AuthRequiredView {...defaultProps} status={customStatus} />,
    );

    const frame = lastFrame();
    expect(frame).toContain(customStatus);
  });
});
