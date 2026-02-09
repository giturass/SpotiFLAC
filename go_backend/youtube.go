// Package gobackend provides YouTube download functionality via Cobalt API
// YouTube is a lossy-only provider (not part of lossless fallback chain)
package gobackend

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type YouTubeDownloader struct {
	client *http.Client
	apiURL string
	mu     sync.Mutex
}

var (
	globalYouTubeDownloader *YouTubeDownloader
	youtubeDownloaderOnce   sync.Once
)

type YouTubeQuality string

const (
	YouTubeQualityOpus256 YouTubeQuality = "opus_256"
	YouTubeQualityMP3320  YouTubeQuality = "mp3_320"
)

type CobaltRequest struct {
	URL             string `json:"url"`
	AudioBitrate    string `json:"audioBitrate,omitempty"`
	AudioFormat     string `json:"audioFormat,omitempty"`
	DownloadMode    string `json:"downloadMode,omitempty"`
	FilenameStyle   string `json:"filenameStyle,omitempty"`
	DisableMetadata bool   `json:"disableMetadata,omitempty"`
}

type CobaltResponse struct {
	Status   string `json:"status"`
	URL      string `json:"url,omitempty"`
	Filename string `json:"filename,omitempty"`
	Error    *struct {
		Code    string `json:"code"`
		Context *struct {
			Service string `json:"service,omitempty"`
			Limit   int    `json:"limit,omitempty"`
		} `json:"context,omitempty"`
	} `json:"error,omitempty"`
}

type YouTubeDownloadResult struct {
	FilePath    string
	Title       string
	Artist      string
	Album       string
	ReleaseDate string
	TrackNumber int
	DiscNumber  int
	ISRC        string
	Format      string // "opus" or "mp3"
	Bitrate     int
	LyricsLRC   string
	CoverData   []byte
}

// NewYouTubeDownloader creates or returns the singleton YouTube downloader
	youtubeDownloaderOnce.Do(func() {
		globalYouTubeDownloader = &YouTubeDownloader{
			client: NewHTTPClientWithTimeout(120 * time.Second),
			apiURL: "https://api.qwkuns.me", // Cobalt-based API
		}
	})
	return globalYouTubeDownloader
}

// SearchYouTube searches for a track on YouTube and returns the best matching video URL
func (y *YouTubeDownloader) SearchYouTube(trackName, artistName string) (string, error) {
	// Build search query
	query := fmt.Sprintf("%s %s", artistName, trackName)

	// Use YouTube's search to find the video
	// We'll use a simple approach: construct a YouTube Music search URL pattern
	// The actual video ID will be resolved by Cobalt
	searchQuery := url.QueryEscape(query)

	GoLog("[YouTube] Search query: %s\n", query)

	// For now, we'll need to use YouTube Music's /watch endpoint with search
	// A better approach is to use YouTube Data API, but Cobalt can handle music.youtube.com
	youtubeMusicURL := fmt.Sprintf("https://music.youtube.com/search?q=%s", searchQuery)

	return youtubeMusicURL, nil
}

