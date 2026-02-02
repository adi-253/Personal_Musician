// Package main provides the TUI (Terminal User Interface) for Personal Musician.
// This module uses Bubble Tea to create an interactive terminal interface.
package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// View represents the current active view in the TUI.
type View int

const (
	ViewLibrary View = iota // Default view - show local music files
	ViewSearch              // Search input view
	ViewResults             // Search results view
)

// Styles for the TUI
var (
	// Color palette
	primaryColor   = lipgloss.Color("#7C3AED") // Purple
	secondaryColor = lipgloss.Color("#10B981") // Green
	accentColor    = lipgloss.Color("#F59E0B") // Amber
	textColor      = lipgloss.Color("#E5E7EB") // Light gray
	mutedColor     = lipgloss.Color("#6B7280") // Muted gray

	// Title style
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	// Header style
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textColor).
			Background(primaryColor).
			Padding(0, 1)

	// Selected item style
	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor)

	// Normal item style
	normalStyle = lipgloss.NewStyle().
			Foreground(textColor)

	// Muted style for secondary info
	mutedStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Status bar style
	statusStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	// Now playing style
	nowPlayingStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor)

	// Help style
	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	// Box style
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1)
)

// Model represents the application state for Bubble Tea.
type Model struct {
	// Dependencies
	player     *Player
	downloader *Downloader
	ctx        context.Context
	cancelFunc context.CancelFunc

	// View state
	currentView View
	width       int
	height      int

	// Library view state
	libraryFiles  []MusicFile
	libraryCursor int

	// Search state
	searchInput  textinput.Model
	searchQuery  string
	isSearching  bool
	searchError  string

	// Search results state (YouTube results)
	youtubeResults []SearchResult
	resultsCursor  int

	// Download state
	downloadProgress progress.Model
	downloadSpinner  spinner.Model

	// Status message
	statusMessage string
	statusTimer   int

	// Playback refresh ticker
	tickCount int
}

// Messages for Bubble Tea
type (
	// tickMsg is sent periodically to update the UI.
	tickMsg time.Time

	// youtubeSearchCompleteMsg is sent when a YouTube search completes.
	youtubeSearchCompleteMsg struct {
		results []SearchResult
		err     error
	}

	// libraryRefreshMsg is sent when the library needs refreshing.
	libraryRefreshMsg []MusicFile

	// statusMsg is sent to display a temporary status message.
	statusMsg string

	// downloadCompleteMsg is sent when a download completes.
	downloadCompleteMsg struct{}
)

// NewModel creates a new TUI model with all dependencies.
func NewModel(player *Player, downloader *Downloader) Model {
	// Initialize text input for search
	ti := textinput.New()
	ti.Placeholder = "Search for music on YouTube..."
	ti.CharLimit = 100
	ti.Width = 50

	// Initialize progress bar
	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 30

	// Initialize spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(primaryColor)

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	return Model{
		player:           player,
		downloader:       downloader,
		ctx:              ctx,
		cancelFunc:       cancel,
		currentView:      ViewLibrary,
		searchInput:      ti,
		downloadProgress: prog,
		downloadSpinner:  sp,
	}
}

// Init initializes the Bubble Tea program.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.refreshLibrary(),
		m.tickCmd(),
	)
}

