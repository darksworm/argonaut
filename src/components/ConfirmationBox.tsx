import { Box, Text } from "ink";
import TextInput from "ink-text-input";
import React, { useState } from "react";

export interface ConfirmationOption {
  key: string;
  label: string;
  value: boolean;
  disabled?: boolean;
}

interface ConfirmationBoxProps {
  title: string;
  message: string;
  target?: string;
  isMulti?: boolean;
  options?: ConfirmationOption[];
  onConfirm: (confirmed: boolean) => void;
}

// TODO: handle the input completely in this component
export default function ConfirmationBox(props: ConfirmationBoxProps) {
  const {
    title,
    message,
    target,
    isMulti = false,
    options = [],
    onConfirm,
  } = props;
  const [confirmInput, setConfirmInput] = useState("");

  const handleInputChange = (val: string) => {
    const filtered = (val || "").replace(/[^a-zA-Z]/g, "").toLowerCase();
    // Allow only prefixes of y/yes or n/no
    if (/^(y(es?)?|n(o?)?)?$/.test(filtered)) {
      // Limit to max length of the longest allowed word
      setConfirmInput(filtered.slice(0, 3));
    }
    // else ignore invalid characters (do not update state)
  };

  const handleInputSubmit = (val: string) => {
    const t = (val || "").trim().toLowerCase();
    // reset input each submit
    setConfirmInput("");
    if (t === "y" || t === "yes") {
      onConfirm(true);
      return;
    }
    if (t === "n" || t === "no") {
      onConfirm(false);
      return;
    }
    if (t === "") {
      // Do nothing on empty submit
      return;
    }
    // Ignore any other input, stay in confirm mode
  };

  return (
    <Box
      borderStyle="round"
      borderColor="yellow"
      paddingX={2}
      paddingY={1}
      flexDirection="column"
      width="100%"
    >
      <Text bold>{title}</Text>
      <Box marginTop={1}>
        <Text>
          {message}{" "}
          {target && (
            <Text color="magentaBright" bold>
              {isMulti ? `(${target})` : target}
            </Text>
          )}
          ? (y/n):{" "}
        </Text>
        <TextInput
          value={confirmInput}
          onChange={handleInputChange}
          onSubmit={handleInputSubmit}
        />
      </Box>
      {options.length > 0 && (
        <Box marginTop={1}>
          <Text>
            {options.map((opt, i) => (
              <React.Fragment key={opt.key}>
                {i > 0 && " • "}
                {opt.disabled ? (
                  <Text dimColor>{opt.label}: disabled</Text>
                ) : (
                  <>
                    {opt.label} [{opt.key}]:{" "}
                    <Text color={(opt.value ? "yellow" : undefined) as any}>
                      {opt.value ? "on" : "off"}
                    </Text>
                  </>
                )}
              </React.Fragment>
            ))}
          </Text>
        </Box>
      )}
    </Box>
  );
}
