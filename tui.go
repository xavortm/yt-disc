package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	checkStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	okStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

// View states

type viewState int

const (
	viewLoading viewState = iota
	viewPicker
	viewDownload
	viewDiscList
	viewDiscDetail
)

// String returns a human-readable label for the view state.
func (v viewState) String() string {
	switch v {
	case viewLoading:
		return "loading"
	case viewPicker:
		return "picker"
	case viewDownload:
		return "download"
	case viewDiscList:
		return "disc-list"
	case viewDiscDetail:
		return "disc-detail"
	default:
		return "unknown"
	}
}

// Messages

type downloadedMsg struct {
	idx int
	err error
}

type discsLoadedMsg []Disc

type songDiscardedMsg struct{ err error }

// songsProbed carries songs with durations filled in.
type songsProbed struct {
	songs []Song
}

type errMsg struct{ err error }

// tickMsg drives the elapsed-time display during loading.
type tickMsg time.Time

// fetchDoneMsg carries the result of a metadata fetch (initial or add-to-disc).
type fetchDoneMsg struct {
	name   string
	videos []VideoMeta
	single bool // true = single video, auto-download without picker
	err    error
}

// Model

type model struct {
	view viewState

	// Picker
	playlistName string
	videos       []VideoMeta
	selected     map[int]bool
	cursor       int
	scroll       int

	// Download
	dlIndices    []int
	dlPos        int
	dlLog        []string
	dlStartTrack int
	spinner      spinner.Model

	// Disc browser
	discs      []Disc
	discCursor int

	// Disc detail
	disc       *Disc
	songCursor int

	// URL input (Flow 3: add songs to existing disc)
	urlInput   textinput.Model
	inputMode  bool
	targetDisc string

	// Config
	cfg appConfig

	// Initial fetch
	fetchURL  string    // URL to fetch on init (empty = skip)
	fetchType ParsedURL // parsed URL for initial fetch

	// Loading state
	loadingStart time.Time

	// Window
	width  int
	height int

	loading  bool
	err      error
	quitting bool
}

// Layout constants.
const (
	headerLines      = 7  // lines reserved above the picker list
	maxURLDisplay    = 60 // max chars shown for a URL in the loading view
	pickerTitleWidth = 55
	dlTitleWidth     = 50
	discNameWidth    = 28
	songNameWidth    = 48
)

// newBaseModel creates a model with shared defaults (spinner, text input, config).
func newBaseModel(cfg appConfig) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	ti := textinput.New()
	ti.Placeholder = "Paste YouTube URL..."
	ti.CharLimit = 256
	return model{
		selected:     make(map[int]bool),
		dlStartTrack: 1,
		spinner:      s,
		urlInput:     ti,
		cfg:          cfg,
	}
}

// newFetchModel creates a model that fetches metadata on Init.
func newFetchModel(rawURL string, parsed ParsedURL, cfg appConfig) model {
	m := newBaseModel(cfg)
	m.view = viewLoading
	m.fetchURL = rawURL
	m.fetchType = parsed
	m.loadingStart = time.Now()
	return m
}

func newListModel(cfg appConfig) model {
	m := newBaseModel(cfg)
	m.view = viewDiscList
	return m
}

func (m model) Init() tea.Cmd {
	switch m.view {
	case viewLoading:
		return tea.Batch(m.spinner.Tick, timerTick(), fetchMetaCmd(m.fetchURL, m.fetchType))
	case viewDiscList:
		return loadDiscsCmd(m.cfg.outputDir)
	}
	return nil
}

