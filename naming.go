package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	bracketNoise = regexp.MustCompile(`\[[^\]]*\]`)
	parenNoise   = regexp.MustCompile(`(?i)\(\s*(official\s*(video|audio|music\s*video|visualizer|lyric\s*video)?|lyrics?|lyric\s*video|hd|hq|4k|1080p|720p|audio|music\s*video|visualizer|video\s*oficial|videoclip|clip\s*officiel)\s*\)`)
	nonAlnum     = regexp.MustCompile(`[^a-z0-9]+`)
	multiUnder   = regexp.MustCompile(`_+`)
	unsafeChars  = regexp.MustCompile(`[<>:"/\\|?*]+`)
)

const maxFilenameLen = 60
const maxFolderNameLen = 80

// SanitizeFilename converts a YouTube video title into a CD-safe filename.
// It transliterates diacritics, strips noise tags, lowercases, collapses
// non-alphanumeric runs into underscores, and truncates to 60 chars.
func SanitizeFilename(title string, trackNum int) string {
	s := transliterate(title)
	s = bracketNoise.ReplaceAllString(s, "")
	s = parenNoise.ReplaceAllString(s, "")
	s = strings.ToLower(s)
	s = nonAlnum.ReplaceAllString(s, "_")
	s = multiUnder.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")

	if len(s) > maxFilenameLen {
		s = s[:maxFilenameLen]
		s = strings.TrimRight(s, "_")
	}

	return fmt.Sprintf("%02d_%s.mp3", trackNum, s)
}

// SanitizeFolderName converts a playlist title into a filesystem-safe folder name.
func SanitizeFolderName(title string) string {
	s := transliterate(title)
	s = bracketNoise.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	if s == "" {
		return "Untitled"
	}
	s = unsafeChars.ReplaceAllString(s, "_")
	s = multiUnder.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_. ")

	if len(s) > maxFolderNameLen {
		s = s[:maxFolderNameLen]
		s = strings.TrimRight(s, "_. ")
	}
	if s == "" {
		return "Untitled"
	}
	return s
}

// cyrillicReplacer maps Bulgarian (and common Russian) Cyrillic to Latin.
// Multi-char mappings (ж→zh, щ→sht, etc.) must come before single-char ones
// so strings.NewReplacer matches them correctly.
var cyrillicReplacer = strings.NewReplacer(
	// Bulgarian multi-char mappings (uppercase first, then lowercase)
	"Щ", "Sht", "щ", "sht",
	"Ж", "Zh", "ж", "zh",
	"Ц", "Ts", "ц", "ts",
	"Ч", "Ch", "ч", "ch",
	"Ш", "Sh", "ш", "sh",
	"Ю", "Yu", "ю", "yu",
	"Я", "Ya", "я", "ya",
	// Extra Russian multi-char
	"Ё", "Yo", "ё", "yo",

	// Single-char mappings
	"А", "A", "а", "a",
	"Б", "B", "б", "b",
	"В", "V", "в", "v",
	"Г", "G", "г", "g",
	"Д", "D", "д", "d",
	"Е", "E", "е", "e",
	"З", "Z", "з", "z",
	"И", "I", "и", "i",
	"Й", "Y", "й", "y",
	"К", "K", "к", "k",
	"Л", "L", "л", "l",
	"М", "M", "м", "m",
	"Н", "N", "н", "n",
	"О", "O", "о", "o",
	"П", "P", "п", "p",
	"Р", "R", "р", "r",
	"С", "S", "с", "s",
	"Т", "T", "т", "t",
	"У", "U", "у", "u",
	"Ф", "F", "ф", "f",
	"Х", "H", "х", "h",
	"Ъ", "A", "ъ", "a",
	"Ь", "Y", "ь", "y",
	// Extra Russian single-char
	"Э", "E", "э", "e",
	"Ы", "Y", "ы", "y",
)

func transliterate(s string) string {
	// First pass: Cyrillic → Latin (before NFD, since Cyrillic doesn't decompose)
	s = cyrillicReplacer.Replace(s)
	// Second pass: strip diacritics (é→e, ü→u, etc.)
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s) // errors only on invalid UTF-8; safe to ignore
	return result
}
