# yt-disc

CLI tool that downloads songs from YouTube playlists and organizes them into CD-sized folders (80 min audio / 700 MB data).

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
yt-disc --mode data <url>      # Data CD mode (700 MB instead of 80 min)
yt-disc -b 320 <url>           # Custom bitrate (default: 192k)
```

## Keys

**Picker:** `j/k` navigate, `space` toggle, `a` all, `n` none, `s` save, `q` quit

**Disc browser:** `j/k` navigate, `enter` open, `x` discard song, `u` add URL, `b` back, `q` quit
