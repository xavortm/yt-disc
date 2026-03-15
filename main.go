package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type appConfig struct {
	outputDir string
	bitrate   int
	margin    time.Duration
	normalize bool
}

// capacity returns the effective disc capacity after subtracting the safety margin.
func (c appConfig) capacity() time.Duration {
	effective := AudioCDCapacity - c.margin
	if effective < 0 {
		return 0
	}
	return effective
}

func main() {
	var cfg appConfig
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolving home directory: %v\n", err)
		os.Exit(1)
	}
	defOut := filepath.Join(home, "CDs")

	var marginStr string
	flag.StringVar(&cfg.outputDir, "output-dir", defOut, "output directory for disc folders")
	flag.StringVar(&cfg.outputDir, "o", defOut, "output directory (shorthand)")
	flag.IntVar(&cfg.bitrate, "bitrate", 192, "mp3 bitrate in kbps (64-320)")
	flag.IntVar(&cfg.bitrate, "b", 192, "mp3 bitrate (shorthand)")
	flag.StringVar(&marginStr, "margin", "30s", "safety margin subtracted from 80-min disc capacity (e.g. 30s, 1m, 2m30s)")
	flag.StringVar(&marginStr, "m", "30s", "safety margin (shorthand)")
	flag.BoolVar(&cfg.normalize, "normalize", true, "normalize audio levels via FFmpeg loudnorm filter")
	flag.BoolVar(&cfg.normalize, "n", true, "normalize audio (shorthand)")
	flag.Parse()

	margin, err := time.ParseDuration(marginStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid margin %q: %v\n", marginStr, err)
		os.Exit(1)
	}
	if margin < 0 {
		fmt.Fprintln(os.Stderr, "margin must not be negative")
		os.Exit(1)
	}
	if margin >= AudioCDCapacity {
		fmt.Fprintf(os.Stderr, "margin must be less than %v\n", AudioCDCapacity)
		os.Exit(1)
	}
	cfg.margin = margin

	if cfg.bitrate < 64 || cfg.bitrate > 320 {
		fmt.Fprintln(os.Stderr, "bitrate must be between 64 and 320 kbps")
		os.Exit(1)
	}

	if err := os.MkdirAll(cfg.outputDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "output directory %q: %v\n", cfg.outputDir, err)
		os.Exit(1)
	}

	if err := checkDeps(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: yt-disc <playlist-or-video-url>")
		fmt.Fprintln(os.Stderr, "       yt-disc list")
		fmt.Fprintln(os.Stderr, "       yt-disc -o ~/MyDiscs <url>")
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		runList(cfg)
	default:
		runFetch(args[0], cfg)
	}
}

func runFetch(rawURL string, cfg appConfig) {
	parsed := ParseYouTubeURL(rawURL)

	switch parsed.Type {
	case URLPlaylist, URLAmbiguous, URLSingle:
		runTUI(newFetchModel(rawURL, parsed, cfg))
	default:
		fmt.Fprintln(os.Stderr, "Invalid YouTube URL.")
		os.Exit(1)
	}
}

func runList(cfg appConfig) {
	runTUI(newListModel(cfg))
}

func runTUI(m model) {
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func checkDeps() error {
	for _, dep := range []string{"yt-dlp", "ffprobe"} {
		if _, err := exec.LookPath(dep); err != nil {
			return fmt.Errorf("%s not found in PATH; install with: brew install %s", dep, dep)
		}
	}
	return nil
}