// Update handles incoming messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.downloadProgress.Width = msg.Width - 20

	case tickMsg:
		m.tickCount++
		// Decrement status timer
		if m.statusTimer > 0 {
			m.statusTimer--
			if m.statusTimer == 0 {
				m.statusMessage = ""
			}
		}
		
		// Check if download completed and refresh library
		if !m.downloader.IsDownloading() {
			dp := m.downloader.GetProgress()
			if dp.Progress >= 100 && len(dp.Files) > 0 {
				return m, tea.Batch(m.tickCmd(), m.refreshLibrary())
			}
		}
		
		return m, m.tickCmd()

	case youtubeSearchCompleteMsg:
		m.isSearching = false
		if msg.err != nil {
			m.searchError = msg.err.Error()
			m.youtubeResults = nil
		} else if len(msg.results) == 0 {
			m.searchError = "No results found"
			m.youtubeResults = nil
		} else {
			m.youtubeResults = msg.results
			m.resultsCursor = 0
			m.currentView = ViewResults
			m.searchError = ""
		}

	case libraryRefreshMsg:
		m.libraryFiles = msg
		m.player.SetPlaylist(msg)
		if m.libraryCursor >= len(msg) && len(msg) > 0 {
			m.libraryCursor = len(msg) - 1
		}

	case statusMsg:
		m.statusMessage = string(msg)
		m.statusTimer = 10 // Show for ~5 seconds (10 ticks at 500ms)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.downloadSpinner, cmd = m.downloadSpinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update text input if in search view
	if m.currentView == ViewSearch {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleKeyPress processes keyboard input.
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys (work in all views)
	switch msg.String() {
	case "ctrl+c", "q":
		m.cancelFunc()
		return m, tea.Quit

	case " ": // Space - toggle pause
		if m.currentView != ViewSearch { // Don't capture space in search input
			m.player.TogglePause()
			return m, nil
		}

	case "left": // Previous song
		if m.currentView != ViewSearch {
			if err := m.player.PrevSong(); err == nil {
				return m, m.refreshLibrary()
			}
			return m, nil
		}

	case "right": // Next song
		if m.currentView != ViewSearch {
			if err := m.player.NextSong(); err == nil {
				return m, m.refreshLibrary()
			}
			return m, nil
		}

	case "s": // Open search
		if m.currentView != ViewSearch {
			m.currentView = ViewSearch
			m.searchInput.Focus()
			m.searchInput.SetValue("")
			return m, textinput.Blink
		}

	case "tab": // Switch views
		if m.currentView == ViewSearch {
			m.currentView = ViewLibrary
			m.searchInput.Blur()
		} else if len(m.youtubeResults) > 0 {
			if m.currentView == ViewLibrary {
				m.currentView = ViewResults
			} else {
				m.currentView = ViewLibrary
			}
		}
		return m, nil

	case "esc": // Back to library
		if m.currentView != ViewLibrary {
			m.currentView = ViewLibrary
			m.searchInput.Blur()
			return m, nil
		}
	}

	// View-specific keys
	switch m.currentView {
	case ViewSearch:
		return m.handleSearchKeys(msg)
	case ViewLibrary:
		return m.handleLibraryKeys(msg)
	case ViewResults:
		return m.handleResultsKeys(msg)
	}

	return m, nil
}

// handleSearchKeys handles keys in the search view.
func (m Model) handleSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		query := strings.TrimSpace(m.searchInput.Value())
		if query != "" {
			m.searchQuery = query
			m.isSearching = true
			m.searchError = ""
			return m, m.performYouTubeSearch(query)
		}
	}

	// Let text input handle most keys
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

// handleLibraryKeys handles keys in the library view.
func (m Model) handleLibraryKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.libraryCursor > 0 {
			m.libraryCursor--
		}
	case "down", "j":
		if m.libraryCursor < len(m.libraryFiles)-1 {
			m.libraryCursor++
		}
	case "enter":
		if len(m.libraryFiles) > 0 && m.libraryCursor < len(m.libraryFiles) {
			if err := m.player.PlayIndex(m.libraryCursor); err != nil {
				return m, func() tea.Msg { return statusMsg("Error: " + err.Error()) }
			}
			return m, func() tea.Msg { return statusMsg("Now playing: " + m.libraryFiles[m.libraryCursor].Name) }
		}
	}
	return m, nil
}

