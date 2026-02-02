# 🎵 Personal Musician: Development Guidelines

**Personal Musician** is a minimalist CLI-based music player designed to fetch and play music via the BitTorrent protocol, specifically targeting **MP3 files** for optimized terminal-based listening.

---

## 🛠 Tech Stack
* **Language:** Golang (v1.21+)
* **Torrent Engine:** [anacrolix/torrent](https://github.com/anacrolix/torrent)
* **Search Engine:** [ismaelpadilla/gotorrent](https://github.com/ismaelpadilla/gotorrent)
* **Project Module:** `github.com/adi-253/Personal_Musician`

### 📚 Official Documentation Links
1. **Torrent Client:** [anacrolix/torrent](https://github.com/anacrolix/torrent) | [Pkg.go.dev](https://pkg.go.dev/github.com/anacrolix/torrent)
2. **Search Engine:** [ismaelpadilla/gotorrent](https://github.com/ismaelpadilla/gotorrent)

---

## 📂 Project Structure
* `main.go`: Entry point for application initialization and global state.
* `search.go`: (New) Uses logic from `gotorrent` to fetch magnet links based on user input.
* `downloader.go`: Manages magnet links and filters for **MP3 files** using the `anacrolix` client.
* `player.go`: Handles audio decoding, playback state (Pause/Resume), and navigation.
* `filesystem.go`: Scans the `./Music` folder and manages the local file indexing.

---

## 🔄 Application Workflow & Navigation
1.  **Bootstrapping:** Creates the `./Music` folder and indexes existing MP3s.
2.  **Search:** * User enters a song name in the CLI.
    * The app uses the `gotorrent` search logic to find the best matching magnet link.
3.  **Local Check:** Skips download if a file with a similar name already exists in `./Music/`.
4.  **Targeted Download:**
    * Filters the torrent for `.mp3` files only.
    * Downloads the file directly into the `./Music` folder.
5.  **Playback & Control:**
    * **Pause/Play:** Toggle the audio stream.
    * **Forward/Backward:** Navigate through the local file list.

---

## ⚖️ Development Standards
* **Error Handling:** Gracefully handle "No results found" from the search engine.
* **Concurrency:** Ensure search and download happen on background threads to keep the TUI responsive.