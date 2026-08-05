# Screenshots & GIFs

The images in `assets/` are recorded with [vhs](https://github.com/charmbracelet/vhs)
against the local demo cluster, then compressed. The tapes live in
`assets/tapes/`; this page covers the prep, the recording, and the two
compression pipelines.

## Prerequisites

- `vhs` (needs `ttyd` ≥ 1.7.4 and `ffmpeg` on PATH)
- `python3` with Pillow (PNG quantization)
- The local demo cluster: `make argocd-up`

## Prep the demo data

Recordings look dead without recent activity — Kubernetes events expire after
about an hour. Before recording:

```sh
# fresh sync-operation events + a recent "synced Ns ago" summary
argocd app sync rollout-bluegreen

# fresh pod-level events (Scheduled/Pulled/Created/Started)
kubectl --context k3d-argocd-demo -n argonaut-demo delete pod <a bluegreen pod>
```

Build the binary the tapes launch:

```sh
go build -o ./argonaut-demo ./cmd/app
```

## Record

Run from the repo root (the tapes use relative paths — vhs's parser rejects
some absolute paths, e.g. segments starting with `-`):

```sh
vhs assets/tapes/events_pane.tape   # writes assets/argonaut_events.gif
vhs assets/tapes/themes.tape        # writes assets/argonaut_themes.gif
vhs assets/tapes/help.tape          # writes help_raw.png (+ a discardable gif)
```

vhs occasionally fails with `recording failed` right after a previous run
(ttyd port contention) — just rerun it.

Conventions baked into the tapes:

- **Geometry**: `FontSize 20`, `Width 1440` — about 118 columns, matching the
  chunky, legible-at-a-glance look of the existing assets. The side-by-side
  events pane needs ≥ 100 columns.
- **`assets/tapes/demo-config.toml`** pins the theme and disables the update
  check, so the status bar shows hints instead of "New version available".
- `Hide`/`Show` cut startup and command-typing; recordings start on the
  interesting frame.

## Compress

### GIFs (ffmpeg)

Raw vhs output is oversized. Re-encode with a **single global palette**:

```sh
ffmpeg -i assets/argonaut_events.gif -vf \
  "fps=8,scale=1080:-1:flags=lanczos,split[a][b];[a]palettegen=max_colors=96[p];[b][p]paletteuse=dither=bayer:bayer_scale=3" \
  assets/argonaut_events.gif
```

- `scale=1080` is plenty — GitHub renders the README column at ~830 px.
- `max_colors=96` for single-theme recordings; `256` when the colors change
  wholesale (the themes gif).
- **Never use per-frame palettes** (`palettegen=stats_mode=single` +
  `paletteuse=new=1`): every frame becomes a keyframe, which kills GIF delta
  compression (6 MB instead of 0.5 MB for the themes gif).

### PNGs (Pillow)

Terminal screenshots use few colors, so indexed-color PNGs are ~4× smaller
with no visible loss:

```python
from PIL import Image
im = Image.open("help_raw.png").convert("RGB")
im.quantize(colors=256, method=Image.FASTOCTREE, dither=Image.NONE) \
  .save("assets/argonaut_help.png", optimize=True)
```

Octree at 256 colors keeps the theme hues intact; median-cut at low color
counts washes them out.