// handleResultsKeys handles keys in the search results view (YouTube).
func (m Model) handleResultsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.resultsCursor > 0 {
			m.resultsCursor--
		}
	case "down", "j":
		if m.resultsCursor < len(m.youtubeResults)-1 {
			m.resultsCursor++
		}
	case "enter":
		if len(m.youtubeResults) > 0 && m.resultsCursor < len(m.youtubeResults) {
			result := m.youtubeResults[m.resultsCursor]
			if err := m.downloader.DownloadFromYouTube(m.ctx, result.VideoID, result.Title); err != nil {
				return m, func() tea.Msg { return statusMsg("Download error: " + err.Error()) }
			}
			return m, tea.Batch(
				m.downloadSpinner.Tick,
				func() tea.Msg { return statusMsg("Downloading: " + result.Title) },
			)
		}
	}
	return m, nil
}

// View renders the TUI.
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var sections []string

	// Title
	title := titleStyle.Render("🎵 Personal Musician")
	sections = append(sections, title)

	// Now playing bar
	sections = append(sections, m.renderNowPlaying())

	// Main content based on current view
	switch m.currentView {
	case ViewSearch:
		sections = append(sections, m.renderSearchView())
	case ViewLibrary:
		sections = append(sections, m.renderLibraryView())
	case ViewResults:
		sections = append(sections, m.renderResultsView())
	}

	// Download progress (if downloading)
	if m.downloader.IsDownloading() {
		sections = append(sections, m.renderDownloadProgress())
	}

	// Status message
	if m.statusMessage != "" {
		sections = append(sections, statusStyle.Render(m.statusMessage))
	}

	// Help bar
	sections = append(sections, m.renderHelp())

	return strings.Join(sections, "\n")
}

// renderNowPlaying renders the now playing section.
func (m Model) renderNowPlaying() string {
	state := m.player.GetState()

	if !state.IsPlaying && state.CurrentFile == "" {
		return mutedStyle.Render("♪ No song playing")
	}

	// Get current file info
	files := m.player.GetPlaylist()
	var songName string
	if state.CurrentIndex >= 0 && state.CurrentIndex < len(files) {
		songName = files[state.CurrentIndex].Name
	} else {
		songName = state.CurrentFile
	}

	// Status icon
	var icon string
	if state.IsPaused {
		icon = "⏸"
	} else if state.IsPlaying {
		icon = "▶"
	} else {
		icon = "♪"
	}

	// Format time
	posStr := FormatDuration(state.Position)
	durStr := FormatDuration(state.Duration)

	// Progress bar (simple)
	var progressBar string
	if state.Duration > 0 {
		pct := float64(state.Position) / float64(state.Duration)
		barWidth := 20
		filled := int(pct * float64(barWidth))
		progressBar = strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	}

	playing := fmt.Sprintf("%s %s  %s  %s/%s  [%d/%d]",
		icon,
		nowPlayingStyle.Render(songName),
		progressBar,
		posStr,
		durStr,
		state.CurrentIndex+1,
		state.TotalTracks,
	)

	return boxStyle.Render(playing)
}

// renderSearchView renders the search input view.
func (m Model) renderSearchView() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render(" 🔍 YouTube Search ") + "\n\n")
	b.WriteString(m.searchInput.View() + "\n")

	if m.isSearching {
		b.WriteString(m.downloadSpinner.View() + " Searching YouTube...\n")
	}

	if m.searchError != "" {
		b.WriteString(mutedStyle.Render("⚠ " + m.searchError) + "\n")
	}

	return b.String()
}

