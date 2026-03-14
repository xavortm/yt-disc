package main

import "testing"

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		title    string
		trackNum int
		want     string
	}{
		{
			title:    "Beyoncé — Crazy in Love [Official Video]",
			trackNum: 1,
			want:     "01_beyonce_crazy_in_love.mp3",
		},
		{
			title:    "Daft Punk - Get Lucky (feat. Pharrell Williams)",
			trackNum: 2,
			want:     "02_daft_punk_get_lucky_feat_pharrell_williams.mp3",
		},
		{
			title:    "Song Title (Official Music Video) [HD]",
			trackNum: 3,
			want:     "03_song_title.mp3",
		},
		{
			title:    "Artist - Song (Remix)",
			trackNum: 4,
			want:     "04_artist_song_remix.mp3",
		},
		{
			title:    "Artist - Song (Live)",
			trackNum: 5,
			want:     "05_artist_song_live.mp3",
		},
		{
			title:    "Song (Lyrics)",
			trackNum: 6,
			want:     "06_song.mp3",
		},
		{
			title:    "   Lots   of   spaces   ",
			trackNum: 7,
			want:     "07_lots_of_spaces.mp3",
		},
		{
			title:    "Ünder Prëssure (Official Audio) [Remastered]",
			trackNum: 8,
			want:     "08_under_pressure.mp3",
		},
		{
			title:    "",
			trackNum: 1,
			want:     "01_untitled.mp3",
		},
		{
			title:    "A Very Long Song Title That Exceeds The Maximum Length Allowed For CD Safe Filenames On Disk",
			trackNum: 10,
			want:     "10_a_very_long_song_title_that_exceeds_the_maximum_length_allow.mp3",
		},
		{
			title:    "Artist – Song Name (HD)",
			trackNum: 11,
			want:     "11_artist_song_name.mp3",
		},
		{
			title:    "Great Track (Official Visualizer)",
			trackNum: 12,
			want:     "12_great_track.mp3",
		},
		{
			title:    "Cool Song (Acoustic)",
			trackNum: 13,
			want:     "13_cool_song_acoustic.mp3",
		},
		// Bulgarian Cyrillic titles
		{
			title:    "Щурците - Вятър ме носи",
			trackNum: 14,
			want:     "14_shturtsite_vyatar_me_nosi.mp3",
		},
		{
			title:    "здрасти свят",
			trackNum: 15,
			want:     "15_zdrasti_svyat.mp3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := SanitizeFilename(tt.title, tt.trackNum)
			if got != tt.want {
				t.Errorf("SanitizeFilename(%q, %d)\n  got  %q\n  want %q",
					tt.title, tt.trackNum, got, tt.want)
			}
		})
	}
}

func TestSanitizeFolderName(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"My Playlist", "My Playlist"},
		{"Playlist: Best of 2024", "Playlist_ Best of 2024"},
		{"  ", "Untitled"},
		{"", "Untitled"},
		{"Beyoncé's Greatest Hits [Deluxe]", "Beyonce's Greatest Hits [Deluxe]"},
		{"Normal Name", "Normal Name"},
		{"Has/Slashes\\And:Colons", "Has_Slashes_And_Colons"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := SanitizeFolderName(tt.title)
			if got != tt.want {
				t.Errorf("SanitizeFolderName(%q)\n  got  %q\n  want %q",
					tt.title, got, tt.want)
			}
		})
	}
}

func TestTransliterate(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Beyoncé", "Beyonce"},
		{"Ünder", "Under"},
		{"naïve", "naive"},
		{"plain ascii", "plain ascii"},
		// Bulgarian Cyrillic
		{"здрасти", "zdrasti"},
		{"Щурците", "Shturtsite"},
		{"АБВГД", "ABVGD"},
		{"жълт", "zhalt"},
		{"Южна нощ", "Yuzhna nosht"},
		{"победа", "pobeda"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := transliterate(tt.in)
			if got != tt.want {
				t.Errorf("transliterate(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
