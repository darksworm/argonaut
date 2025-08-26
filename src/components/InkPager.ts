import { rawStdoutWrite } from "../ink-control";

export interface InkPagerOptions {
  title?: string;
  searchEnabled?: boolean;
}

// Reusable Ink-based pager with scroll and search
export async function showInkPager(
  content: string,
  options: InkPagerOptions = {},
): Promise<void> {
  const lines = content.split("\n");
  const terminalRows = (process.stdout as any)?.rows || 24;
  // Use full terminal height minus space for title and status line
  const titleLines = options.title ? 2 : 0; // title + empty line
  let maxRows = Math.max(8, terminalRows - titleLines - 1); // -1 for status line

  let topLine = 0;
  let searchTerm = "";
  let searchMatches: number[] = [];
  let currentMatch = 0;

  const updateSearch = (term: string) => {
    searchTerm = term;
    searchMatches = [];
    if (term) {
      lines.forEach((line, idx) => {
        if (line.toLowerCase().includes(term.toLowerCase())) {
          searchMatches.push(idx);
        }
      });
    }
    currentMatch = 0;
  };

  const render = () => {
    // Clear screen and move to top
    rawStdoutWrite("\x1b[2J\x1b[H");

    // Show title if provided
    if (options.title) {
      rawStdoutWrite(`\x1b[1m${options.title}\x1b[0m\n\n`);
    }

    const visibleLines = lines.slice(topLine, topLine + maxRows);

    // Render content lines
    for (let i = 0; i < maxRows; i++) {
      if (i < visibleLines.length) {
        const line = visibleLines[i];
        const lineNum = topLine + i;
        let displayLine = line;

        // Highlight search matches if search is enabled
        if (
          options.searchEnabled !== false &&
          searchTerm &&
          line.toLowerCase().includes(searchTerm.toLowerCase())
        ) {
          const isCurrentMatch = searchMatches[currentMatch] === lineNum;
          const highlightColor = isCurrentMatch
            ? "\x1b[43m\x1b[30m"
            : "\x1b[33m"; // Yellow bg for current, yellow text for others
          const regex = new RegExp(
            `(${searchTerm.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")})`,
            "gi",
          );
          displayLine = line.replace(regex, `${highlightColor}$1\x1b[0m`);
        }

        rawStdoutWrite(`${displayLine}\n`);
      } else {
        // Pad with empty lines to fill the screen
        rawStdoutWrite("\n");
      }
    }

    // Status line
    const totalLines = lines.length;
    const progress =
      totalLines > 0 ? Math.round((topLine / totalLines) * 100) : 0;
    let statusLine = `Lines ${topLine + 1}-${Math.min(topLine + maxRows, totalLines)} of ${totalLines} (${progress}%)`;

    if (options.searchEnabled !== false && searchTerm) {
      statusLine += ` | Search: "${searchTerm}" (${searchMatches.length} matches)`;
      if (searchMatches.length > 0) {
        statusLine += ` [${currentMatch + 1}/${searchMatches.length}]`;
      }
    }

    const controls =
      options.searchEnabled !== false
        ? " | q:quit j:down k:up space:page-down b:page-up /:search n:next N:prev g:top G:bottom"
        : " | q:quit j:down k:up space:page-down b:page-up g:top G:bottom";
    statusLine += controls;

    rawStdoutWrite(
      `\x1b[7m${statusLine.padEnd((process.stdout as any)?.columns || 80)}\x1b[0m`,
    );
  };

  process.emit("external-enter");
  // hack - wait a tick to let Ink unmount and clear the screen
  await new Promise((resolve) => setTimeout(resolve, 0));

  return new Promise<void>((resolve, reject) => {
    const stdin = process.stdin;
    let cleanedUp = false;

    // Centralized cleanup function
    let cleanup = () => {
      if (cleanedUp) return;
      cleanedUp = true;

      stdin.removeListener("data", handleInput);
      process.stdout.off("resize", onResize);
      try {
        (stdin as any).setRawMode?.(false);
        stdin.pause();
      } catch {}
      // Don't clear screen - let Ink handle the redraw

      // Hand stdin back to Ink
      process.emit("external-exit");
    };

    try {
      (stdin as any).setRawMode?.(true);
      stdin.resume();
    } catch (error) {
      cleanup();
      reject(error);
      return;
    }

    let inputBuffer = "";
    let inSearchMode = false;

    // Handle terminal resize
    const updateSize = () => {
      const newTerminalRows = (process.stdout as any)?.rows || 24;
      const newTitleLines = options.title ? 2 : 0;
      const newMaxRows = Math.max(8, newTerminalRows - newTitleLines - 1);

      // Update maxRows for the current session
      if (newMaxRows !== maxRows) {
        maxRows = newMaxRows;
        render();
      }
    };

    const onResize = () => updateSize();
    process.stdout.on("resize", onResize);

    const handleInput = (chunk: Buffer) => {
      try {
        const key = chunk.toString();

        if (inSearchMode && options.searchEnabled !== false) {
          if (key === "\r" || key === "\n") {
            // Enter - finish search
            inSearchMode = false;
            updateSearch(inputBuffer);
            inputBuffer = "";
            if (searchMatches.length > 0) {
              topLine = Math.max(0, searchMatches[0] - Math.floor(maxRows / 2));
            }
            render();
          } else if (key === "\x1b") {
            // Escape - cancel search
            inSearchMode = false;
            inputBuffer = "";
            render();
          } else if (key === "\x7f" || key === "\b") {
            // Backspace in search
            inputBuffer = inputBuffer.slice(0, -1);
            rawStdoutWrite(`\x1b[2K\rSearch: ${inputBuffer}`);
          } else if (key.length === 1 && key >= " ") {
            // Regular character in search
            inputBuffer += key;
            rawStdoutWrite(`\x1b[2K\rSearch: ${inputBuffer}`);
          }
          return;
        }

        switch (key) {
          case "q":
          case "Q":
            cleanup();
            resolve();
            break;
          case "j":
          case "\x1b[B": // Down arrow
            topLine = Math.min(
              topLine + 1,
              Math.max(0, lines.length - maxRows),
            );
            render();
            break;
          case "k":
          case "\x1b[A": // Up arrow
            topLine = Math.max(topLine - 1, 0);
            render();
            break;
          case " ":
          case "\x1b[6~": // Page Down
            topLine = Math.min(
              topLine + maxRows,
              Math.max(0, lines.length - maxRows),
            );
            render();
            break;
          case "b":
          case "\x1b[5~": // Page Up
            topLine = Math.max(topLine - maxRows, 0);
            render();
            break;
          case "/":
            if (options.searchEnabled !== false) {
              inSearchMode = true;
              inputBuffer = "";
              rawStdoutWrite("\x1b[2K\rSearch: ");
            }
            break;
          case "n":
            if (options.searchEnabled !== false && searchMatches.length > 0) {
              currentMatch = (currentMatch + 1) % searchMatches.length;
              topLine = Math.max(
                0,
                searchMatches[currentMatch] - Math.floor(maxRows / 2),
              );
              render();
            }
            break;
          case "N":
            if (options.searchEnabled !== false && searchMatches.length > 0) {
              currentMatch =
                (currentMatch - 1 + searchMatches.length) %
                searchMatches.length;
              topLine = Math.max(
                0,
                searchMatches[currentMatch] - Math.floor(maxRows / 2),
              );
              render();
            }
            break;
          case "g":
            topLine = 0;
            render();
            break;
          case "G":
            topLine = Math.max(0, lines.length - maxRows);
            render();
            break;
        }
      } catch (error) {
        cleanup();
        reject(error);
      }
    };

    // Handle process termination signals
    const signalHandler = () => {
      cleanup();
      process.exit(0);
    };

    process.on("SIGINT", signalHandler);
    process.on("SIGTERM", signalHandler);

    // Clean up signal handlers when done
    const originalCleanup = cleanup;
    cleanup = () => {
      process.off("SIGINT", signalHandler);
      process.off("SIGTERM", signalHandler);
      originalCleanup();
    };

    stdin.on("data", handleInput);
    render();
  });
}
