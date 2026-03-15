# yt-disc

CLI tool that downloads songs from YouTube playlists and organizes them into CD-sized folders (80 min).

## Requirements

- Go 1.24+
- [yt-dlp](https://github.com/yt-dlp/yt-dlp)
- [ffprobe](https://ffmpeg.org/) (part of ffmpeg)

```
brew install yt-dlp ffmpeg
```

## Install

```
go install github.com/xavortm/yt-disc@latest
```

Or build from source:

```
git clone git@github.com:xavortm/yt-disc.git
cd yt-disc
go build -o yt-disc .
```

## Usage

```
yt-disc <youtube-url>          # Pick songs from playlist, download to ~/CDs/
yt-disc list                   # Browse existing disc folders
yt-disc -o ~/MyDiscs <url>     # Custom output directory
yt-disc -b 320 <url>           # Custom bitrate (default: 192k, range: 64-320)
yt-disc -m 1m <url>            # Safety margin subtracted from 80-min capacity (default: 30s)
yt-disc -n=false <url>         # Disable audio normalization
```

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output-dir` | `-o` | `~/CDs` | Output directory for disc folders |
| `--bitrate` | `-b` | `192` | MP3 bitrate in kbps (64–320) |
| `--margin` | `-m` | `30s` | Safety margin subtracted from 80-min disc capacity |
| `--normalize` | `-n` | `true` | Normalize audio levels via FFmpeg loudnorm filter |

## TUI Settings

While in the picker or disc detail view, press **N** to open the settings panel. Settings can be toggled per session without restarting:

- **Normalize audio (loudnorm)** — even out perceived volume across tracks
