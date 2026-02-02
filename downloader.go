// Package main provides the YouTube download functionality for Personal Musician.
// This module uses yt-dlp to download audio from YouTube videos.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Downloader manages YouTube downloads using yt-dlp.
type Downloader struct {
	musicDir string
	mu       sync.Mutex

	// Current download state
	downloadedFiles []string
	progress        float64
	status          string
	isDownloading   bool
	cancelFunc      context.CancelFunc
	cmd             *exec.Cmd
}

// DownloadProgress holds the current download progress information.
type DownloadProgress struct {
	Progress      float64  // Percentage 0-100
	Status        string   // Current status message
	IsDownloading bool     // Whether a download is in progress
	Files         []string // List of downloaded file paths
}

// NewDownloader creates a new Downloader instance.
// The musicDir is where downloaded audio files will be saved.
func NewDownloader(musicDir string) (*Downloader, error) {
	// Get absolute path for the music directory
	absPath, err := filepath.Abs(musicDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve music directory: %w", err)
	}

	// Ensure the directory exists
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create music directory: %w", err)
	}

	// Check if yt-dlp is available
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		return nil, fmt.Errorf("yt-dlp not found. Please install it: pip install yt-dlp")
	}

	return &Downloader{
		musicDir: absPath,
		status:   "Idle",
	}, nil
}

// Close shuts down the downloader gracefully.
func (d *Downloader) Close() error {
	d.CancelDownload()
	return nil
}

// DownloadFromYouTube starts downloading audio from a YouTube video.
// This method is non-blocking and downloads in the background.
// Use GetProgress() to monitor the download status.
func (d *Downloader) DownloadFromYouTube(ctx context.Context, videoID string, title string) error {
	d.mu.Lock()
	if d.isDownloading {
		d.mu.Unlock()
		return fmt.Errorf("a download is already in progress")
	}
	d.isDownloading = true
	d.progress = 0
	d.status = "Starting download..."
	d.downloadedFiles = nil

	// Create cancellable context
	downloadCtx, cancel := context.WithCancel(ctx)
	d.cancelFunc = cancel
	d.mu.Unlock()

	// Start the download in a goroutine
	go d.downloadVideo(downloadCtx, videoID, title)

	return nil
}

// downloadVideo handles the actual download process using yt-dlp.
func (d *Downloader) downloadVideo(ctx context.Context, videoID string, title string) {
	defer func() {
		d.mu.Lock()
		d.isDownloading = false
		d.cancelFunc = nil
		d.cmd = nil
		d.mu.Unlock()
	}()

	// Create safe filename
	safeTitle := sanitizeFilename(title)
	if safeTitle == "" {
		safeTitle = videoID
	}

	outputPath := filepath.Join(d.musicDir, safeTitle+".%(ext)s")
	videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)

	d.setStatus("Downloading with yt-dlp...", true)

	// Use yt-dlp to download audio and convert to mp3
	cmd := exec.CommandContext(ctx, "yt-dlp",
		"-x",                    // Extract audio
		"--audio-format", "mp3", // Convert to MP3
		"--audio-quality", "0",  // Best quality
		"-o", outputPath,        // Output path template
		"--no-playlist",         // Don't download playlists
		"--quiet",               // Less output
		"--progress",            // Show progress
		videoURL,
	)

	d.mu.Lock()
	d.cmd = cmd
	d.mu.Unlock()

	// Capture output for progress
	output, err := cmd.CombinedOutput()
	
	if ctx.Err() != nil {
		d.setStatus("Download cancelled", false)
		return
	}

	if err != nil {
		d.setStatus(fmt.Sprintf("Download failed: %v", err), false)
		// Log the output for debugging
		if len(output) > 0 {
			fmt.Printf("yt-dlp output: %s\n", string(output))
		}
		return
	}

	// Find the downloaded file
	mp3Path := filepath.Join(d.musicDir, safeTitle+".mp3")
	
	// Check if file exists
	if _, err := os.Stat(mp3Path); os.IsNotExist(err) {
		// Try to find any file that matches the pattern
		matches, _ := filepath.Glob(filepath.Join(d.musicDir, safeTitle+".*"))
		if len(matches) > 0 {
			mp3Path = matches[0]
		} else {
			d.setStatus("Download completed but file not found", false)
			return
		}
	}

	// Success!
	d.mu.Lock()
	d.downloadedFiles = []string{mp3Path}
	d.progress = 100
	d.status = fmt.Sprintf("Downloaded: %s", filepath.Base(mp3Path))
	d.isDownloading = false
	d.mu.Unlock()
}

// sanitizeFilename removes invalid characters from a filename.
func sanitizeFilename(name string) string {
	// Remove or replace invalid characters
	re := regexp.MustCompile(`[<>:"/\\|?*]`)
	safe := re.ReplaceAllString(name, "")
	
	// Trim spaces and dots
	safe = strings.TrimSpace(safe)
	safe = strings.Trim(safe, ".")
	
	// Limit length
	if len(safe) > 100 {
		safe = safe[:100]
	}
	
	return safe
}

// setStatus updates the download status in a thread-safe manner.
func (d *Downloader) setStatus(status string, downloading bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.status = status
	d.isDownloading = downloading
}

// GetProgress returns the current download progress.
func (d *Downloader) GetProgress() DownloadProgress {
	d.mu.Lock()
	defer d.mu.Unlock()
	return DownloadProgress{
		Progress:      d.progress,
		Status:        d.status,
		IsDownloading: d.isDownloading,
		Files:         d.downloadedFiles,
	}
}

// CancelDownload cancels the current download if one is in progress.
func (d *Downloader) CancelDownload() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cancelFunc != nil {
		d.cancelFunc()
		d.cancelFunc = nil
	}
	if d.cmd != nil && d.cmd.Process != nil {
		d.cmd.Process.Kill()
	}
	d.isDownloading = false
	d.status = "Download cancelled"
}

// IsDownloading returns whether a download is currently in progress.
func (d *Downloader) IsDownloading() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.isDownloading
}
