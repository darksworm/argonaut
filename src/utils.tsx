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

export function uniqueSorted<T>(arr: T[]): T[] {
  return Array.from(new Set(arr)).sort((a: any, b: any) =>
    `${a}`.localeCompare(`${b}`),
  );
}

export function fmtScope(set: Set<string>, max = 2): string {
  if (!set.size) return "—";
  const arr = Array.from(set);
  if (arr.length <= max) return arr.join(",");
  return `${arr.slice(0, max).join(",")} (+${arr.length - max})`;
}

export function shortSha(s?: string) {
  return (s || "").slice(0, 7);
}

export function singleLine(input?: string): string {
  const s = String(input || "");
  // Replace newlines/tabs with spaces and collapse multiple spaces
  return s
    .replace(/[\r\n\t]+/g, " ")
    .replace(/\s{2,}/g, " ")
    .trim();
}
