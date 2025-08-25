import { Box, useInput } from "ink";
import type React from "react";
import { useAppState } from "../../contexts/AppStateContext";
import Help from "../Help";

export const HelpModal: React.FC = () => {
  const { state, dispatch } = useAppState();

  // Handle input for help modal
  useInput((input, key) => {
    if (state.mode !== "help") return;

    if (input === "q" || input === "?" || key.escape) {
      dispatch({ type: "SET_MODE", payload: "normal" });
    }
  });

  if (state.mode !== "help") {
    return null;
  }

  return (
    <Box flexDirection="column" marginTop={1} flexGrow={1}>
      <Help />
    </Box>
  );
};
