package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

type appConfig struct {
	outputDir string
	mode      string
	bitrate   int
}

func main() {
	var cfg appConfig
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolving home directory: %v\n", err)
		os.Exit(1)
	}
	defOut := filepath.Join(home, "CDs")

	flag.StringVar(&cfg.outputDir, "output-dir", defOut, "output directory for disc folders")
	flag.StringVar(&cfg.outputDir, "o", defOut, "output directory (shorthand)")
	flag.StringVar(&cfg.mode, "mode", "audio", "capacity mode: audio (80 min) or data (700 MB)")
	flag.StringVar(&cfg.mode, "m", "audio", "capacity mode (shorthand)")
	flag.IntVar(&cfg.bitrate, "bitrate", 192, "mp3 bitrate in kbps")
	flag.IntVar(&cfg.bitrate, "b", 192, "mp3 bitrate (shorthand)")
	flag.Parse()

	if err := checkDeps(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: yt-disc <playlist-or-video-url>")
		fmt.Fprintln(os.Stderr, "       yt-disc list")
		fmt.Fprintln(os.Stderr, "       yt-disc --mode data <url>")
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
