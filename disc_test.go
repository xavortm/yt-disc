package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCreateDisc(t *testing.T) {
	dir := t.TempDir()
	path, err := CreateDisc(dir, "My Disc")
	if err != nil {
		t.Fatalf("CreateDisc: %v", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("disc folder was not created")
	}
	if filepath.Base(path) != "My Disc" {
		t.Errorf("path base = %q, want %q", filepath.Base(path), "My Disc")
	}
}

func TestListDiscs(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "disc1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "disc2"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "not-a-disc.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	discs, err := ListDiscs(dir)
	if err != nil {
		t.Fatalf("ListDiscs: %v", err)
	}
	if len(discs) != 2 {
		t.Fatalf("got %d discs, want 2", len(discs))
	}
}

func TestListDiscsNonExistent(t *testing.T) {
	discs, err := ListDiscs("/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Fatalf("expected nil error for non-existent dir, got: %v", err)
	}
	if discs != nil {
		t.Fatalf("expected nil, got %v", discs)
	}
}

func TestListSongs(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "01_first_song.mp3"), "fake mp3")
	writeTestFile(t, filepath.Join(dir, "03_third_song.mp3"), "fake mp3")
	writeTestFile(t, filepath.Join(dir, "02_second_song.mp3"), "fake mp3")
	writeTestFile(t, filepath.Join(dir, "readme.txt"), "not a song")
	if err := os.MkdirAll(filepath.Join(dir, "discarded"), 0o755); err != nil {
		t.Fatal(err)
	}

	songs, err := ListSongs(dir)
	if err != nil {
		t.Fatalf("ListSongs: %v", err)
	}
	if len(songs) != 3 {
		t.Fatalf("got %d songs, want 3", len(songs))
	}
	if songs[0].TrackNum != 1 || songs[1].TrackNum != 2 || songs[2].TrackNum != 3 {
		t.Errorf("songs not sorted by track: %d, %d, %d",
			songs[0].TrackNum, songs[1].TrackNum, songs[2].TrackNum)
	}
}

func TestDiscardSong(t *testing.T) {
	dir := t.TempDir()
	songPath := filepath.Join(dir, "01_song.mp3")
	writeTestFile(t, songPath, "fake mp3")

	if err := DiscardSong(songPath); err != nil {
		t.Fatalf("DiscardSong: %v", err)
	}
	if _, err := os.Stat(songPath); !os.IsNotExist(err) {
		t.Error("song still exists at original location")
	}
	discardedPath := filepath.Join(dir, "discarded", "01_song.mp3")
	if _, err := os.Stat(discardedPath); os.IsNotExist(err) {
		t.Error("song not found in discarded folder")
	}
}

func TestNextTrackNum(t *testing.T) {
	t.Run("empty dir", func(t *testing.T) {
		dir := t.TempDir()
		n, err := NextTrackNum(dir)
		if err != nil {
			t.Fatalf("NextTrackNum: %v", err)
		}
		if n != 1 {
			t.Errorf("got %d, want 1", n)
		}
	})

	t.Run("with gaps", func(t *testing.T) {
		dir := t.TempDir()
		writeTestFile(t, filepath.Join(dir, "01_first.mp3"), "x")
		writeTestFile(t, filepath.Join(dir, "03_third.mp3"), "x")

		n, err := NextTrackNum(dir)
		if err != nil {
			t.Fatalf("NextTrackNum: %v", err)
		}
		if n != 4 {
			t.Errorf("got %d, want 4", n)
		}
	})
}

func TestParseTrackNum(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{"01_song.mp3", 1},
		{"12_another.mp3", 12},
		{"song.mp3", 0},
		{"", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTrackNum(tt.name)
			if got != tt.want {
				t.Errorf("parseTrackNum(%q) = %d, want %d", tt.name, got, tt.want)
			}
		})
	}
}

func TestTotalDuration(t *testing.T) {
	songs := []Song{
		{Duration: 3 * time.Minute},
		{Duration: 4 * time.Minute},
	}
	got := TotalDuration(songs)
	want := 7 * time.Minute
	if got != want {
		t.Errorf("TotalDuration = %v, want %v", got, want)
	}
}

func TestSongDisplayName(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"01_never_gonna_give_you_up.mp3", "Never Gonna Give You Up"},
		{"12_another_song.mp3", "Another Song"},
		{"03_a.mp3", "A"},
		{"song_without_tracknum.mp3", "Song Without Tracknum"},
		{"01_untitled.mp3", "Untitled"},
		{"99_single.mp3", "Single"},
		{"100_three_digits.mp3", "Three Digits"},
	}
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := SongDisplayName(tt.filename)
			if got != tt.want {
				t.Errorf("SongDisplayName(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestWriteSongsTxt(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "01_first_song.mp3"), "fake")
	writeTestFile(t, filepath.Join(dir, "02_second_song.mp3"), "fake")

	songs := []Song{
		{Name: "01_first_song.mp3", TrackNum: 1, Duration: 3*time.Minute + 30*time.Second},
		{Name: "02_second_song.mp3", TrackNum: 2, Duration: 4*time.Minute + 15*time.Second},
	}
	if err := writeSongsTxt(dir, "My Test Playlist", songs); err != nil {
		t.Fatalf("writeSongsTxt: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "songs.txt"))
	if err != nil {
		t.Fatalf("reading songs.txt: %v", err)
	}
	content := string(data)

	// Verify header
	if !strings.Contains(content, "My Test Playlist") {
		t.Error("songs.txt missing playlist name")
	}
	if !strings.Contains(content, "2 songs") {
		t.Error("songs.txt missing song count")
	}
	if !strings.Contains(content, "7:45") {
		t.Error("songs.txt missing total duration")
	}
	// Verify song lines
	if !strings.Contains(content, "First Song") {
		t.Error("songs.txt missing first song display name")
	}
	if !strings.Contains(content, "Second Song") {
		t.Error("songs.txt missing second song display name")
	}
	if !strings.Contains(content, "3:30") {
		t.Error("songs.txt missing first song duration")
	}
}

func TestReadSongsTxtName(t *testing.T) {
	t.Run("existing file", func(t *testing.T) {
		dir := t.TempDir()
		writeTestFile(t, filepath.Join(dir, "songs.txt"), "Best Hits 2024\n5 songs · 20:00\n")
		got := ReadSongsTxtName(dir)
		if got != "Best Hits 2024" {
			t.Errorf("ReadSongsTxtName = %q, want %q", got, "Best Hits 2024")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		dir := t.TempDir()
		got := ReadSongsTxtName(dir)
		if got != "" {
			t.Errorf("ReadSongsTxtName = %q, want empty", got)
		}
	})
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
