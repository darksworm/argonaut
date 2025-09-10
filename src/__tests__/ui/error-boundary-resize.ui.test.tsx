import { describe, expect, spyOn, test } from "bun:test";
import { render } from "ink-testing-library";
import React from "react";
import { ErrorBoundary } from "../../components/ErrorBoundary";

// Ensure the error boundary forces a terminal refresh so content is positioned correctly
// when an error occurs.
describe("ErrorBoundary terminal refresh", () => {
  test("triggers resize hack on mount", async () => {
    const stdout: any = process.stdout;
    const originalCols = stdout.columns;
    Object.defineProperty(stdout, "columns", { value: 80, writable: true });
    const emitSpy = spyOn(stdout, "emit");

    const Boom = () => {
      throw new Error("boom");
    };

    render(
      <ErrorBoundary>
        <Boom />
      </ErrorBoundary>,
    );

    // wait for the resize hack which uses setTimeout
    await new Promise((r) => setTimeout(r, 10));

    const resizeCalls = emitSpy.mock.calls.filter((c) => c[0] === "resize");
    expect(resizeCalls.length).toBeGreaterThanOrEqual(2);

    emitSpy.mockRestore();
    Object.defineProperty(stdout, "columns", {
      value: originalCols,
      writable: true,
    });
  });
});
