// Package main provides the filesystem management for Personal Musician.
// This module handles the local ./Music folder and tracks downloaded MP3 files.
package main

import (
	"os"
	"path/filepath"
	"strings"
)

// MusicDir is the default directory where downloaded MP3 files are stored.
const MusicDir = "./Music"

// MusicFile represents a local MP3 file with its metadata.
type MusicFile struct {
	Name     string // Display name (filename without extension)
	Path     string // Full path to the file
	FileName string // Filename with extension
}

// InitMusicDir creates the Music directory if it doesn't exist.
// Returns an error if the directory cannot be created.
func InitMusicDir() error {
	// Create the directory with read/write/execute permissions for user
	return os.MkdirAll(MusicDir, 0755)
}

// ScanMusicFiles scans the Music directory and returns all MP3 files.
// Returns an empty slice if no files are found or if the directory doesn't exist.
func ScanMusicFiles() ([]MusicFile, error) {
	var files []MusicFile

	// Check if directory exists
	if _, err := os.Stat(MusicDir); os.IsNotExist(err) {
		return files, nil // Return empty slice, not an error
	}

	// Walk through the Music directory
	err := filepath.Walk(MusicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-MP3 files
		if info.IsDir() {
			return nil
		}

		// Check if file is an MP3 (case-insensitive)
		if strings.ToLower(filepath.Ext(path)) == ".mp3" {
			fileName := filepath.Base(path)
			name := strings.TrimSuffix(fileName, filepath.Ext(fileName))

			files = append(files, MusicFile{
				Name:     name,
				Path:     path,
				FileName: fileName,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// FileExists checks if a file with a similar name already exists in the Music directory.
// Uses case-insensitive comparison and ignores file extensions.
func FileExists(name string) bool {
	files, err := ScanMusicFiles()
	if err != nil {
		return false
	}

	// Normalize the search name (lowercase, no extension)
	searchName := strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name)))

	for _, file := range files {
		// Compare normalized names
		existingName := strings.ToLower(file.Name)
		if existingName == searchName || strings.Contains(existingName, searchName) {
			return true
		}
	}

	return false
}

// GetFilePath returns the full path to a music file by name.
// Returns empty string if the file is not found.
func GetFilePath(name string) string {
	files, err := ScanMusicFiles()
	if err != nil {
		return ""
	}

	searchName := strings.ToLower(name)

	for _, file := range files {
		if strings.ToLower(file.Name) == searchName {
			return file.Path
		}
	}

	return ""
}

// GetMusicDirAbsPath returns the absolute path to the Music directory.
func GetMusicDirAbsPath() (string, error) {
	return filepath.Abs(MusicDir)
}
