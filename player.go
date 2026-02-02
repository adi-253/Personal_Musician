// Package main provides the audio playback functionality for Personal Musician.
// This module uses gopxl/beep for decoding and playing MP3 files.
package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
)

// Player manages audio playback state and controls.
type Player struct {
	mu sync.Mutex

	// Audio stream components
	streamer   beep.StreamSeekCloser
	ctrl       *beep.Ctrl
	sampleRate beep.SampleRate
	format     beep.Format

	// Playback state
	currentFile    string
	isPlaying      bool
	isPaused       bool
	speakerInit    bool
	position       time.Duration
	duration       time.Duration

	// Playlist management
	playlist      []MusicFile
	currentIndex  int
	onSongChange  func() // Callback when song changes
}

// PlaybackState holds current playback information.
type PlaybackState struct {
	CurrentFile  string
	IsPlaying    bool
	IsPaused     bool
	Position     time.Duration
	Duration     time.Duration
	CurrentIndex int
	TotalTracks  int
}

// NewPlayer creates a new Player instance.
func NewPlayer() *Player {
	return &Player{
		currentIndex: -1,
	}
}

// SetPlaylist sets the current playlist of songs.
func (p *Player) SetPlaylist(files []MusicFile) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.playlist = files
	if len(files) > 0 && p.currentIndex < 0 {
		p.currentIndex = 0
	}
}

// SetOnSongChange sets a callback function to be called when the song changes.
func (p *Player) SetOnSongChange(callback func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onSongChange = callback
}

// GetPlaylist returns the current playlist.
func (p *Player) GetPlaylist() []MusicFile {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playlist
}

// PlayFile loads and plays an MP3 file.
func (p *Player) PlayFile(filePath string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Close any existing stream
	p.stopInternal()

	// Open the MP3 file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	// Decode the MP3 file
	streamer, format, err := mp3.Decode(file)
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to decode MP3: %w", err)
	}

	// Initialize speaker if not already done (only once per app lifetime)
	if !p.speakerInit {
		if err := speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10)); err != nil {
			streamer.Close()
			return fmt.Errorf("failed to initialize speaker: %w", err)
		}
		p.speakerInit = true
		p.sampleRate = format.SampleRate
	}

	// Resample if sample rates differ
	var resampled beep.Streamer = streamer
	if format.SampleRate != p.sampleRate {
		resampled = beep.Resample(4, format.SampleRate, p.sampleRate, streamer)
	}

	// Create control wrapper for pause/resume functionality
	p.ctrl = &beep.Ctrl{Streamer: resampled, Paused: false}

	// Store state
	p.streamer = streamer
	p.format = format
	p.currentFile = filePath
	p.isPlaying = true
	p.isPaused = false

	// Calculate duration
	p.duration = format.SampleRate.D(streamer.Len())

	// Play the audio
	speaker.Play(beep.Seq(p.ctrl, beep.Callback(func() {
		// Called when playback finishes
		p.mu.Lock()
		p.isPlaying = false
		p.isPaused = false
		callback := p.onSongChange
		p.mu.Unlock()
		
		// Auto-advance to next song
		go func() {
			p.NextSong()
			if callback != nil {
				callback()
			}
		}()
	})))

	return nil
}

// PlayIndex plays a song from the playlist by index.
func (p *Player) PlayIndex(index int) error {
	p.mu.Lock()
	if index < 0 || index >= len(p.playlist) {
		p.mu.Unlock()
		return fmt.Errorf("index out of range")
	}
	p.currentIndex = index
	filePath := p.playlist[index].Path
	p.mu.Unlock()

	return p.PlayFile(filePath)
}

// TogglePause toggles between pause and resume states.
func (p *Player) TogglePause() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ctrl == nil || !p.isPlaying {
		return
	}

	speaker.Lock()
	p.ctrl.Paused = !p.ctrl.Paused
	p.isPaused = p.ctrl.Paused
	speaker.Unlock()
}

// Stop stops the current playback.
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopInternal()
}

// stopInternal stops playback without locking (internal use).
func (p *Player) stopInternal() {
	if p.streamer != nil {
		speaker.Clear()
		p.streamer.Close()
		p.streamer = nil
		p.ctrl = nil
	}
	p.isPlaying = false
	p.isPaused = false
}

// NextSong advances to the next song in the playlist.
func (p *Player) NextSong() error {
	p.mu.Lock()
	if len(p.playlist) == 0 {
		p.mu.Unlock()
		return fmt.Errorf("playlist is empty")
	}

	// Move to next song (wrap around)
	nextIndex := (p.currentIndex + 1) % len(p.playlist)
	p.mu.Unlock()

	return p.PlayIndex(nextIndex)
}

// PrevSong goes back to the previous song in the playlist.
func (p *Player) PrevSong() error {
	p.mu.Lock()
	if len(p.playlist) == 0 {
		p.mu.Unlock()
		return fmt.Errorf("playlist is empty")
	}

	// Move to previous song (wrap around)
	prevIndex := p.currentIndex - 1
	if prevIndex < 0 {
		prevIndex = len(p.playlist) - 1
	}
	p.mu.Unlock()

	return p.PlayIndex(prevIndex)
}

// GetState returns the current playback state.
func (p *Player) GetState() PlaybackState {
	p.mu.Lock()
	defer p.mu.Unlock()

	state := PlaybackState{
		CurrentFile:  p.currentFile,
		IsPlaying:    p.isPlaying,
		IsPaused:     p.isPaused,
		Duration:     p.duration,
		CurrentIndex: p.currentIndex,
		TotalTracks:  len(p.playlist),
	}

	// Get current position if playing
	if p.streamer != nil && p.format.SampleRate > 0 {
		speaker.Lock()
		pos := p.streamer.Position()
		speaker.Unlock()
		state.Position = p.format.SampleRate.D(pos)
	}

	return state
}

// GetDuration returns the duration of the current track.
func (p *Player) GetDuration() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.duration
}

// GetPosition returns the current playback position.
func (p *Player) GetPosition() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.streamer == nil || p.format.SampleRate == 0 {
		return 0
	}

	speaker.Lock()
	pos := p.streamer.Position()
	speaker.Unlock()

	return p.format.SampleRate.D(pos)
}

// Close releases all resources held by the player.
func (p *Player) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopInternal()
}

// FormatDuration formats a duration as MM:SS.
func FormatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
