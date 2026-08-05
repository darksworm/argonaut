#!/usr/bin/env python3
"""Quantize *_raw.png captures into indexed-color assets/argonaut_<name>.png.

Terminal screenshots use few colors, so a 256-color octree palette shrinks
them ~4x with no visible loss (median-cut washes out theme hues). Run from
the repo root after recording the still-frame tapes.
"""
from pathlib import Path

from PIL import Image

for raw in sorted(Path(".").glob("*_raw.png")):
    out = Path("assets") / f"argonaut_{raw.stem.removesuffix('_raw')}.png"
    im = Image.open(raw).convert("RGB")
    im.quantize(colors=256, method=Image.FASTOCTREE, dither=Image.NONE).save(
        out, optimize=True
    )
    print(f"{raw} -> {out} ({out.stat().st_size // 1024} KB)")
