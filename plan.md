# yt-disc — Implementation Plan

## What we're building

A personal Go CLI that downloads songs from YouTube playlists and organizes them into CD-sized folders. Interactive TUI for picking songs. No burn integration — just folder prep.

**Core decisions:**
- Name: **yt-disc**
- Default: Audio CD (80 min)
- YouTube: **goutubedl** (wraps yt-dlp)
- Removed songs: moved to `discarded/` subfolder
- Disc folders on disk = source of truth

---

## Project Structure

```
yt-disc/
├── main.go        # CLI entry: parse args, start TUI
├── youtube.go     # goutubedl: fetch playlist/video metadata, download audio
├── urlparse.go    # URL type detection (single vs playlist vs ambiguous)
├── disc.go        # Disc folder ops: create, list, songs, discard, probe durations
├── naming.go      # CD-safe filename sanitization
├── tui.go         # Bubble Tea: all views (playlist picker, disc list, disc view, download)
├── go.mod
└── project.md
```

That's it. No `internal/`, no `service/`, no `interfaces.go`, no `engine.go`, no config package. This is a personal tool — we can refactor into packages **when** complexity demands it, not before.

---

## Dependencies

| Package | Why |
|---|---|
| `github.com/wader/goutubedl` | Fetch YouTube metadata + download audio |
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/lipgloss` | Terminal styling |
| `github.com/charmbracelet/bubbles` | List, spinner, progress components |
| Standard library | `os/exec` (ffprobe), `path/filepath`, `net/url` |

No YAML config library. Use CLI flags. If config is needed later, add it later.

---

## CLI Interface

```bash
yt-disc <playlist-or-video-url>     # Flow 1: pick songs, create disc
yt-disc list                         # Flow 2: browse existing disc folders
yt-disc --mode data                  # Data CD mode (700 MB instead of 80 min)
yt-disc -o ~/MyDiscs <url>           # Custom output dir (default: ~/CDs)
```

Flags: `--output-dir/-o` (default `~/CDs`), `--mode/-m` (audio|data), `--bitrate/-b` (default 192).
Parse with `flag` stdlib. No cobra, no viper, no config files.

---

## Flow 1: New Disc from Playlist

1. `yt-disc <playlist-url>`
2. Fetch metadata via goutubedl → get song titles + durations
3. Show interactive picker (Bubble Tea list with checkboxes)
4. Header: selected count, total duration vs 80:00, progress bar
5. Keys: `j/k` nav, `space` toggle, `a` all, `n` none, `s` save, `q` quit
6. On save: create folder `<output>/<playlist-name>/`, download selected songs
7. Show download progress (spinner per song)
8. Files renamed with CD-safe names (`01_artist_song.mp3`)

## Flow 2: List & Manage Discs

1. `yt-disc list`
2. Scan output dir, probe each disc's audio files with ffprobe
3. Show list: disc name, song count, duration used
4. `enter` to open a disc → show songs, duration, capacity
5. `x` to discard a song (moves to `discarded/` subfolder)
6. `b` back, `q` quit

## Flow 3: Add to Existing Disc

1. While in disc view, paste a URL
2. Single video → fetch, download, add with next track number
3. Playlist → open picker, selected songs go into existing disc

---

## File Naming

```
Input:  "Beyoncé — Crazy in Love [Official Video]"
→ transliterate diacritics → remove [brackets] → lowercase
→ replace non-alnum with _ → collapse underscores → truncate to 60 chars
→ prepend track num → append .mp3
Output: "01_beyonce_crazy_in_love.mp3"
```

Keep `(feat. X)`, `(Remix)`, `(Live)`. Remove `[Official Video]`, `[HD]`, `(Lyrics)`, etc.
Track gaps after discard are fine — don't renumber.

---

## CD Capacity

- **Audio mode** (default): 80 minutes. Track by summing durations.
- **Data mode**: 700 MB. Track by summing file sizes.
- For playlist metadata: use duration from goutubedl.
- For existing files: use ffprobe.
- Allow overflow — show red warning, don't block.

---

## Startup Checks

On launch, verify `yt-dlp` and `ffprobe` are in PATH. If not, print install instructions and exit. That's it.

---

## Testing

Write tests for the parts that have actual logic:

- **`naming_test.go`** — Table-driven tests for filename sanitization. This is the fiddliest pure logic.
- **`urlparse_test.go`** — Table-driven tests for URL type detection (single/playlist/ambiguous/invalid).
- **`disc_test.go`** — Folder operations with `t.TempDir()`: create, list, discard, restore, next track number.

Don't test: TUI rendering, goutubedl wrapper calls, trivial glue code.
Don't write: fake interfaces, mock engines, 4-layer test pyramids, coverage targets.
Write tests for things that can actually break in non-obvious ways.

---

## Implementation Order

### Phase 1: Core Logic
- `go.mod` — rename module, add dependencies
- `naming.go` + tests — filename sanitization
- `urlparse.go` + tests — URL detection
- `disc.go` + tests — folder CRUD, ffprobe duration reading
- `youtube.go` — goutubedl wrapper (fetch metadata, download)

### Phase 2: Flow 1 (main flow)
- `tui.go` — playlist picker view, download progress view
- `main.go` — CLI arg parsing, wire everything, start TUI

### Phase 3: Flow 2 + 3
- Add disc list view and disc content view to `tui.go`
- Add URL paste handling in disc view
- `list` subcommand in `main.go`

### Phase 4: Polish (only if needed)
- Overflow warnings styling
- Graceful Ctrl+C during downloads
- Edge cases as they come up

---

## What we're NOT doing (yet)

- No config file / YAML — flags are enough for 3 settings
- No `internal/service/` abstraction layer — direct function calls
- No interface-driven architecture — premature for a single-user CLI
- No fake/mock test infrastructure — test real logic with real (temp) filesystems
- No coverage targets — write tests that catch bugs, not tests that hit numbers
- No restore command for discarded songs — just a folder, user can mv manually if desperate
- No duplicate detection by video ID — filename collision handling is enough

When this tool grows to the point where we _feel the pain_ of a flat structure, we refactor. Not before.