// Update dispatches messages to the appropriate handler.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case downloadedMsg:
		return m.handleDownloaded(msg)

	case discsLoadedMsg:
		m.discs = []Disc(msg)
		return m, nil

	case songDiscardedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else if m.disc != nil {
			songs, err := ListSongs(m.disc.Path)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.disc.Songs = songs
			if m.songCursor >= len(songs) && m.songCursor > 0 {
				m.songCursor--
			}
		}
		return m, nil

	case songsProbed:
		if m.disc != nil {
			m.disc.Songs = msg.songs
		}
		m.loading = false
		return m, nil

	case fetchDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		if len(msg.videos) == 0 {
			m.err = fmt.Errorf("no videos found in playlist")
			return m, nil
		}
		m.playlistName = msg.name
		m.videos = msg.videos
		// Single video from disc-detail: auto-download without picker.
		if msg.single && len(msg.videos) == 1 {
			m.selected = map[int]bool{0: true}
			return m.beginDownload()
		}
		m.selected = make(map[int]bool)
		m.cursor = 0
		m.scroll = 0
		m.view = viewPicker
		return m, nil

	case errMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case tickMsg:
		if m.view == viewLoading || m.loading {
			return m, timerTick()
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// URL input mode
	if m.inputMode {
		return m.handleInputKey(msg)
	}

	// Loading: only ctrl+c
	if m.loading || m.view == viewLoading {
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	}

	// Active download: only ctrl+c
	if m.view == viewDownload && m.dlPos < len(m.dlIndices) {
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	}

	// Download complete
	if m.view == viewDownload && m.dlPos >= len(m.dlIndices) {
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "b":
			if m.targetDisc != "" && m.disc != nil {
				m.view = viewDiscDetail
				songs, err := ListSongs(m.disc.Path)
				if err != nil {
					m.err = err
					return m, nil
				}
				m.disc.Songs = songs
				m.targetDisc = ""
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, probeSongsCmd(songs))
			}
		}
		return m, nil
	}

	// Clear error on any key
	if m.err != nil {
		m.err = nil
		return m, nil
	}

	switch m.view {
	case viewPicker:
		return m.updatePicker(msg)
	case viewDiscList:
		return m.updateDiscList(msg)
	case viewDiscDetail:
		return m.updateDiscDetail(msg)
	}
	return m, nil
}

func (m model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.inputMode = false
		m.urlInput.Reset()
		return m, nil
	case "enter":
		raw := strings.TrimSpace(m.urlInput.Value())
		m.inputMode = false
		m.urlInput.Reset()
		if raw == "" {
			return m, nil
		}
		parsed := ParseYouTubeURL(raw)
		if parsed.Type == URLInvalid {
			m.err = fmt.Errorf("invalid YouTube URL")
			return m, nil
		}
		if m.disc != nil {
			m.targetDisc = m.disc.Path
			n, err := NextTrackNum(m.disc.Path)
			if err != nil {
				m.err = fmt.Errorf("reading track numbers: %w", err)
				return m, nil
			}
			m.dlStartTrack = n
		}
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, fetchMetaCmd(raw, parsed))
	default:
		var cmd tea.Cmd
		m.urlInput, cmd = m.urlInput.Update(msg)
		return m, cmd
	}
}

// --- Picker ---

func (m model) updatePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(m.videos)-1 {
			m.cursor++
			m.fixScroll()
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.fixScroll()
		}
	case " ":
		if m.selected[m.cursor] {
			delete(m.selected, m.cursor)
		} else {
			m.selected[m.cursor] = true
		}
	case "a":
		for i := range m.videos {
			m.selected[i] = true
		}
	case "n":
		m.selected = make(map[int]bool)
	case "s", "enter":
		if len(m.selected) == 0 {
			return m, nil
		}
		return m.beginDownload()
	}
	return m, nil
}

func (m *model) fixScroll() {
	vis := m.visibleLines()
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+vis {
		m.scroll = m.cursor - vis + 1
	}
}

func (m model) visibleLines() int {
	h := m.height - headerLines
	if h < 5 {
		return 20
	}
	return h
}

