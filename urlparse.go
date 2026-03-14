package main

import (
	"net/url"
	"strings"
	"unicode"
)

// URLType indicates the kind of YouTube URL.
type URLType int

const (
	URLInvalid   URLType = iota
	URLSingle            // single video
	URLPlaylist          // playlist page
	URLAmbiguous         // video within a playlist (has both v= and list=)
)

// String returns a human-readable label for the URL type.
func (t URLType) String() string {
	switch t {
	case URLSingle:
		return "single"
	case URLPlaylist:
		return "playlist"
	case URLAmbiguous:
		return "ambiguous"
	default:
		return "invalid"
	}
}

// ParsedURL holds the parsed components of a YouTube URL.
type ParsedURL struct {
	Type       URLType
	VideoID    string
	PlaylistID string
}

// maxIDLen is the longest video or playlist ID we accept.
// Real YouTube IDs are 11 (video) or ~34 (playlist) characters.
const maxIDLen = 128

// ParseYouTubeURL detects whether a URL points to a single video, playlist, or both.
func ParseYouTubeURL(raw string) ParsedURL {
	p := ParsedURL{Type: URLInvalid}

	// Reject URLs with control characters (null bytes, newlines, tabs, etc.).
	if strings.ContainsFunc(raw, unicode.IsControl) {
		return p
	}

	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return p
	}

	// Only allow HTTP(S) schemes.
	if u.Scheme != "http" && u.Scheme != "https" {
		return p
	}

	host := strings.ToLower(u.Hostname())
	q := u.Query()

	switch {
	case host == "youtu.be":
		p.VideoID = strings.TrimPrefix(u.Path, "/")
		p.PlaylistID = q.Get("list")

	case host == "youtube.com" || host == "www.youtube.com" ||
		host == "m.youtube.com" || host == "music.youtube.com":
		switch {
		case strings.HasPrefix(u.Path, "/watch"):
			p.VideoID = q.Get("v")
			p.PlaylistID = q.Get("list")
		case strings.HasPrefix(u.Path, "/playlist"):
			p.PlaylistID = q.Get("list")
		case strings.HasPrefix(u.Path, "/shorts/"):
			p.VideoID = strings.TrimPrefix(u.Path, "/shorts/")
		}

	default:
		return p
	}

	// Clean up extracted IDs.
	p.VideoID = strings.TrimSpace(p.VideoID)
	p.PlaylistID = strings.TrimSpace(p.PlaylistID)
	if len(p.VideoID) > maxIDLen {
		p.VideoID = ""
	}
	if len(p.PlaylistID) > maxIDLen {
		p.PlaylistID = ""
	}

	switch {
	case p.VideoID != "" && p.PlaylistID != "":
		p.Type = URLAmbiguous
	case p.PlaylistID != "":
		p.Type = URLPlaylist
	case p.VideoID != "":
		p.Type = URLSingle
	}

	return p
}
