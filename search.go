// Package main provides YouTube search functionality for Personal Musician.
// This module searches YouTube for music and returns video URLs.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// SearchResult represents a single YouTube search result.
type SearchResult struct {
	VideoID   string // YouTube video ID
	Title     string // Video title
	Channel   string // Channel name
	Duration  string // Video duration
	Thumbnail string // Thumbnail URL
}

// SearchYouTube searches YouTube for videos matching the query.
// Returns a slice of SearchResult with video information.
func SearchYouTube(query string) ([]SearchResult, error) {
	// Use YouTube's search page and parse results
	searchURL := fmt.Sprintf("https://www.youtube.com/results?search_query=%s",
		url.QueryEscape(query+" audio"))

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	// Create request with browser-like headers
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse the response to extract video data
	return parseYouTubeResults(string(body))
}

// parseYouTubeResults extracts video information from YouTube HTML response.
func parseYouTubeResults(html string) ([]SearchResult, error) {
	var results []SearchResult

	// Find the ytInitialData JSON in the HTML
	re := regexp.MustCompile(`var ytInitialData = ({.*?});`)
	matches := re.FindStringSubmatch(html)
	
	if len(matches) < 2 {
		// Try alternative pattern
		re = regexp.MustCompile(`ytInitialData\s*=\s*({.*?});`)
		matches = re.FindStringSubmatch(html)
	}

	if len(matches) < 2 {
		return nil, fmt.Errorf("could not find video data in response")
	}

	// Parse the JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(matches[1]), &data); err != nil {
		return nil, fmt.Errorf("failed to parse video data: %w", err)
	}

	// Navigate the nested JSON structure to find video renderers
	// YouTube's JSON structure is deeply nested
	contents := navigateJSON(data,
		"contents",
		"twoColumnSearchResultsRenderer",
		"primaryContents",
		"sectionListRenderer",
		"contents",
	)

	if contents == nil {
		return results, nil
	}

	contentsList, ok := contents.([]interface{})
	if !ok {
		return results, nil
	}

	for _, section := range contentsList {
		sectionMap, ok := section.(map[string]interface{})
		if !ok {
			continue
		}

		itemRenderer := sectionMap["itemSectionRenderer"]
		if itemRenderer == nil {
			continue
		}

		itemRendererMap, ok := itemRenderer.(map[string]interface{})
		if !ok {
			continue
		}

		items, ok := itemRendererMap["contents"].([]interface{})
		if !ok {
			continue
		}

		for _, item := range items {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			videoRenderer, ok := itemMap["videoRenderer"].(map[string]interface{})
			if !ok {
				continue
			}

			result := extractVideoInfo(videoRenderer)
			if result.VideoID != "" {
				results = append(results, result)
				if len(results) >= 10 { // Limit to 10 results
					return results, nil
				}
			}
		}
	}

	return results, nil
}

// extractVideoInfo extracts video information from a videoRenderer object.
func extractVideoInfo(renderer map[string]interface{}) SearchResult {
	result := SearchResult{}

	// Get video ID
	if videoID, ok := renderer["videoId"].(string); ok {
		result.VideoID = videoID
	}

	// Get title
	if title, ok := renderer["title"].(map[string]interface{}); ok {
		if runs, ok := title["runs"].([]interface{}); ok && len(runs) > 0 {
			if run, ok := runs[0].(map[string]interface{}); ok {
				if text, ok := run["text"].(string); ok {
					result.Title = text
				}
			}
		}
	}

	// Get channel name
	if channel, ok := renderer["ownerText"].(map[string]interface{}); ok {
		if runs, ok := channel["runs"].([]interface{}); ok && len(runs) > 0 {
			if run, ok := runs[0].(map[string]interface{}); ok {
				if text, ok := run["text"].(string); ok {
					result.Channel = text
				}
			}
		}
	}

	// Get duration
	if lengthText, ok := renderer["lengthText"].(map[string]interface{}); ok {
		if simpleText, ok := lengthText["simpleText"].(string); ok {
			result.Duration = simpleText
		}
	}

	return result
}

// navigateJSON navigates through nested JSON using a sequence of keys.
func navigateJSON(data interface{}, keys ...string) interface{} {
	current := data
	for _, key := range keys {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[key]
		} else {
			return nil
		}
	}
	return current
}

// GetYouTubeURL returns the full YouTube URL for a video ID.
func GetYouTubeURL(videoID string) string {
	return fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
}

// FormatSearchResult returns a formatted string representation of a SearchResult.
func FormatSearchResult(r SearchResult) string {
	return fmt.Sprintf("%s [%s] - %s", r.Title, r.Duration, r.Channel)
}

// Legacy torrent functions kept for compatibility but deprecated
// TorrentResult represents a single search result (deprecated - kept for compatibility).
type TorrentResult struct {
	ID       string
	Name     string
	InfoHash string
	Seeders  int
	Leechers int
	Size     int64
	Category string
}

// SearchTorrents is deprecated - use SearchYouTube instead.
func SearchTorrents(query string) ([]TorrentResult, error) {
	return nil, fmt.Errorf("torrent search deprecated, use YouTube search")
}

// GetMagnetLink is deprecated.
func GetMagnetLink(result TorrentResult) string {
	return ""
}

// FormatSize converts bytes to human-readable format.
func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatResult formats a TorrentResult (deprecated).
func FormatResult(r TorrentResult) string {
	return strings.TrimSpace(r.Name)
}
