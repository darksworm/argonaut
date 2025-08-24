import { Box, Text, useInput } from "ink";
import React, { useEffect, useState } from "react";
import {
  DESK_HEIGHT,
  DESK_WIDTH,
  getPaperclipInventory,
  STAPLER_SPEED,
} from "./PaperclipDatabase";

export const rulerLineMode = String.fromCharCode(
  105,
  108,
  105,
  107,
  101,
  97,
  114,
  103,
  111,
  110,
  97,
  117,
  116,
);

interface OfficeSupplyProps {
  onExit: () => void;
}

// Pre-load office supplies at module level for instant access
let cachedSupplies: string[] | null = null;

const loadSupplies = (): string[] => {
  if (cachedSupplies === null) {
    try {
      console.log("Loading office supplies...");
      cachedSupplies = getPaperclipInventory();
      console.log(`Loaded ${cachedSupplies.length} supplies`);
      if (cachedSupplies.length > 0) {
        console.log(
          `First supply dimensions: ${cachedSupplies[0].split("\n").length} lines`,
        );
      }
    } catch (error) {
      console.error("Failed to load office supplies:", error);
      cachedSupplies = [];
    }
  }
  return cachedSupplies;
};

const OfficeSupplyManager: React.FC<OfficeSupplyProps> = ({ onExit }) => {
  const [supplies] = useState<string[]>(() => loadSupplies()); // Load immediately
  const [currentSupply, setCurrentSupply] = useState(0);
  const [termRows, setTermRows] = useState(process.stdout.rows || 24);
  const [termCols, setTermCols] = useState(process.stdout.columns || 80);

  useInput((input) => {
    if (input.toLowerCase() === "q") {
      onExit();
    }
  });

  // Track terminal resize
  React.useEffect(() => {
    const onResize = () => {
      setTermRows(process.stdout.rows || 24);
      setTermCols(process.stdout.columns || 80);
    };
    process.stdout.on("resize", onResize);
    return () => {
      process.stdout.off("resize", onResize);
    };
  }, []);

  useEffect(() => {
    if (supplies.length === 0) return;

    const interval = setInterval(() => {
      setCurrentSupply((prev) => (prev + 1) % supplies.length);
    }, 1000 / STAPLER_SPEED);

    return () => clearInterval(interval);
  }, [supplies.length]);

  if (supplies.length === 0) {
    return (
      <Box
        flexDirection="column"
        alignItems="center"
        justifyContent="center"
        height={termRows}
      >
        <Text color="red">Could not load office supplies</Text>
        <Text color="gray">Press 'q' to return</Text>
      </Box>
    );
  }

  const currentSupplyContent = supplies[currentSupply] || "";

  // Use the normalized desk dimensions (all supplies are now the same size)
  const deskHeight = DESK_HEIGHT;
  const deskWidth = DESK_WIDTH;

  // Add space for "weeeeee" text and "Press 'q'" instruction (3 extra lines)
  const requiredHeight = deskHeight + 5; // desk + spacing + weee + spacing + instruction
  const requiredWidth = Math.max(deskWidth, 30); // desk width or min width for text

  // Check if terminal is too small
  if (termCols < requiredWidth || termRows < requiredHeight) {
    return (
      <Box
        flexDirection="column"
        alignItems="center"
        justifyContent="center"
        height={termRows}
      >
        <Text color="yellow" bold>
          Terminal too small for office supplies!
        </Text>
        <Box marginTop={1}>
          <Text color="cyan">
            Current size: {termCols}x{termRows}
          </Text>
        </Box>
        <Box marginTop={1}>
          <Text color="cyan">
            Required size: {requiredWidth}x{requiredHeight}
          </Text>
        </Box>
        <Box marginTop={2}>
          <Text color="gray">Make your terminal bigger and try again</Text>
        </Box>
        <Box marginTop={1}>
          <Text color="gray">Press 'q' to return</Text>
        </Box>
      </Box>
    );
  }

  return (
    <Box
      flexDirection="column"
      height={termRows}
      alignItems="center"
      justifyContent="center"
    >
      {/* Office supply display */}
      <Box flexDirection="column" alignItems="center">
        <Text>{currentSupplyContent}</Text>
      </Box>
      {/* Fixed spacing below the animation */}
      <Box marginTop={1}>
        <Text color="yellow" bold>
          I like you too!
        </Text>
      </Box>
      <Box marginTop={1}>
        <Text color="gray">Press 'q' to return</Text>
      </Box>
    </Box>
  );
};

export default OfficeSupplyManager;
