package main

import "testing"

func TestParseYouTubeURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantType  URLType
		wantVideo string
		wantList  string
	}{
		{
			name:     "playlist page",
			url:      "https://www.youtube.com/playlist?list=PLrAXtmErZgOeiKm4sgNOknGvNjby9efdf",
			wantType: URLPlaylist,
			wantList: "PLrAXtmErZgOeiKm4sgNOknGvNjby9efdf",
		},
		{
			name:      "single video",
			url:       "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			wantType:  URLSingle,
			wantVideo: "dQw4w9WgXcQ",
		},
		{
			name:      "video in playlist",
			url:       "https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PLrAXtmErZgOeiKm4sgNOknGvNjby9efdf",
			wantType:  URLAmbiguous,
			wantVideo: "dQw4w9WgXcQ",
			wantList:  "PLrAXtmErZgOeiKm4sgNOknGvNjby9efdf",
		},
		{
			name:      "short url",
			url:       "https://youtu.be/dQw4w9WgXcQ",
			wantType:  URLSingle,
			wantVideo: "dQw4w9WgXcQ",
		},
		{
			name:      "short url with list",
			url:       "https://youtu.be/dQw4w9WgXcQ?list=PLrAXtmErZgOeiKm4sgNOknGvNjby9efdf",
			wantType:  URLAmbiguous,
			wantVideo: "dQw4w9WgXcQ",
			wantList:  "PLrAXtmErZgOeiKm4sgNOknGvNjby9efdf",
		},
		{
			name:      "shorts url",
			url:       "https://www.youtube.com/shorts/abc123",
			wantType:  URLSingle,
			wantVideo: "abc123",
		},
		{
			name:     "music.youtube.com playlist",
			url:      "https://music.youtube.com/playlist?list=PLxyz",
			wantType: URLPlaylist,
			wantList: "PLxyz",
		},
		{
			name:     "invalid - not a url",
			url:      "not-a-url",
			wantType: URLInvalid,
		},
		{
			name:     "invalid - non-youtube domain",
			url:      "https://vimeo.com/12345",
			wantType: URLInvalid,
		},
		{
			name:     "invalid - empty",
			url:      "",
			wantType: URLInvalid,
		},
		{
			name:      "mobile youtube",
			url:       "https://m.youtube.com/watch?v=abc123",
			wantType:  URLSingle,
			wantVideo: "abc123",
		},
		{
			name:     "youtube watch with no v param",
			url:      "https://www.youtube.com/watch",
			wantType: URLInvalid,
		},
		{
			name:      "music.youtube.com watch",
			url:       "https://music.youtube.com/watch?v=xyz789",
			wantType:  URLSingle,
			wantVideo: "xyz789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseYouTubeURL(tt.url)
			if got.Type != tt.wantType {
				t.Errorf("Type: got %v, want %v", got.Type, tt.wantType)
			}
			if got.VideoID != tt.wantVideo {
				t.Errorf("VideoID: got %q, want %q", got.VideoID, tt.wantVideo)
			}
			if got.PlaylistID != tt.wantList {
				t.Errorf("PlaylistID: got %q, want %q", got.PlaylistID, tt.wantList)
			}
		})
	}
}
