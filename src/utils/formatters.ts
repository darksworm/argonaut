/**
 * Pure formatting utility functions
 * Extracted from utils.tsx for better organization and Go migration
 */

/**
 * Get color styling for app state
 */
export function colorFor(appState: string): {
  color?: any;
  dimColor?: boolean;
} {
  const v = (appState || "").toLowerCase();
  if (v === "synced" || v === "healthy") return { color: "green" };
  if (v === "outofsync" || v === "degraded") return { color: "red" };
  if (v === "progressing" || v === "warning" || v === "suspicious")
    return { color: "yellow" };
  if (v === "unknown") return { dimColor: true };
  return {};
}

/**
 * Convert ISO date to human-readable "time since" format
 */
export function humanizeSince(iso?: string): string {
  if (!iso) return "—";
  const t = new Date(iso).getTime();
  if (!Number.isFinite(t)) return "—";
  const s = Math.max(0, Math.floor((Date.now() - t) / 1000));
  if (s < 60) return `${s}s`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h`;
  const d = Math.floor(h / 24);
  if (d < 30) return `${d}d`;
  const mo = Math.floor(d / 30);
  if (mo < 12) return `${mo}mo`;
  const y = Math.floor(mo / 12);
  return `${y}y`;
}

/**
 * Shorten SHA to first 7 characters
 */
export function shortSha(s?: string): string {
  return (s || "").slice(0, 7);
}

/**
 * Convert multiline text to single line
 */
export function singleLine(input?: string): string {
  const s = String(input || "");
  // Replace newlines/tabs with spaces and collapse multiple spaces
  return s
    .replace(/[\r\n\t]+/g, " ")
    .replace(/\s{2,}/g, " ")
    .trim();
}

/**
 * Format file size in human-readable format
 */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / k ** i).toFixed(1))} ${sizes[i]}`;
}

/**
 * Format number with thousands separator
 */
export function formatNumber(num: number): string {
  return num.toLocaleString();
}

/**
 * Truncate text to maximum length with ellipsis
 */
export function truncate(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text;
  return `${text.slice(0, maxLength - 3)}...`;
}

/**
 * Capitalize first letter of string
 */
export function capitalize(str: string): string {
  if (!str) return str;
  return str.charAt(0).toUpperCase() + str.slice(1).toLowerCase();
}
