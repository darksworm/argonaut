#!/usr/bin/env python3
"""
Script to remap ANSI color codes to match Ghostty cyberdream theme colors.
"""

import sys
import re

# Ghostty cyberdream theme color mapping
# Based on /Users/ilmars/.config/ghostty/themes/cyberdream
CYBERDREAM_COLORS = {
    # Standard colors (0-7)
    '\x1b[30m': '\x1b[38;2;22;24;26m',      # black -> #16181a
    '\x1b[31m': '\x1b[38;2;255;110;94m',    # red -> #ff6e5e
    '\x1b[32m': '\x1b[38;2;94;255;108m',    # green -> #5eff6c
    '\x1b[33m': '\x1b[38;2;241;255;94m',    # yellow -> #f1ff5e
    '\x1b[34m': '\x1b[38;2;94;161;255m',    # blue -> #5ea1ff
    '\x1b[35m': '\x1b[38;2;189;94;255m',    # magenta -> #bd5eff
    '\x1b[36m': '\x1b[38;2;94;241;255m',    # cyan -> #5ef1ff
    '\x1b[37m': '\x1b[38;2;255;255;255m',   # white -> #ffffff

    # Bright colors (8-15) - using same colors as standard in cyberdream
    '\x1b[90m': '\x1b[38;2;60;64;72m',      # bright black -> #3c4048
    '\x1b[91m': '\x1b[38;2;255;110;94m',    # bright red -> #ff6e5e
    '\x1b[92m': '\x1b[38;2;94;255;108m',    # bright green -> #5eff6c
    '\x1b[93m': '\x1b[38;2;241;255;94m',    # bright yellow -> #f1ff5e
    '\x1b[94m': '\x1b[38;2;94;161;255m',    # bright blue -> #5ea1ff
    '\x1b[95m': '\x1b[38;2;189;94;255m',    # bright magenta -> #bd5eff
    '\x1b[96m': '\x1b[38;2;94;241;255m',    # bright cyan -> #5ef1ff
    '\x1b[97m': '\x1b[38;2;255;255;255m',   # bright white -> #ffffff
}

def remap_colors(content):
    """Remap ANSI color codes to RGB values matching cyberdream theme."""
    result = content
    for ansi_code, rgb_code in CYBERDREAM_COLORS.items():
        result = result.replace(ansi_code, rgb_code)
    return result

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python3 ghostty_color_remap.py input_file output_file")
        sys.exit(1)

    input_file = sys.argv[1]
    output_file = sys.argv[2]

    try:
        with open(input_file, 'rb') as f:
            content = f.read()

        # Convert to string for processing
        content_str = content.decode('utf-8', errors='replace')

        # Remap to RGB colors
        remapped = remap_colors(content_str)

        # Write back as bytes
        with open(output_file, 'wb') as f:
            f.write(remapped.encode('utf-8'))

        print(f"Cyberdream color remapping complete: {input_file} -> {output_file}")

    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)