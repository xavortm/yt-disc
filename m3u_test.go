package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeMP3Stub(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTracksCRUD(t *testing.T) {
	dir := t.TempDir()

	// Load from non-existent file returns nil.
	tracks, err := LoadTracks(dir)
	if err != nil {
		t.Fatalf("LoadTracks empty: %v", err)
	}
	if tracks != nil {
		t.Fatalf("expected nil, got %v", tracks)
	}

	// Append two tracks.
	if _, err := AppendTrack(dir, Track{File: "01_song.mp3", Title: "Song One", Duration: 180}); err != nil {
		t.Fatalf("AppendTrack: %v", err)
	}
	if _, err := AppendTrack(dir, Track{File: "02_song.mp3", Title: "Song Two", VideoID: "abc", Duration: 200}); err != nil {
		t.Fatalf("AppendTrack: %v", err)
	}

	tracks, err = LoadTracks(dir)
	if err != nil {
		t.Fatalf("LoadTracks: %v", err)
	}
	if len(tracks) != 2 {
		t.Fatalf("got %d tracks, want 2", len(tracks))
	}
	if tracks[0].Title != "Song One" || tracks[1].VideoID != "abc" {
		t.Errorf("unexpected tracks: %+v", tracks)
	}

	// Remove first track.
	if _, err := RemoveTrack(dir, "01_song.mp3"); err != nil {
		t.Fatalf("RemoveTrack: %v", err)
	}
	tracks, err = LoadTracks(dir)
	if err != nil {
		t.Fatalf("LoadTracks after remove: %v", err)
	}
	if len(tracks) != 1 || tracks[0].File != "02_song.mp3" {
		t.Errorf("after remove: %+v", tracks)
	}
}

func TestFormatM3U(t *testing.T) {
	tracks := []Track{
		{File: "01_song_a.mp3", Title: "Artist - Song A", Duration: 225},
		{File: "02_song_b.mp3", Title: "Artist - Song B", Duration: 198},
	}
	got := FormatM3U(tracks)
	want := "#EXTM3U\n#EXTINF:225,Artist - Song A\n01_song_a.mp3\n#EXTINF:198,Artist - Song B\n02_song_b.mp3\n"
	if got != want {
		t.Errorf("FormatM3U:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestFormatM3UEmpty(t *testing.T) {
	got := FormatM3U(nil)
	if got != "#EXTM3U\n" {
		t.Errorf("FormatM3U(nil) = %q, want header only", got)
	}
}

func TestWriteM3U(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "My Disc")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	tracks := []Track{{File: "01_x.mp3", Title: "X", Duration: 60}}
	if err := WriteM3U(dir, tracks); err != nil {
		t.Fatalf("WriteM3U: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "My Disc.m3u"))
	if err != nil {
		t.Fatalf("reading m3u: %v", err)
	}
	if !strings.HasPrefix(string(data), "#EXTM3U") {
		t.Error("m3u missing header")
	}
	if !strings.Contains(string(data), "01_x.mp3") {
		t.Error("m3u missing track entry")
	}
}

func TestValidateM3U(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "disc")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeMP3Stub(t, dir, "01_a.mp3")
	writeMP3Stub(t, dir, "02_b.mp3")

	tracks := []Track{
		{File: "01_a.mp3", Title: "A", Duration: 100},
		{File: "02_b.mp3", Title: "B", Duration: 200},
	}
	if err := SaveTracks(dir, tracks); err != nil {
		t.Fatal(err)
	}
	if err := WriteM3U(dir, tracks); err != nil {
		t.Fatal(err)
	}

	// Should be valid.
	ok, issues, err := ValidateM3U(dir)
	if err != nil {
		t.Fatalf("ValidateM3U: %v", err)
	}
	if !ok {
		t.Errorf("expected valid, got issues: %v", issues)
	}

	// Add an orphan mp3.
	writeMP3Stub(t, dir, "03_c.mp3")
	ok, issues, err = ValidateM3U(dir)
	if err != nil {
		t.Fatalf("ValidateM3U: %v", err)
	}
	if ok {
		t.Error("expected invalid after adding orphan")
	}
	found := false
	for _, iss := range issues {
		if strings.Contains(iss, "untracked") && strings.Contains(iss, "03_c.mp3") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected untracked issue for 03_c.mp3, got: %v", issues)
	}

	// Remove a tracked file.
	os.Remove(filepath.Join(dir, "01_a.mp3"))
	ok, issues, err = ValidateM3U(dir)
	if err != nil {
		t.Fatalf("ValidateM3U: %v", err)
	}
	if ok {
		t.Error("expected invalid after removing tracked file")
	}
	found = false
	for _, iss := range issues {
		if strings.Contains(iss, "missing") && strings.Contains(iss, "01_a.mp3") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing issue for 01_a.mp3, got: %v", issues)
	}
}

func TestTitleFromFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"01_beyonce_crazy_in_love.mp3", "beyonce crazy in love"},
		{"14_shturtsite_vyatar_me_nosi.mp3", "shturtsite vyatar me nosi"},
		{"no_track_prefix.mp3", "no track prefix"},
		{"untitled.mp3", "untitled"},
	}
	for _, tt := range tests {
		got := titleFromFilename(tt.input)
		if got != tt.want {
			t.Errorf("titleFromFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRecordUnrecordRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "disc")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeMP3Stub(t, dir, "01_a.mp3")
	writeMP3Stub(t, dir, "02_b.mp3")

	// Record two tracks.
	if err := RecordTrack(dir, Track{File: "01_a.mp3", Title: "Song A", Duration: 100}); err != nil {
		t.Fatalf("RecordTrack: %v", err)
	}
	if err := RecordTrack(dir, Track{File: "02_b.mp3", Title: "Song B", Duration: 200}); err != nil {
		t.Fatalf("RecordTrack: %v", err)
	}

	// Should be valid: tracks.json, m3u, and files all in sync.
	ok, issues, err := ValidateM3U(dir)
	if err != nil {
		t.Fatalf("ValidateM3U after record: %v", err)
	}
	if !ok {
		t.Errorf("expected valid after recording, got issues: %v", issues)
	}

	// Unrecord one track.
	if err := UnrecordTrack(dir, "01_a.mp3"); err != nil {
		t.Fatalf("UnrecordTrack: %v", err)
	}

	// tracks.json should have one entry, m3u should match.
	tracks, err := LoadTracks(dir)
	if err != nil {
		t.Fatalf("LoadTracks: %v", err)
	}
	if len(tracks) != 1 || tracks[0].File != "02_b.mp3" {
		t.Errorf("after unrecord: %+v", tracks)
	}

	m3uData, err := os.ReadFile(filepath.Join(dir, "disc.m3u"))
	if err != nil {
		t.Fatalf("reading m3u: %v", err)
	}
	if strings.Contains(string(m3uData), "01_a.mp3") {
		t.Error("m3u still contains unrecorded track")
	}
	if !strings.Contains(string(m3uData), "02_b.mp3") {
		t.Error("m3u missing remaining track")
	}
}
