package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// FormatM3U builds an Extended M3U playlist string from tracks.
func FormatM3U(tracks []Track) string {
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	for _, t := range tracks {
		b.WriteString(fmt.Sprintf("#EXTINF:%d,%s\n%s\n", t.Duration, t.Title, t.File))
	}
	return b.String()
}

// WriteM3U writes an Extended M3U playlist into the disc folder.
// The filename is derived from the folder name.
func WriteM3U(discPath string, tracks []Track) error {
	name := filepath.Base(discPath) + ".m3u"
	return os.WriteFile(filepath.Join(discPath, name), []byte(FormatM3U(tracks)), 0o644)
}

// RecordTrack appends a track to tracks.json and rewrites the M3U in one operation.
func RecordTrack(discPath string, t Track) error {
	tracks, err := AppendTrack(discPath, t)
	if err != nil {
		return fmt.Errorf("recording track: %w", err)
	}
	if err := WriteM3U(discPath, tracks); err != nil {
		return fmt.Errorf("writing m3u: %w", err)
	}
	return nil
}

// UnrecordTrack removes a track by filename from tracks.json and rewrites the M3U.
func UnrecordTrack(discPath string, filename string) error {
	tracks, err := RemoveTrack(discPath, filename)
	if err != nil {
		return fmt.Errorf("removing track: %w", err)
	}
	if err := WriteM3U(discPath, tracks); err != nil {
		return fmt.Errorf("writing m3u: %w", err)
	}
	return nil
}

// RegenerateM3U rebuilds tracks.json and the .m3u file from the current
// disc folder state. Tracks already in tracks.json keep their metadata;
// orphan .mp3 files get a best-effort entry using the filename as title.
func RegenerateM3U(discPath string) error {
	songs, err := ListSongs(discPath)
	if err != nil {
		return err
	}

	existing, err := LoadTracks(discPath)
	if err != nil {
		return err
	}
	byFile := make(map[string]Track, len(existing))
	for _, t := range existing {
		byFile[t.File] = t
	}

	tracks := make([]Track, 0, len(songs))
	for _, s := range songs {
		if t, ok := byFile[s.Name]; ok {
			// Re-probe duration if it was 0 or file changed.
			dur, probeErr := ProbeDuration(s.Path)
			if probeErr == nil {
				t.Duration = int(math.Round(dur.Seconds()))
			}
			tracks = append(tracks, t)
		} else {
			// Orphan mp3 — create best-effort entry from filename.
			dur, _ := ProbeDuration(s.Path)
			tracks = append(tracks, Track{
				File:     s.Name,
				Title:    titleFromFilename(s.Name),
				Duration: int(math.Round(dur.Seconds())),
			})
		}
	}

	if err := SaveTracks(discPath, tracks); err != nil {
		return err
	}
	return WriteM3U(discPath, tracks)
}

// ValidateM3U compares tracks.json against actual .mp3 files on disk.
// Returns true if everything matches, plus a list of issues found.
func ValidateM3U(discPath string) (bool, []string, error) {
	songs, err := ListSongs(discPath)
	if err != nil {
		return false, nil, err
	}
	tracks, err := LoadTracks(discPath)
	if err != nil {
		return false, nil, err
	}

	onDisk := make(map[string]bool, len(songs))
	for _, s := range songs {
		onDisk[s.Name] = true
	}
	inJSON := make(map[string]bool, len(tracks))
	for _, t := range tracks {
		inJSON[t.File] = true
	}

	var issues []string
	for _, t := range tracks {
		if !onDisk[t.File] {
			issues = append(issues, fmt.Sprintf("missing file: %s", t.File))
		}
	}
	for _, s := range songs {
		if !inJSON[s.Name] {
			issues = append(issues, fmt.Sprintf("untracked file: %s", s.Name))
		}
	}

	// Check .m3u file exists.
	m3uName := filepath.Base(discPath) + ".m3u"
	if _, err := os.Stat(filepath.Join(discPath, m3uName)); os.IsNotExist(err) {
		issues = append(issues, "m3u file missing")
	}

	return len(issues) == 0, issues, nil
}

// titleFromFilename converts "01_beyonce_crazy_in_love.mp3" to
// "beyonce crazy in love" as a fallback display name.
func titleFromFilename(name string) string {
	name = strings.TrimSuffix(name, filepath.Ext(name))
	// Strip leading track number prefix (e.g. "01_").
	parts := strings.SplitN(name, "_", 2)
	if len(parts) == 2 {
		if _, err := fmt.Sscanf(parts[0], "%d", new(int)); err == nil {
			name = parts[1]
		}
	}
	return strings.ReplaceAll(name, "_", " ")
}
