import { Box, Text } from "ink";
import TextInput from "ink-text-input";
import type React from "react";
import { useAppState } from "../../contexts/AppStateContext";

interface SearchBarProps {
  onSubmit: () => void;
}

export const SearchBar: React.FC<SearchBarProps> = ({ onSubmit }) => {
  const { state, dispatch } = useAppState();
  const { navigation, ui } = state;

  if (state.mode !== "search") {
    return null;
  }

  const handleSubmit = () => {
    dispatch({ type: "SET_SELECTED_IDX", payload: 0 });
    dispatch({ type: "SET_MODE", payload: "normal" });

    if (navigation.view === "apps") {
      dispatch({ type: "SET_ACTIVE_FILTER", payload: ui.searchQuery });
    } else {
      onSubmit(); // For drill down in other views
    }
  };

  return (
    <Box borderStyle="round" borderColor="yellow" paddingX={1}>
      <Text bold color="cyan">
        Search
      </Text>
      <Box width={1} />
      <TextInput
        value={ui.searchQuery}
        onChange={(value) =>
          dispatch({ type: "SET_SEARCH_QUERY", payload: value })
        }
        onSubmit={handleSubmit}
      />
      <Box width={2} />
      <Text dimColor>
        (Enter{" "}
        {navigation.view === "apps" ? "keeps filter" : "opens first result"},
        Esc cancels)
      </Text>
    </Box>
  );
};
