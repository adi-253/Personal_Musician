# 🎵 Personal Musician

A minimal, beautiful CLI music player that searches and downloads music from YouTube directly to your terminal. Built with Go and featuring a sleek TUI powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea).

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![Platform](https://img.shields.io/badge/Platform-Linux%20|%20macOS-lightgrey)

##  Features

- **YouTube Search** — Search millions of songs directly from your terminal
- **One-Click Download** — Download audio as MP3 using yt-dlp
- **Built-in Player** — Play music without leaving the terminal
- **Local Library** — Manage your downloaded music collection
- **Beautiful TUI** — Modern terminal UI with colors, progress bars, and smooth navigation
- **Keyboard Driven** — Full keyboard navigation for a seamless experience


## 🚀 Installation

### Prerequisites

- **Go 1.21+** 
- **yt-dlp** — Required for downloading from YouTube
- **ffmpeg** — Required for audio conversion

#### Install yt-dlp

```bash
# Using pip
pip install yt-dlp

# Or using Homebrew (macOS)
brew install yt-dlp

# Or using your package manager (Linux)
sudo apt install yt-dlp  # Debian/Ubuntu
sudo pacman -S yt-dlp    # Arch Linux
```

#### Install ffmpeg

```bash
# macOS
brew install ffmpeg

# Debian/Ubuntu
sudo apt install ffmpeg

# Arch Linux
sudo pacman -S ffmpeg
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/adi-253/Personal_Musician.git
cd Personal_Musician

# Build the application
go build -o personal-musician .

# Run it
./personal-musician
```

### Install with Go

```bash
go install github.com/adi-253/Personal_Musician@latest
```

## Usage

Simply run the application:

```bash
./personal-musician
```

### Keyboard Controls

| Key | Action |
|-----|--------|
| `Space` | Pause/Resume playback |
| `←` / `→` | Previous/Next song |
| `↑` / `↓` | Navigate lists |
| `Enter` | Select/Confirm |
| `s` | Open  search |
| `Tab` | Switch between Library and Results |
| `Esc` | Back to library |
| `q` / `Ctrl+C` | Quit |

## Project Structure

```
Personal_Musician/
├── main.go          # Application entry point
├── tui.go           # Terminal UI (Bubble Tea)
├── player.go        # Audio playback (beep)
├── search.go        # YouTube search
├── downloader.go    # YouTube download (yt-dlp)
├── filesystem.go    # Local file management
├── Music/           # Downloaded songs directory
└── go.mod           # Go module definition
```

## Tech Stack

| Component | Library |
|-----------|---------|
| **TUI Framework** | [Bubble Tea](https://github.com/charmbracelet/bubbletea) |
| **TUI Styling** | [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| **Audio Playback** | [beep](https://github.com/gopxl/beep) |
| **YouTube Download** | [yt-dlp](https://github.com/yt-dlp/yt-dlp) (external) |

## How It Works

1. **Search** — Enter a song name and search YouTube
2. **Download** — Select a result to download as MP3
3. **Play** — Songs are saved to `./Music/` and auto-added to your library
4. **Enjoy** — Navigate your library and control playback with keyboard shortcuts


