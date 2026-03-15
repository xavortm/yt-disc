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
	"unicode"
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

const songsTxtFile = "songs.txt"

// WriteSongsTxt writes a songs.txt manifest into the disc folder.
// The file contains a header with the disc name, song count, and total
// duration, followed by one numbered line per song in track order.
func WriteSongsTxt(discPath, displayName string) error {
	songs, err := ListSongs(discPath)
	if err != nil {
		return fmt.Errorf("listing songs for manifest: %w", err)
	}

	for i := range songs {
		if dur, err := ProbeDuration(songs[i].Path); err == nil {
			songs[i].Duration = dur
		}
	}

	return writeSongsTxt(discPath, displayName, songs)
}

// writeSongsTxt writes songs.txt from pre-populated song data.
func writeSongsTxt(discPath, displayName string, songs []Song) error {
	total := TotalDuration(songs)

	var b strings.Builder
	b.WriteString(displayName + "\n")
	b.WriteString(fmt.Sprintf("%d songs · %s\n", len(songs), fmtDuration(total)))
	b.WriteString("────────────────────────────────────────\n")

	for i, s := range songs {
		name := SongDisplayName(s.Name)
		dur := fmtDuration(s.Duration)
		b.WriteString(fmt.Sprintf("%2d. %-40s  %s\n", i+1, name, dur))
	}

	return os.WriteFile(filepath.Join(discPath, songsTxtFile), []byte(b.String()), 0o644)
}

// ReadSongsTxtName reads the display name (first line) from an existing
// songs.txt. Returns an empty string if the file does not exist or is empty.
func ReadSongsTxtName(discPath string) string {
	data, err := os.ReadFile(filepath.Join(discPath, songsTxtFile))
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(string(data), "\n")
	return strings.TrimSpace(line)
}

// SongDisplayName converts a sanitized track filename back to a readable name.
// "02_never_gonna_give_you_up.mp3" → "Never Gonna Give You Up"
func SongDisplayName(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	// Strip leading track number prefix (e.g., "01_").
	if idx := strings.Index(name, "_"); idx > 0 && idx <= 3 {
		if _, err := strconv.Atoi(name[:idx]); err == nil {
			name = name[idx+1:]
		}
	}
	name = strings.ReplaceAll(name, "_", " ")
	return titleCase(name)
}

func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		r := []rune(w)
		r[0] = unicode.ToUpper(r[0])
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}

func parseTrackNum(name string) int {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(parts[0]) // non-numeric prefix → 0, which is fine
	return n
}
