import { Box, Text } from "ink";
import type React from "react";
import stringWidth from "string-width";
import { useAppState } from "../../contexts/AppStateContext";
import type { AppItem } from "../../types/domain";
import { colorFor } from "../../utils";

const COL = {
  mark: 2,
  name: 36,
  sync: 4,
  health: 6,
} as const;

const GUTTER = 1;
const Sep: React.FC = () => <Box width={GUTTER} />;

interface ListViewProps {
  visibleItems: any[];
  availableRows: number;
}

export const ListView: React.FC<ListViewProps> = ({
  visibleItems,
  availableRows,
}) => {
  const { state } = useAppState();
  const { navigation, selections, terminal } = state;
  const { view, selectedIdx } = navigation;
  const { scopeClusters, scopeNamespaces, scopeProjects, selectedApps } =
    selections;

  // Calculate visible slice
  const listRows = Math.max(0, availableRows);
  const start = Math.max(
    0,
    Math.min(
      Math.max(0, selectedIdx - Math.floor(listRows / 2)),
      Math.max(0, visibleItems.length - listRows),
    ),
  );
  const end = Math.min(visibleItems.length, start + listRows);
  const rowsSlice = visibleItems.slice(start, end);

  // Layout calculations for apps view
  const MIN_NAME = 12;
  const ASCII_ICONS = {
    check: "V",
    warn: "!",
    quest: "?",
    delta: "^",
  } as const;

  const SYNC_LABEL: Record<string, string> = {
    Synced: "Synced",
    OutOfSync: "OutOfSync",
    Unknown: "Unknown",
    Degraded: "Degraded",
  };

  const HEALTH_LABEL: Record<string, string> = {
    Healthy: "Healthy",
    Missing: "Missing",
    Degraded: "Degraded",
    Progressing: "Progressing",
    Unknown: "Unknown",
  };

  const rightPadTo = (s: string, width: number) => {
    const w = stringWidth(s);
    return w >= width ? s : " ".repeat(width - w) + s;
  };

  const SYNC_WIDE = 11;
  const HEALTH_WIDE = 14;
  const overhead = 6;
  const fixedNoLastWide = SYNC_WIDE + GUTTER + HEALTH_WIDE;

  const canShowLabels = (cols: number) =>
    cols >= MIN_NAME + fixedNoLastWide + 2 * GUTTER + overhead + 15;

  const SYNC_COL = canShowLabels(terminal.cols) ? SYNC_WIDE : COL.sync;
  const HEALTH_COL = canShowLabels(terminal.cols) ? HEALTH_WIDE : COL.health;

  const SYNC_ICON_ASCII: Record<string, string> = {
    Synced: ASCII_ICONS.check,
    OutOfSync: ASCII_ICONS.delta,
    Unknown: ASCII_ICONS.quest,
    Degraded: ASCII_ICONS.warn,
  };

  const HEALTH_ICON_ASCII: Record<string, string> = {
    Healthy: ASCII_ICONS.check,
    Missing: ASCII_ICONS.quest,
    Degraded: ASCII_ICONS.warn,
    Progressing: ".",
    Unknown: ASCII_ICONS.quest,
  };

  const getSyncIcon = (s: string) => SYNC_ICON_ASCII[s];
  const getHealthIcon = (h: string) =>
    HEALTH_ICON_ASCII[h] ?? ASCII_ICONS.quest;

  return (
    <Box flexDirection="column">
      {/* Header row */}
      <Box width="100%">
        <Box flexGrow={1} flexShrink={1} minWidth={0}>
          <Text bold color="yellowBright" wrap="truncate">
            NAME
          </Text>
        </Box>

        {view === "apps" && (
          <>
            <Sep />
            <Box width={SYNC_COL} justifyContent="flex-end">
              <Text bold color="yellowBright" wrap="truncate">
                SYNC
              </Text>
            </Box>
            <Sep />
            <Box width={HEALTH_COL} justifyContent="flex-end">
              <Text bold color="yellowBright" wrap="truncate">
                HEALTH
              </Text>
            </Box>
          </>
        )}
      </Box>

      {/* Data rows */}
      {rowsSlice.map((it: any, i: number) => {
        const actualIndex = start + i;
        const isCursor = actualIndex === selectedIdx;

        if (view === "apps") {
          const a = it as AppItem;
          const isChecked = selectedApps.has(a.name);
          const active = isCursor || isChecked;

          return (
            <Box
              key={a.name}
              width="100%"
              backgroundColor={active ? "magentaBright" : undefined}
              flexWrap="nowrap"
              justifyContent="flex-start"
            >
              <Box flexGrow={1} flexShrink={1} minWidth={0}>
                <Text wrap="truncate-end">{a.name}</Text>
              </Box>

              <Sep />
              <Box width={SYNC_COL} flexShrink={0} justifyContent="flex-end">
                <Text
                  color={colorFor(a.sync).color as any}
                  dimColor={colorFor(a.sync).dimColor as any}
                >
                  {rightPadTo(
                    canShowLabels(terminal.cols)
                      ? `${getSyncIcon(a.sync)} ${SYNC_LABEL[a.sync] ?? ""}`
                      : `${getSyncIcon(a.sync)}`,
                    SYNC_COL,
                  )}
                </Text>
              </Box>

              <Sep />
              <Box width={HEALTH_COL} flexShrink={0} justifyContent="flex-end">
                <Text
                  color={colorFor(a.health).color as any}
                  dimColor={colorFor(a.health).dimColor as any}
                >
                  {rightPadTo(
                    canShowLabels(terminal.cols)
                      ? `${getHealthIcon(a.health)} ${HEALTH_LABEL[a.health] ?? ""}`
                      : `${getHealthIcon(a.health)}`,
                    HEALTH_COL,
                  )}
                </Text>
              </Box>
            </Box>
          );
        }

        // Non-apps views (clusters/namespaces/projects)
        const label = String(it);
        const isChecked =
          (view === "clusters" && scopeClusters.has(label)) ||
          (view === "namespaces" && scopeNamespaces.has(label)) ||
          (view === "projects" && scopeProjects.has(label));
        const active = isCursor || isChecked;

        return (
          <Box
            key={label}
            width="100%"
            backgroundColor={active ? "magentaBright" : undefined}
          >
            <Box flexGrow={1} flexShrink={1} minWidth={0}>
              <Text wrap="truncate-end">{label}</Text>
            </Box>
          </Box>
        );
      })}

      {visibleItems.length === 0 && (
        <Box paddingY={1} paddingX={2}>
          <Text dimColor>No items.</Text>
        </Box>
      )}
    </Box>
  );
};