func (m model) beginDownload() (model, tea.Cmd) {
	var indices []int
	for i := range m.videos {
		if m.selected[i] {
			indices = append(indices, i)
		}
	}

	// Resolve disc path once before starting downloads.
	discPath := m.targetDisc
	if discPath == "" {
		var err error
		discPath, err = CreateDisc(m.cfg.outputDir, SanitizeFolderName(m.playlistName))
		if err != nil {
			m.err = err
			return m, nil
		}
	}

	m.dlIndices = indices
	m.dlPos = 0
	m.dlLog = nil
	m.view = viewDownload
	m.targetDisc = discPath
	return m, tea.Batch(m.spinner.Tick, m.dlCmd())
}

func (m model) dlCmd() tea.Cmd {
	if m.dlPos >= len(m.dlIndices) {
		return nil
	}
	idx := m.dlIndices[m.dlPos]
	video := m.videos[idx]
	trackNum := m.dlStartTrack + m.dlPos
	filename := SanitizeFilename(video.Title, trackNum)
	discPath := m.targetDisc
	cfg := m.cfg

	return func() tea.Msg {
		dest := filepath.Join(discPath, filename)
		err := DownloadAudio(context.Background(), video.URL, dest, cfg.bitrate)
		return downloadedMsg{idx: idx, err: err}
	}
}

func (m model) handleDownloaded(msg downloadedMsg) (model, tea.Cmd) {
	video := m.videos[msg.idx]
	if msg.err != nil {
		m.dlLog = append(m.dlLog, errStyle.Render("✗ ")+video.Title+": "+msg.err.Error())
	} else {
		m.dlLog = append(m.dlLog, okStyle.Render("✓ ")+video.Title)
	}
	m.dlPos++
	if m.dlPos >= len(m.dlIndices) {
		return m, nil
	}
	return m, m.dlCmd()
}

// --- Disc List ---

func (m model) updateDiscList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "j", "down":
		if m.discCursor < len(m.discs)-1 {
			m.discCursor++
		}
	case "k", "up":
		if m.discCursor > 0 {
			m.discCursor--
		}
	case "enter":
		if len(m.discs) > 0 {
			d := m.discs[m.discCursor]
			m.disc = &d
			m.songCursor = 0
			m.view = viewDiscDetail
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, probeSongsCmd(d.Songs))
		}
	}
	return m, nil
}

// --- Disc Detail ---

func (m model) updateDiscDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "b", "esc":
		m.disc = nil
		m.view = viewDiscList
		return m, loadDiscsCmd(m.cfg.outputDir)
	case "j", "down":
		if m.disc != nil && m.songCursor < len(m.disc.Songs)-1 {
			m.songCursor++
		}
	case "k", "up":
		if m.songCursor > 0 {
			m.songCursor--
		}
	case "x":
		if m.disc != nil && len(m.disc.Songs) > 0 {
			path := m.disc.Songs[m.songCursor].Path
			return m, func() tea.Msg {
				return songDiscardedMsg{err: DiscardSong(path)}
			}
		}
	case "u":
		m.inputMode = true
		m.urlInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

// --- Commands ---

func loadDiscsCmd(dir string) tea.Cmd {
	return func() tea.Msg {
		discs, err := ListDiscs(dir)
		if err != nil {
			return errMsg{err: err}
		}
		return discsLoadedMsg(discs)
	}
}

// probeSongsCmd probes durations for all songs in the background.
func probeSongsCmd(songs []Song) tea.Cmd {
	return func() tea.Msg {
		probed := make([]Song, len(songs))
		copy(probed, songs)
		for i := range probed {
			if dur, err := ProbeDuration(probed[i].Path); err == nil {
				probed[i].Duration = dur
			}
		}
		return songsProbed{songs: probed}
	}
}

