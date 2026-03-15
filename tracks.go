package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const tracksFile = "tracks.json"

// Track holds metadata for a downloaded song, preserving the original title
// that is lost during filename sanitization.
type Track struct {
	File     string `json:"file"`
	Title    string `json:"title"`
	VideoID  string `json:"videoID,omitempty"`
	Duration int    `json:"duration"` // seconds
}

// LoadTracks reads tracks.json from a disc folder.
// Returns nil slice (no error) if the file does not exist.
func LoadTracks(discPath string) ([]Track, error) {
	data, err := os.ReadFile(filepath.Join(discPath, tracksFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading tracks: %w", err)
	}
	var tracks []Track
	if err := json.Unmarshal(data, &tracks); err != nil {
		return nil, fmt.Errorf("parsing tracks: %w", err)
	}
	return tracks, nil
}

// SaveTracks writes tracks.json to a disc folder.
func SaveTracks(discPath string, tracks []Track) error {
	data, err := json.MarshalIndent(tracks, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding tracks: %w", err)
	}
	return os.WriteFile(filepath.Join(discPath, tracksFile), data, 0o644)
}

// AppendTrack adds a track to tracks.json and returns the resulting list.
func AppendTrack(discPath string, t Track) ([]Track, error) {
	tracks, err := LoadTracks(discPath)
	if err != nil {
		return nil, err
	}
	tracks = append(tracks, t)
	return tracks, SaveTracks(discPath, tracks)
}

// RemoveTrack removes a track by filename from tracks.json and returns the resulting list.
func RemoveTrack(discPath string, filename string) ([]Track, error) {
	tracks, err := LoadTracks(discPath)
	if err != nil {
		return nil, err
	}
	filtered := tracks[:0]
	for _, t := range tracks {
		if t.File != filename {
			filtered = append(filtered, t)
		}
	}
	return filtered, SaveTracks(discPath, filtered)
}