// GetDownloadURL gets the direct download URL from Cobalt API
func (y *YouTubeDownloader) GetDownloadURL(youtubeURL string, quality YouTubeQuality) (*CobaltResponse, error) {
	y.mu.Lock()
	defer y.mu.Unlock()

	var audioFormat string
	var audioBitrate string

	switch quality {
	case YouTubeQualityOpus256:
		audioFormat = "opus"
		audioBitrate = "256"
	case YouTubeQualityMP3320:
		audioFormat = "mp3"
		audioBitrate = "320"
	default:
		audioFormat = "mp3"
		audioBitrate = "320"
	}

	reqBody := CobaltRequest{
		URL:             youtubeURL,
		AudioFormat:     audioFormat,
		AudioBitrate:    audioBitrate,
		DownloadMode:    "audio",
		FilenameStyle:   "basic",
		DisableMetadata: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	GoLog("[YouTube] Requesting from Cobalt API: %s (format: %s, bitrate: %s)\n",
		youtubeURL, audioFormat, audioBitrate)

	req, err := http.NewRequest("POST", y.apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := DoRequestWithUserAgent(y.client, req)
	if err != nil {
		return nil, fmt.Errorf("cobalt API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	GoLog("[YouTube] Cobalt API response status: %d\n", resp.StatusCode)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("cobalt API returned status %d: %s", resp.StatusCode, string(body))
	}

	var cobaltResp CobaltResponse
	if err := json.Unmarshal(body, &cobaltResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if cobaltResp.Status == "error" && cobaltResp.Error != nil {
		return nil, fmt.Errorf("cobalt error: %s", cobaltResp.Error.Code)
	}

	if cobaltResp.Status != "tunnel" && cobaltResp.Status != "redirect" {
		return nil, fmt.Errorf("unexpected cobalt status: %s", cobaltResp.Status)
	}

	if cobaltResp.URL == "" {
		return nil, fmt.Errorf("no download URL in response")
	}

	GoLog("[YouTube] Got download URL from Cobalt (status: %s)\n", cobaltResp.Status)

	return &cobaltResp, nil
}

// DownloadFile downloads the audio file from the given URL
func (y *YouTubeDownloader) DownloadFile(downloadURL, outputPath string, outputFD int, itemID string) error {
	ctx := context.Background()

	if itemID != "" {
		StartItemProgress(itemID)
		defer CompleteItemProgress(itemID)
		ctx = initDownloadCancel(itemID)
		defer clearDownloadCancel(itemID)
	}

	if isDownloadCancelled(itemID) {
		return ErrDownloadCancelled
	}

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := DoRequestWithUserAgent(y.client, req)
	if err != nil {
		if isDownloadCancelled(itemID) {
			return ErrDownloadCancelled
		}
		return fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	expectedSize := resp.ContentLength
	if expectedSize > 0 && itemID != "" {
		SetItemBytesTotal(itemID, expectedSize)
	}

	out, err := openOutputForWrite(outputPath, outputFD)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}

	bufWriter := bufio.NewWriterSize(out, 256*1024)

	var written int64
	if itemID != "" {
		progressWriter := NewItemProgressWriter(bufWriter, itemID)
		written, err = io.Copy(progressWriter, resp.Body)
	} else {
		written, err = io.Copy(bufWriter, resp.Body)
	}

	flushErr := bufWriter.Flush()
	closeErr := out.Close()

	if err != nil {
		cleanupOutputOnError(outputPath, outputFD)
		if isDownloadCancelled(itemID) {
			return ErrDownloadCancelled
		}
		return fmt.Errorf("download interrupted: %w", err)
	}
	if flushErr != nil {
		cleanupOutputOnError(outputPath, outputFD)
		return fmt.Errorf("failed to flush buffer: %w", flushErr)
	}
	if closeErr != nil {
		cleanupOutputOnError(outputPath, outputFD)
		return fmt.Errorf("failed to close file: %w", closeErr)
	}

	if expectedSize > 0 && written != expectedSize {
		cleanupOutputOnError(outputPath, outputFD)
		return fmt.Errorf("incomplete download: expected %d bytes, got %d bytes", expectedSize, written)
	}

	GoLog("[YouTube] Download completed: %d bytes written\n", written)

	return nil
}

// BuildYouTubeSearchURL constructs a YouTube Music search URL for a track
func BuildYouTubeSearchURL(trackName, artistName string) string {
	query := fmt.Sprintf("%s %s official audio", artistName, trackName)
	return fmt.Sprintf("https://music.youtube.com/search?q=%s", url.QueryEscape(query))
}

// BuildYouTubeWatchURL constructs a YouTube watch URL from a video ID
func BuildYouTubeWatchURL(videoID string) string {
	return fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
}

// isYouTubeVideoID checks if a string looks like a YouTube video ID
// YouTube video IDs are exactly 11 characters, containing alphanumeric, - and _
func isYouTubeVideoID(s string) bool {
	if len(s) != 11 {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// IsYouTubeURL checks if the given URL is a YouTube URL
func IsYouTubeURL(urlStr string) bool {
	lower := strings.ToLower(urlStr)
	return strings.Contains(lower, "youtube.com") ||
		strings.Contains(lower, "youtu.be") ||
		strings.Contains(lower, "music.youtube.com")
}

// ExtractYouTubeVideoID extracts the video ID from a YouTube URL
func ExtractYouTubeVideoID(urlStr string) (string, error) {
	// Handle youtu.be short URLs
	if strings.Contains(urlStr, "youtu.be/") {
		parts := strings.Split(urlStr, "youtu.be/")
		if len(parts) >= 2 {
			videoID := strings.Split(parts[1], "?")[0]
			videoID = strings.Split(videoID, "&")[0]
			return strings.TrimSpace(videoID), nil
		}
	}

	// Handle youtube.com URLs
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Check for /watch?v= format
	if v := parsed.Query().Get("v"); v != "" {
		return v, nil
	}

	// Check for /embed/ format
	if strings.Contains(parsed.Path, "/embed/") {
		parts := strings.Split(parsed.Path, "/embed/")
		if len(parts) >= 2 {
			return strings.Split(parts[1], "/")[0], nil
		}
	}

	// Check for /v/ format
	if strings.Contains(parsed.Path, "/v/") {
		parts := strings.Split(parsed.Path, "/v/")
		if len(parts) >= 2 {
			return strings.Split(parts[1], "/")[0], nil
		}
	}

	return "", fmt.Errorf("could not extract video ID from URL")
}

// downloadFromYouTube handles the complete download flow from YouTube
func downloadFromYouTube(req DownloadRequest) (YouTubeDownloadResult, error) {
	downloader := NewYouTubeDownloader()

	// Determine quality from request
	var quality YouTubeQuality
	switch strings.ToLower(req.Quality) {
	case "opus_256", "opus256", "opus":
		quality = YouTubeQualityOpus256
	case "mp3_320", "mp3320", "mp3":
		quality = YouTubeQualityMP3320
	default:
		quality = YouTubeQualityMP3320 // Default to MP3 320kbps
	}

	// Try to get YouTube URL
	// Priority: Direct YouTube Video ID -> Spotify ID -> Deezer ID -> ISRC
	var youtubeURL string
	var lookupErr error

	// Method 0: Check if SpotifyID is actually a YouTube video ID (from YT Music extension)
	// YouTube video IDs are 11 characters, alphanumeric with _ and -
	if req.SpotifyID != "" && isYouTubeVideoID(req.SpotifyID) {
		youtubeURL = BuildYouTubeWatchURL(req.SpotifyID)
		GoLog("[YouTube] SpotifyID appears to be YouTube video ID, using directly: %s\n", youtubeURL)
	}

	// Method 1: Try Spotify ID via SongLink (if it looks like a real Spotify ID)
	if youtubeURL == "" && req.SpotifyID != "" && !isYouTubeVideoID(req.SpotifyID) {
		GoLog("[YouTube] Looking up YouTube URL via SongLink for Spotify ID: %s\n", req.SpotifyID)
		songlink := NewSongLinkClient()
		youtubeURL, lookupErr = songlink.GetYouTubeURLFromSpotify(req.SpotifyID)
		if lookupErr != nil {
			GoLog("[YouTube] SongLink Spotify lookup failed: %v\n", lookupErr)
		} else {
			GoLog("[YouTube] Found YouTube URL via SongLink (Spotify): %s\n", youtubeURL)
		}
	}

	// Method 2: Try Deezer ID if Spotify lookup failed or Spotify ID not available
	if youtubeURL == "" && req.DeezerID != "" {
		GoLog("[YouTube] Looking up YouTube URL via SongLink for Deezer ID: %s\n", req.DeezerID)
		songlink := NewSongLinkClient()
		youtubeURL, lookupErr = songlink.GetYouTubeURLFromDeezer(req.DeezerID)
		if lookupErr != nil {
			GoLog("[YouTube] SongLink Deezer lookup failed: %v\n", lookupErr)
		} else {
			GoLog("[YouTube] Found YouTube URL via SongLink (Deezer): %s\n", youtubeURL)
		}
	}

	// Method 3: Try ISRC if both Spotify and Deezer failed
	if youtubeURL == "" && req.ISRC != "" {
		GoLog("[YouTube] Looking up YouTube URL via SongLink for ISRC: %s\n", req.ISRC)
		songlink := NewSongLinkClient()
		// First get Spotify ID from ISRC, then get YouTube URL
		availability, isrcErr := songlink.CheckTrackAvailability("", req.ISRC)
		if isrcErr == nil && availability.YouTube && availability.YouTubeURL != "" {
			youtubeURL = availability.YouTubeURL
			GoLog("[YouTube] Found YouTube URL via SongLink (ISRC): %s\n", youtubeURL)
		} else if isrcErr != nil {
			GoLog("[YouTube] SongLink ISRC lookup failed: %v\n", isrcErr)
		}
	}

	// Fallback: if we couldn't get URL from SongLink, return error
	// (Cobalt doesn't support search URLs, only direct video URLs)
	if youtubeURL == "" {
		return YouTubeDownloadResult{}, fmt.Errorf("could not find YouTube URL for track: %s - %s (no Spotify/Deezer ID available or track not on YouTube)", req.ArtistName, req.TrackName)
	}

	GoLog("[YouTube] Requesting download from Cobalt for: %s\n", youtubeURL)

	// Get download URL from Cobalt
	cobaltResp, err := downloader.GetDownloadURL(youtubeURL, quality)
	if err != nil {
		return YouTubeDownloadResult{}, fmt.Errorf("failed to get download URL: %w", err)
	}

	// Determine file extension based on quality
	var ext string
	var format string
	var bitrate int
	switch quality {
	case YouTubeQualityOpus256:
		ext = ".opus"
		format = "opus"
		bitrate = 256
	case YouTubeQualityMP3320:
		ext = ".mp3"
		format = "mp3"
		bitrate = 320
	}

	// Build filename
	filename := buildFilenameFromTemplate(req.FilenameFormat, map[string]interface{}{
		"title":  req.TrackName,
		"artist": req.ArtistName,
		"album":  req.AlbumName,
		"track":  req.TrackNumber,
		"year":   extractYear(req.ReleaseDate),
		"disc":   req.DiscNumber,
	})
	filename = sanitizeFilename(filename) + ext

	// Determine output path
	var outputPath string
	isSafOutput := isFDOutput(req.OutputFD) || strings.TrimSpace(req.OutputPath) != ""
	if isSafOutput {
		outputPath = strings.TrimSpace(req.OutputPath)
		if outputPath == "" && isFDOutput(req.OutputFD) {
			outputPath = fmt.Sprintf("/proc/self/fd/%d", req.OutputFD)
		}
	} else {
		outputPath = req.OutputDir + "/" + filename
	}

	GoLog("[YouTube] Downloading to: %s\n", outputPath)

	// Start parallel fetch for cover art and lyrics while downloading
	var parallelResult *ParallelDownloadResult
	if req.EmbedLyrics || req.CoverURL != "" {
		GoLog("[YouTube] Starting parallel fetch for cover and lyrics...\n")
		parallelResult = FetchCoverAndLyricsParallel(
			req.CoverURL,
			req.EmbedMaxQualityCover,
			req.SpotifyID,
			req.TrackName,
			req.ArtistName,
			req.EmbedLyrics,
			int64(req.DurationMS),
		)
	}

	// Download the file
	if err := downloader.DownloadFile(cobaltResp.URL, outputPath, req.OutputFD, req.ItemID); err != nil {
		return YouTubeDownloadResult{}, fmt.Errorf("download failed: %w", err)
	}

	// Extract lyrics LRC if available
	lyricsLRC := ""
	var coverData []byte
	if parallelResult != nil {
		if parallelResult.LyricsLRC != "" {
			lyricsLRC = parallelResult.LyricsLRC
			GoLog("[YouTube] Got lyrics from lrclib (%d lines)\n", len(parallelResult.LyricsData.Lines))
		}
		if parallelResult.CoverData != nil {
			coverData = parallelResult.CoverData
			GoLog("[YouTube] Got cover art (%d bytes)\n", len(coverData))
		}
	}

	return YouTubeDownloadResult{
		FilePath:    outputPath,
		Title:       req.TrackName,
		Artist:      req.ArtistName,
		Album:       req.AlbumName,
		ReleaseDate: req.ReleaseDate,
		TrackNumber: req.TrackNumber,
		DiscNumber:  req.DiscNumber,
		ISRC:        req.ISRC,
		Format:      format,
		Bitrate:     bitrate,
		LyricsLRC:   lyricsLRC,
		CoverData:   coverData,
	}, nil
}
