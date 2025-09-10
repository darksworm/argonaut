import { describe, expect, test } from "bun:test";
import {
  capitalize,
  colorFor,
  formatBytes,
  formatNumber,
  humanizeSince,
  shortSha,
  singleLine,
  truncate,
} from "../../utils/formatters";

describe("formatters", () => {
  describe("colorFor", () => {
    test("should return green for healthy/synced states", () => {
      expect(colorFor("Synced")).toEqual({ color: "green" });
      expect(colorFor("Healthy")).toEqual({ color: "green" });
      expect(colorFor("synced")).toEqual({ color: "green" });
      expect(colorFor("healthy")).toEqual({ color: "green" });
    });

    test("should return red for degraded/outofsync states", () => {
      expect(colorFor("OutOfSync")).toEqual({ color: "red" });
      expect(colorFor("Degraded")).toEqual({ color: "red" });
      expect(colorFor("outofsync")).toEqual({ color: "red" });
      expect(colorFor("degraded")).toEqual({ color: "red" });
    });

    test("should return yellow for warning states", () => {
      expect(colorFor("Progressing")).toEqual({ color: "yellow" });
      expect(colorFor("Warning")).toEqual({ color: "yellow" });
      expect(colorFor("Suspicious")).toEqual({ color: "yellow" });
      expect(colorFor("progressing")).toEqual({ color: "yellow" });
    });

    test("should return dimColor for unknown", () => {
      expect(colorFor("Unknown")).toEqual({ dimColor: true });
      expect(colorFor("unknown")).toEqual({ dimColor: true });
    });

    test("should return empty object for unrecognized states", () => {
      expect(colorFor("SomeRandomState")).toEqual({});
      expect(colorFor("")).toEqual({});
    });
  });

  describe("humanizeSince", () => {
    const now = Date.now();

    test("should return dash for undefined/null", () => {
      expect(humanizeSince()).toBe("—");
      expect(humanizeSince("")).toBe("—");
    });

    test("should return dash for invalid date", () => {
      expect(humanizeSince("invalid-date")).toBe("—");
    });

    test("should format seconds correctly", () => {
      const fiveSecondsAgo = new Date(now - 5000).toISOString();
      expect(humanizeSince(fiveSecondsAgo)).toBe("5s");
    });

    test("should format minutes correctly", () => {
      const threeMinutesAgo = new Date(now - 180000).toISOString(); // 3 minutes
      expect(humanizeSince(threeMinutesAgo)).toBe("3m");
    });

    test("should format hours correctly", () => {
      const twoHoursAgo = new Date(now - 7200000).toISOString(); // 2 hours
      expect(humanizeSince(twoHoursAgo)).toBe("2h");
    });

    test("should format days correctly", () => {
      const fiveDaysAgo = new Date(now - 432000000).toISOString(); // 5 days
      expect(humanizeSince(fiveDaysAgo)).toBe("5d");
    });
  });

  describe("shortSha", () => {
    test("should return first 7 characters of SHA", () => {
      expect(shortSha("abcdef1234567890")).toBe("abcdef1");
    });

    test("should handle short strings", () => {
      expect(shortSha("abc")).toBe("abc");
    });

    test("should handle empty/undefined", () => {
      expect(shortSha()).toBe("");
      expect(shortSha("")).toBe("");
    });
  });

  describe("singleLine", () => {
    test("should convert newlines to spaces", () => {
      expect(singleLine("line1\nline2\nline3")).toBe("line1 line2 line3");
    });

    test("should convert tabs to spaces", () => {
      expect(singleLine("word1\tword2\tword3")).toBe("word1 word2 word3");
    });

    test("should collapse multiple spaces", () => {
      expect(singleLine("word1   word2     word3")).toBe("word1 word2 word3");
    });

    test("should trim whitespace", () => {
      expect(singleLine("  text  ")).toBe("text");
    });

    test("should handle complex whitespace", () => {
      expect(singleLine("  line1\n\n  line2\t\t  line3  ")).toBe(
        "line1 line2 line3",
      );
    });
  });

  describe("formatBytes", () => {
    test("should format 0 bytes", () => {
      expect(formatBytes(0)).toBe("0 B");
    });

    test("should format bytes", () => {
      expect(formatBytes(500)).toBe("500 B");
    });

    test("should format kilobytes", () => {
      expect(formatBytes(1024)).toBe("1 KB");
      expect(formatBytes(1536)).toBe("1.5 KB"); // 1.5 KB
    });

    test("should format megabytes", () => {
      expect(formatBytes(1048576)).toBe("1 MB"); // 1024^2
    });

    test("should format gigabytes", () => {
      expect(formatBytes(1073741824)).toBe("1 GB"); // 1024^3
    });
  });

  describe("formatNumber", () => {
    test("should format numbers with thousands separator", () => {
      expect(formatNumber(1000)).toBe("1,000");
      expect(formatNumber(1234567)).toBe("1,234,567");
    });

    test("should handle small numbers", () => {
      expect(formatNumber(42)).toBe("42");
    });
  });

  describe("truncate", () => {
    test("should not truncate short text", () => {
      expect(truncate("hello", 10)).toBe("hello");
    });

    test("should truncate long text with ellipsis", () => {
      expect(truncate("this is a very long text", 10)).toBe("this is..."); // 10-3=7 chars + "..."
    });

    test("should handle exact length", () => {
      expect(truncate("exactly10!", 10)).toBe("exactly10!");
    });
  });

  describe("capitalize", () => {
    test("should capitalize first letter", () => {
      expect(capitalize("hello")).toBe("Hello");
    });

    test("should handle uppercase input", () => {
      expect(capitalize("HELLO")).toBe("Hello");
    });

    test("should handle mixed case", () => {
      expect(capitalize("hELLo")).toBe("Hello");
    });

    test("should handle empty string", () => {
      expect(capitalize("")).toBe("");
    });
  });
});
