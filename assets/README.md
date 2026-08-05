# Screenshots & GIFs

Everything in this directory is recorded with [vhs](https://github.com/charmbracelet/vhs)
against the local demo cluster, then compressed. The tapes live in
`tapes/`; this page covers the prep, the recording, and the two compression
pipelines. All recordings share `tapes/demo-config.toml` (tokyo-night,
update check off) so the assets look consistent.

## Prerequisites

- `vhs` (needs `ttyd` ≥ 1.7.4 and `ffmpeg` on PATH)
- `python3` with Pillow (PNG quantization)
- `delta` (the diff screenshot)
- The local demo cluster: `make argocd-up`

## Prep the demo data

Recordings look dead without recent activity — Kubernetes events expire after
about an hour. Before recording:

```sh
# fresh sync-operation events + a recent "synced Ns ago" summary
argocd app sync rollout-bluegreen

# fresh pod-level events on every pod (pod names are random, so the tapes
# can't target one specific pod — refresh them all)
kubectl --context k3d-argocd-demo -n argonaut-demo delete pods -l app=bluegreen-demo
```

Build the binary the tapes launch:

```sh
go build -o ./argonaut-demo ./cmd/app
```

## Record

Run from the repo root (the tapes use relative paths — vhs's parser rejects
some absolute paths, e.g. segments starting with `-`):

```sh
for t in assets/tapes/*.tape; do vhs "$t"; done
```

vhs occasionally fails with `recording failed` right after a previous run
(ttyd port contention) — just rerun the tape.

| Tape | Produces |
|------|----------|
| `apps.tape` | `apps_raw.png` — live apps list |
| `sync.tape` | `sync_raw.png` — sync confirmation |
| `diff.tape` | `diff_raw.png` — delta diff pager |
| `rollback.tape` | `rollback_raw.png` — revision picker |
| `delete_apps.tape` | `delete_apps_raw.png` — multi-app delete confirmation |
| `resource_actions.tape` | `resource_actions_raw.png` — actions modal on a Rollout |
| `help.tape` | `help_raw.png` — the `:help` modal |
| `events_pane.tape` | `argonaut_events.gif` — tree + events pane (compress below) |
| `themes.tape` | `argonaut_themes.gif` — theme picker previews (compress below) |

Conventions baked into the tapes:

- **Geometry**: `FontSize 20`, `Width 1440` — about 118 columns, chunky and
  legible at a glance. The side-by-side events pane needs ≥ 100 columns.
- `Hide`/`Show` cut startup and command-typing; recordings start on the
  interesting frame.

## Compress

### Stills (Pillow)

Terminal screenshots use few colors, so indexed-color PNGs are ~4× smaller
with no visible loss:

```sh
python3 assets/tapes/quantize.py   # *_raw.png -> assets/argonaut_<name>.png
```

Octree at 256 colors keeps the theme hues intact; median-cut at low color
counts washes them out.

### GIFs (ffmpeg)

Raw vhs output is oversized. Re-encode with a **single global palette**:

```sh
ffmpeg -i assets/argonaut_events.gif -vf \
  "fps=8,scale=1080:-1:flags=lanczos,split[a][b];[a]palettegen=max_colors=96[p];[b][p]paletteuse=dither=bayer:bayer_scale=3" \
  out.gif && mv out.gif assets/argonaut_events.gif
```

- `scale=1080` is plenty — GitHub renders the README column at ~830 px.
- `max_colors=96` for single-theme recordings; `256` when the colors change
  wholesale (the themes gif).
- **Never use per-frame palettes** (`palettegen=stats_mode=single` +
  `paletteuse=new=1`): every frame becomes a keyframe, which kills GIF delta
  compression (6 MB instead of 0.5 MB for the themes gif).