// fetchMetaCmd fetches playlist or video metadata and returns a fetchDoneMsg.
func fetchMetaCmd(rawURL string, parsed ParsedURL) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		switch parsed.Type {
		case URLPlaylist:
			name, videos, err := FetchPlaylistMeta(ctx, rawURL)
			return fetchDoneMsg{name: name, videos: videos, err: err}
		case URLAmbiguous:
			name, videos, err := FetchPlaylistMeta(ctx, rawURL)
			if err == nil {
				return fetchDoneMsg{name: name, videos: videos}
			}
			video, err2 := FetchVideoMeta(ctx, rawURL)
			if err2 != nil {
				return fetchDoneMsg{err: err2}
			}
			return fetchDoneMsg{name: "Single Video", videos: []VideoMeta{video}, single: true}
		case URLSingle:
			video, err := FetchVideoMeta(ctx, rawURL)
			if err != nil {
				return fetchDoneMsg{err: err}
			}
			return fetchDoneMsg{name: "Single Video", videos: []VideoMeta{video}, single: true}
		default:
			return fetchDoneMsg{err: fmt.Errorf("invalid URL")}
		}
	}
}

func timerTick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// --- Views ---

// View renders the current state.
func (m model) View() string {
	if m.quitting {
		return ""
	}

	var content string
	switch m.view {
	case viewLoading:
		content = m.viewLoading()
	case viewPicker:
		content = m.viewPicker()
	case viewDownload:
		content = m.viewDownload()
	case viewDiscList:
		content = m.viewDiscList()
	case viewDiscDetail:
		content = m.viewDiscDetail()
	}

	if m.err != nil {
		content += "\n" + errStyle.Render("  Error: "+m.err.Error())
		content += "\n" + dimStyle.Render("  Press any key to dismiss")
	}

	if m.loading {
		content += "\n\n  " + m.spinner.View() + " Fetching metadata..."
	}

	if m.inputMode {
		content += "\n\n  " + m.urlInput.View()
	}

	return content
}

func (m model) viewLoading() string {
	var b strings.Builder

	elapsed := time.Since(m.loadingStart).Truncate(time.Second)
	urlHint := m.fetchURL
	if len(urlHint) > maxURLDisplay {
		urlHint = urlHint[:maxURLDisplay-3] + "..."
	}

	b.WriteString(titleStyle.Render("♫ yt-disc") + "\n\n")
	b.WriteString(fmt.Sprintf("  %s Fetching metadata...  %s\n", m.spinner.View(), dimStyle.Render(elapsed.String())))
	b.WriteString(dimStyle.Render("  "+urlHint) + "\n")

	switch m.fetchType.Type {
	case URLPlaylist:
		b.WriteString(dimStyle.Render("  Type: playlist") + "\n")
	case URLAmbiguous:
		b.WriteString(dimStyle.Render("  Type: video+playlist (trying playlist first)") + "\n")
	case URLSingle:
		b.WriteString(dimStyle.Render("  Type: single video") + "\n")
	}

	b.WriteString("\n" + dimStyle.Render("  This can take a while for large playlists."))
	b.WriteString("\n" + dimStyle.Render("  Press q or Ctrl+C to cancel"))

	return b.String()
}

func (m model) viewPicker() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("♫ "+m.playlistName) + "\n")

	var selDur time.Duration
	for i := range m.videos {
		if m.selected[i] {
			selDur += m.videos[i].Duration
		}
	}
	stats := fmt.Sprintf("  %d selected  %s / %s",
		len(m.selected), fmtDuration(selDur), fmtDuration(AudioCDCapacity))
	if selDur > AudioCDCapacity {
		stats += warnStyle.Render("  ⚠ OVER CAPACITY")
	}
	b.WriteString(dimStyle.Render(stats) + "\n\n")

	vis := m.visibleLines()
	end := m.scroll + vis
	if end > len(m.videos) {
		end = len(m.videos)
	}
	for i := m.scroll; i < end; i++ {
		v := m.videos[i]
		prefix := "  "
		if i == m.cursor {
			prefix = cursorStyle.Render("▸ ")
		}
		check := "[ ] "
		if m.selected[i] {
			check = checkStyle.Render("[✓] ")
		}
		title := truncate(v.Title, pickerTitleWidth)
		dur := dimStyle.Render(fmtDuration(v.Duration))
		b.WriteString(fmt.Sprintf("%s%s%-55s %s\n", prefix, check, title, dur))
	}

	b.WriteString("\n" + dimStyle.Render("  j/k: navigate  space: toggle  a: all  n: none  s: save  q: quit"))
	return b.String()
}

