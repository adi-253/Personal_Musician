// Package main is the entry point for Personal Musician.
// Personal Musician is a CLI-based music player that fetches and plays
// music via the BitTorrent protocol, targeting MP3 files for terminal listening.
//
// Features:
// - Search for music torrents via TPB API
// - Download MP3 files via BitTorrent
// - Play music with pause/resume and navigation controls
// - Beautiful terminal UI using Bubble Tea
//
// Usage:
//
//	personal-musician
//
// Controls:
//
//	Space     - Pause/Resume playback
//	←/→       - Previous/Next song
//	↑/↓       - Navigate lists
//	Enter     - Select/Confirm
//	s         - Open search
//	Tab       - Switch views
//	Esc       - Back to library
//	q/Ctrl+C  - Quit
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Print welcome banner
	fmt.Println("🎵 Personal Musician - Starting...")

	// Initialize the Music directory
	if err := InitMusicDir(); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Music directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize the downloader
	downloader, err := NewDownloader(MusicDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing downloader: %v\n", err)
		os.Exit(1)
	}
	defer downloader.Close()

	// Initialize the player
	player := NewPlayer()
	defer player.Close()

	// Scan existing music files and set as playlist
	files, err := ScanMusicFiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not scan music files: %v\n", err)
	}
	player.SetPlaylist(files)

	// Create the TUI model
	model := NewModel(player, downloader)

	// Create and run the Bubble Tea program
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the program
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("👋 Goodbye!")
}
