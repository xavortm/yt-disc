package main

import (
	"os"
	"path/filepath"
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

func TestTotalSize(t *testing.T) {
	songs := []Song{
		{Size: 1024},
		{Size: 2048},
	}
	got := TotalSize(songs)
	if got != 3072 {
		t.Errorf("TotalSize = %d, want 3072", got)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