func (m model) viewDownload() string {
	var b strings.Builder

	target := m.targetDisc
	if target == "" {
		target = filepath.Join(m.cfg.outputDir, SanitizeFolderName(m.playlistName))
	}
	b.WriteString(titleStyle.Render("⬇ Downloading to "+target) + "\n\n")

	if m.dlPos < len(m.dlIndices) {
		idx := m.dlIndices[m.dlPos]
		b.WriteString(fmt.Sprintf("  %s %s (%d/%d)\n\n",
			m.spinner.View(),
			truncate(m.videos[idx].Title, dlTitleWidth),
			m.dlPos+1, len(m.dlIndices)))
	} else {
		b.WriteString(okStyle.Render("  ✓ All downloads complete!") + "\n\n")
	}

	for _, line := range m.dlLog {
		b.WriteString("  " + line + "\n")
	}

	if m.dlPos >= len(m.dlIndices) {
		hint := "  Press q to quit"
		if m.targetDisc != "" {
			hint = "  Press b to go back, q to quit"
		}
		b.WriteString("\n" + dimStyle.Render(hint))
	} else {
		b.WriteString("\n" + dimStyle.Render("  Ctrl+C to cancel"))
	}
	return b.String()
}

func (m model) viewDiscList() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("💿 Your Discs") + "  " + dimStyle.Render(m.cfg.outputDir) + "\n\n")

	if len(m.discs) == 0 {
		b.WriteString(dimStyle.Render("  No discs found.") + "\n")
	}
	for i, d := range m.discs {
		prefix := "  "
		if i == m.discCursor {
			prefix = cursorStyle.Render("▸ ")
		}
		b.WriteString(fmt.Sprintf("%s%-30s  %d songs\n",
			prefix, truncate(d.Name, discNameWidth), len(d.Songs)))
	}

	b.WriteString("\n" + dimStyle.Render("  j/k: navigate  enter: open  q: quit"))
	return b.String()
}

func (m model) viewDiscDetail() string {
	var b strings.Builder

	if m.disc == nil {
		return ""
	}

	total := TotalDuration(m.disc.Songs)
	b.WriteString(titleStyle.Render(fmt.Sprintf("💿 %s  %d songs  %s / %s",
		m.disc.Name, len(m.disc.Songs),
		fmtDuration(total), fmtDuration(AudioCDCapacity))) + "\n")
	if total > AudioCDCapacity {
		b.WriteString(warnStyle.Render("  ⚠ OVER CAPACITY") + "\n")
	}
	b.WriteString("\n")

	if len(m.disc.Songs) == 0 {
		b.WriteString(dimStyle.Render("  No songs.") + "\n")
	}
	for i, s := range m.disc.Songs {
		prefix := "  "
		if i == m.songCursor {
			prefix = cursorStyle.Render("▸ ")
		}
		dur := dimStyle.Render(fmtDuration(s.Duration))
		b.WriteString(fmt.Sprintf("%s%-50s %s\n",
			prefix, truncate(s.Name, songNameWidth), dur))
	}

	b.WriteString("\n" + dimStyle.Render("  j/k: navigate  x: discard  u: add URL  b: back  q: quit"))
	return b.String()
}

// --- Helpers ---

func fmtDuration(d time.Duration) string {
	if d == 0 {
		return "?:??"
	}
	total := int(d.Seconds())
	m := total / 60
	s := total % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return string(r[:max])
	}
	return string(r[:max-3]) + "..."
}
