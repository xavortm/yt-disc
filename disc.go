package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const AudioCDCapacity = 80 * time.Minute

// Song represents an audio file on a disc.
type Song struct {
	Path     string
	Name     string
	TrackNum int
	Duration time.Duration
}

// Disc represents a folder of songs.
type Disc struct {
	Name  string
	Path  string
	Songs []Song
}

// CreateDisc creates a new disc folder under baseDir and returns its path.
func CreateDisc(baseDir, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("disc name cannot be empty")
	}
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return "", fmt.Errorf("invalid disc name: %q", name)
	}
	p := filepath.Join(baseDir, name)
	if err := os.MkdirAll(p, 0o755); err != nil {
		return "", fmt.Errorf("creating disc folder: %w", err)
	}
	return p, nil
}

// ListDiscs returns all disc folders under baseDir.
func ListDiscs(baseDir string) ([]Disc, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading disc dir: %w", err)
	}

	var discs []Disc
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		d := Disc{Name: e.Name(), Path: filepath.Join(baseDir, e.Name())}
		d.Songs, _ = ListSongs(d.Path) // best-effort; disc is still listed if songs fail
		discs = append(discs, d)
	}
	return discs, nil
}

// ListSongs returns mp3 files in a disc folder sorted by track number.
func ListSongs(discPath string) ([]Song, error) {
	entries, err := os.ReadDir(discPath)
	if err != nil {
		return nil, fmt.Errorf("reading songs: %w", err)
	}

	var songs []Song
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") || !strings.HasSuffix(strings.ToLower(e.Name()), ".mp3") {
			continue
		}
		s := Song{
			Path:     filepath.Join(discPath, e.Name()),
			Name:     e.Name(),
			TrackNum: parseTrackNum(e.Name()),
		}
		songs = append(songs, s)
	}
	sort.Slice(songs, func(i, j int) bool {
		return songs[i].TrackNum < songs[j].TrackNum
	})
	return songs, nil
}

// DiscardSong moves a song into the discarded/ subfolder within the disc.
func DiscardSong(songPath string) error {
	discDir := filepath.Dir(songPath)
	discardDir := filepath.Join(discDir, "discarded")
	if err := os.MkdirAll(discardDir, 0o755); err != nil {
		return fmt.Errorf("creating discarded dir: %w", err)
	}
	return os.Rename(songPath, filepath.Join(discardDir, filepath.Base(songPath)))
}

// NextTrackNum returns the next available track number for a disc folder.
func NextTrackNum(discPath string) (int, error) {
	songs, err := ListSongs(discPath)
	if err != nil {
		return 1, err
	}
	if len(songs) == 0 {
		return 1, nil
	}
	return songs[len(songs)-1].TrackNum + 1, nil
}

// ProbeDuration uses ffprobe to get the duration of an audio file.
func ProbeDuration(path string) (time.Duration, error) {
	out, err := exec.Command(
		"ffprobe", "-v", "quiet",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	).Output()
	if err != nil {
		return 0, fmt.Errorf("probing %s: %w", filepath.Base(path), err)
	}
	secs, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0, fmt.Errorf("parsing duration: %w", err)
	}
	return time.Duration(secs * float64(time.Second)), nil
}

// TotalDuration sums song durations.
func TotalDuration(songs []Song) time.Duration {
	var total time.Duration
	for _, s := range songs {
		total += s.Duration
	}
	return total
}

func parseTrackNum(name string) int {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(parts[0]) // non-numeric prefix → 0, which is fine
	return n
}
