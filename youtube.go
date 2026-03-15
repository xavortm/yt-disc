package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/wader/goutubedl"
)

// VideoMeta holds metadata for a single YouTube video.
type VideoMeta struct {
	ID       string
	Title    string
	Duration time.Duration
	URL      string
}

// FetchPlaylistMeta returns the playlist title and metadata for each video.
func FetchPlaylistMeta(ctx context.Context, playlistURL string) (string, []VideoMeta, error) {
	result, err := goutubedl.New(ctx, playlistURL, goutubedl.Options{
		Type: goutubedl.TypePlaylist,
	})
	if err != nil {
		return "", nil, fmt.Errorf("fetching playlist: %w", err)
	}

	var videos []VideoMeta
	for _, e := range result.Info.Entries {
		videos = append(videos, VideoMeta{
			ID:       e.ID,
			Title:    e.Title,
			Duration: time.Duration(e.Duration * float64(time.Second)),
			URL:      fmt.Sprintf("https://www.youtube.com/watch?v=%s", e.ID),
		})
	}
	return result.Info.Title, videos, nil
}

// FetchVideoMeta returns metadata for a single video.
func FetchVideoMeta(ctx context.Context, videoURL string) (VideoMeta, error) {
	result, err := goutubedl.New(ctx, videoURL, goutubedl.Options{
		Type: goutubedl.TypeSingle,
	})
	if err != nil {
		return VideoMeta{}, fmt.Errorf("fetching video: %w", err)
	}
	return VideoMeta{
		ID:       result.Info.ID,
		Title:    result.Info.Title,
		Duration: time.Duration(result.Info.Duration * float64(time.Second)),
		URL:      videoURL,
	}, nil
}

// DownloadAudio downloads a video as mp3 via yt-dlp.
// destPath should end in .mp3; yt-dlp handles the conversion.
// When normalize is true, the FFmpeg loudnorm filter is applied to even out
// perceived volume across tracks.
func DownloadAudio(ctx context.Context, videoURL, destPath string, bitrate int, normalize bool) error {
	base := strings.TrimSuffix(destPath, ".mp3")
	args := []string{
		"--extract-audio",
		"--audio-format", "mp3",
		"--audio-quality", fmt.Sprintf("%dk", bitrate),
		"--output", base + ".%(ext)s",
		"--no-playlist",
		"--quiet",
		"--no-warnings",
	}
	if normalize {
		args = append(args, "--postprocessor-args", "ffmpeg:-af loudnorm")
	}
	args = append(args, videoURL)
	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		if len(msg) > 512 {
			msg = string([]rune(msg)[:512]) + "…"
		}
		return fmt.Errorf("yt-dlp: %w\n%s", err, msg)
	}
	return nil
}