// renderLibraryView renders the local music library.
func (m Model) renderLibraryView() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render(" 📚 Library ") + "\n\n")

	if len(m.libraryFiles) == 0 {
		b.WriteString(mutedStyle.Render("No music files found in ./Music\n"))
		b.WriteString(mutedStyle.Render("Press 's' to search and download music\n"))
		return b.String()
	}

	// Calculate visible range for scrolling
	maxVisible := m.height - 15 // Leave room for other UI elements
	if maxVisible < 5 {
		maxVisible = 5
	}

	start := 0
	if m.libraryCursor >= maxVisible {
		start = m.libraryCursor - maxVisible + 1
	}

	end := start + maxVisible
	if end > len(m.libraryFiles) {
		end = len(m.libraryFiles)
	}

	// Get current playing index
	state := m.player.GetState()

	for i := start; i < end; i++ {
		file := m.libraryFiles[i]
		var line string

		// Playing indicator
		var prefix string
		if i == state.CurrentIndex && state.IsPlaying {
			if state.IsPaused {
				prefix = "⏸ "
			} else {
				prefix = "▶ "
			}
		} else {
			prefix = "  "
		}

		if i == m.libraryCursor {
			line = selectedStyle.Render(fmt.Sprintf("%s> %s", prefix, file.Name))
		} else {
			line = normalStyle.Render(fmt.Sprintf("%s  %s", prefix, file.Name))
		}

		b.WriteString(line + "\n")
	}

	// Scroll indicator
	if len(m.libraryFiles) > maxVisible {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("\n(%d/%d)", m.libraryCursor+1, len(m.libraryFiles))))
	}

	return b.String()
}

// renderResultsView renders the YouTube search results.
func (m Model) renderResultsView() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render(fmt.Sprintf(" 🎬 Results for '%s' ", m.searchQuery)) + "\n\n")

	if len(m.youtubeResults) == 0 {
		b.WriteString(mutedStyle.Render("No results\n"))
		return b.String()
	}

	// Calculate visible range
	maxVisible := m.height - 15
	if maxVisible < 5 {
		maxVisible = 5
	}

	start := 0
	if m.resultsCursor >= maxVisible {
		start = m.resultsCursor - maxVisible + 1
	}

	end := start + maxVisible
	if end > len(m.youtubeResults) {
		end = len(m.youtubeResults)
	}

	for i := start; i < end; i++ {
		result := m.youtubeResults[i]
		info := fmt.Sprintf("[%s] %s", result.Duration, result.Channel)

		var line string
		if i == m.resultsCursor {
			line = selectedStyle.Render("> " + result.Title)
			line += "\n  " + mutedStyle.Render(info)
		} else {
			line = normalStyle.Render("  " + result.Title)
			line += "\n  " + mutedStyle.Render(info)
		}

		b.WriteString(line + "\n")
	}

	return b.String()
}

// renderDownloadProgress renders the download progress bar.
func (m Model) renderDownloadProgress() string {
	dp := m.downloader.GetProgress()
	
	var b strings.Builder
	b.WriteString("\n" + m.downloadSpinner.View())
	b.WriteString(fmt.Sprintf(" %s\n", dp.Status))
	b.WriteString(m.downloadProgress.ViewAs(dp.Progress / 100))
	
	return b.String()
}

// renderHelp renders the help bar.
func (m Model) renderHelp() string {
	var keys []string

	switch m.currentView {
	case ViewSearch:
		keys = []string{"enter: search", "esc: cancel", "tab: library"}
	case ViewLibrary:
		keys = []string{"↑/↓: navigate", "enter: play", "s: search", "space: pause"}
	case ViewResults:
		keys = []string{"↑/↓: navigate", "enter: download", "tab: library", "esc: back"}
	}

	// Add playback controls
	keys = append(keys, "←/→: prev/next", "q: quit")

	return helpStyle.Render(strings.Join(keys, " • "))
}

// Command functions

// tickCmd returns a command that sends a tick message periodically.
func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// performYouTubeSearch returns a command that performs a YouTube search.
func (m Model) performYouTubeSearch(query string) tea.Cmd {
	return func() tea.Msg {
		results, err := SearchYouTube(query)
		return youtubeSearchCompleteMsg{results: results, err: err}
	}
}

// refreshLibrary returns a command that refreshes the music library.
func (m Model) refreshLibrary() tea.Cmd {
	return func() tea.Msg {
		files, _ := ScanMusicFiles()
		return libraryRefreshMsg(files)
	}
}
