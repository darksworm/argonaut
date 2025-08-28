import { execa } from "execa";
import packageJson from "../../package.json";

describe("CLI version flag", () => {
  it("prints version with --version", async () => {
    const { stdout, exitCode } = await execa("bun", [
      "src/main.tsx",
      "--version",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout.trim()).toBe(packageJson.version);
  });

  it("prints version with -v", async () => {
    const { stdout, exitCode } = await execa("bun", ["src/main.tsx", "-v"]);
    expect(exitCode).toBe(0);
    expect(stdout.trim()).toBe(packageJson.version);
  });
});
