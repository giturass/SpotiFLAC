package gobackend

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Spotify image size codes (same as PC version)
const (
	spotifySize640 = "ab67616d0000b273" // 640x640
	spotifySizeMax = "ab67616d000082c1" // Max resolution (~2000x2000)
)

// downloadCoverToMemory downloads cover art and returns as bytes (no file creation)
// This avoids file permission issues on Android
func downloadCoverToMemory(coverURL string, maxQuality bool) ([]byte, error) {
	if coverURL == "" {
		return nil, fmt.Errorf("no cover URL provided")
	}

	fmt.Printf("[Cover] Downloading cover from: %s\n", coverURL)

	// Upgrade to max quality if requested
	downloadURL := coverURL
	if maxQuality {
		downloadURL = upgradeToMaxQuality(coverURL)
		if downloadURL != coverURL {
			fmt.Printf("[Cover] Upgraded to max quality URL: %s\n", downloadURL)
		}
	}

	client := NewHTTPClientWithTimeout(DefaultTimeout)

	// Create request with User-Agent (required by Spotify CDN)
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := DoRequestWithUserAgent(client, req)
	if err != nil {
		return nil, fmt.Errorf("failed to download cover: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("cover download failed: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read cover data: %w", err)
	}

	fmt.Printf("[Cover] Downloaded %d bytes\n", len(data))
	return data, nil
}

// upgradeToMaxQuality upgrades Spotify cover URL to maximum quality
// Uses same logic as PC version - replaces 640x640 size code with max resolution
func upgradeToMaxQuality(coverURL string) string {
	// Spotify image URLs can be upgraded by changing the size parameter
	// Format: https://i.scdn.co/image/ab67616d0000b273...
	// ab67616d0000b273 = 640x640
	// ab67616d000082c1 = Max resolution (~2000x2000)

	if strings.Contains(coverURL, spotifySize640) {
		// Try max resolution first
		maxURL := strings.Replace(coverURL, spotifySize640, spotifySizeMax, 1)

		// Verify max resolution URL is available
		client := NewHTTPClientWithTimeout(DefaultTimeout)
		req, err := http.NewRequest("HEAD", maxURL, nil)
		if err == nil {
			resp, err := DoRequestWithUserAgent(client, req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return maxURL
				}
			}
		}
	}

	return coverURL
}

// GetCoverFromSpotify gets cover URL from Spotify metadata
func GetCoverFromSpotify(imageURL string, maxQuality bool) string {
	if imageURL == "" {
		return ""
	}

	if maxQuality {
		return upgradeToMaxQuality(imageURL)
	}

	return imageURL
}
