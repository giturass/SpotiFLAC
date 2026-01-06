package gobackend

import (
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

const (
	deezerSearchURL  = "https://api.deezer.com/search"
	deezerTrackURL   = "https://api.deezer.com/track/%s"
	deezerAlbumURL   = "https://api.deezer.com/album/%s"
	deezerArtistURL  = "https://api.deezer.com/artist/%s"
	deezerPlaylistURL = "https://api.deezer.com/playlist/%s"
	
	deezerCacheTTL = 10 * time.Minute
)

// DeezerClient handles Deezer API interactions (no auth required)
type DeezerClient struct {
	httpClient  *http.Client
	searchCache map[string]*cacheEntry
	albumCache  map[string]*cacheEntry
	artistCache map[string]*cacheEntry
	cacheMu     sync.RWMutex
}

// Singleton instance
var (
	deezerClient     *DeezerClient
	deezerClientOnce sync.Once
)

// GetDeezerClient returns singleton Deezer client
func GetDeezerClient() *DeezerClient {
	deezerClientOnce.Do(func() {
		deezerClient = &DeezerClient{
			httpClient:  NewHTTPClientWithTimeout(15 * time.Second),
			searchCache: make(map[string]*cacheEntry),
			albumCache:  make(map[string]*cacheEntry),
			artistCache: make(map[string]*cacheEntry),
		}
	})
	return deezerClient
}

// Deezer API response types
type deezerTrack struct {
	ID             int64  `json:"id"`
	Title          string `json:"title"`
	Duration       int    `json:"duration"` // in seconds
	TrackPosition  int    `json:"track_position"`
	DiskNumber     int    `json:"disk_number"`
	ISRC           string `json:"isrc"`
	Link           string `json:"link"`
	Artist         deezerArtist `json:"artist"`
	Album          deezerAlbumSimple `json:"album"`
	Contributors   []deezerArtist `json:"contributors"`
}

type deezerArtist struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Picture   string `json:"picture"`
	PictureMedium string `json:"picture_medium"`
	PictureBig    string `json:"picture_big"`
	PictureXL     string `json:"picture_xl"`
	NbFan     int    `json:"nb_fan"`
}

type deezerAlbumSimple struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Cover       string `json:"cover"`
	CoverMedium string `json:"cover_medium"`
	CoverBig    string `json:"cover_big"`
	CoverXL     string `json:"cover_xl"`
}

type deezerAlbumFull struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Cover       string `json:"cover"`
	CoverMedium string `json:"cover_medium"`
	CoverBig    string `json:"cover_big"`
	CoverXL     string `json:"cover_xl"`
	ReleaseDate string `json:"release_date"`
	NbTracks    int    `json:"nb_tracks"`
	Artist      deezerArtist `json:"artist"`
	Contributors []deezerArtist `json:"contributors"`
	Tracks      struct {
		Data []deezerTrack `json:"data"`
	} `json:"tracks"`
}

type deezerArtistFull struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	PictureMedium string `json:"picture_medium"`
	PictureBig    string `json:"picture_big"`
	PictureXL     string `json:"picture_xl"`
	NbFan         int    `json:"nb_fan"`
	NbAlbum       int    `json:"nb_album"`
}

type deezerPlaylistFull struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Picture     string `json:"picture"`
	PictureMedium string `json:"picture_medium"`
	PictureBig    string `json:"picture_big"`
	PictureXL     string `json:"picture_xl"`
	NbTracks    int    `json:"nb_tracks"`
	Creator     struct {
		Name string `json:"name"`
	} `json:"creator"`
	Tracks      struct {
		Data []deezerTrack `json:"data"`
	} `json:"tracks"`
}

// SearchAll searches for tracks and artists on Deezer
func (c *DeezerClient) SearchAll(ctx context.Context, query string, trackLimit, artistLimit int) (*SearchAllResult, error) {
	cacheKey := fmt.Sprintf("deezer:all:%s:%d:%d", query, trackLimit, artistLimit)
	
	c.cacheMu.RLock()
	if entry, ok := c.searchCache[cacheKey]; ok && !entry.isExpired() {
		c.cacheMu.RUnlock()
		return entry.data.(*SearchAllResult), nil
	}
	c.cacheMu.RUnlock()

	result := &SearchAllResult{
		Tracks:  make([]TrackMetadata, 0),
		Artists: make([]SearchArtistResult, 0),
	}

	// Search tracks
	trackURL := fmt.Sprintf("%s/track?q=%s&limit=%d", deezerSearchURL, url.QueryEscape(query), trackLimit)
	var trackResp struct {
		Data []deezerTrack `json:"data"`
	}
	if err := c.getJSON(ctx, trackURL, &trackResp); err != nil {
		return nil, fmt.Errorf("deezer track search failed: %w", err)
	}

	for _, track := range trackResp.Data {
		result.Tracks = append(result.Tracks, c.convertTrack(track))
	}

	// Search artists
	artistURL := fmt.Sprintf("%s/artist?q=%s&limit=%d", deezerSearchURL, url.QueryEscape(query), artistLimit)
	var artistResp struct {
		Data []deezerArtist `json:"data"`
	}
	if err := c.getJSON(ctx, artistURL, &artistResp); err == nil {
		for _, artist := range artistResp.Data {
			result.Artists = append(result.Artists, SearchArtistResult{
				ID:         fmt.Sprintf("deezer:%d", artist.ID),
				Name:       artist.Name,
				Images:     c.getBestArtistImage(artist),
				Followers:  artist.NbFan,
				Popularity: 0,
			})
		}
	}

	// Cache result
	c.cacheMu.Lock()
	c.searchCache[cacheKey] = &cacheEntry{
		data:      result,
		expiresAt: time.Now().Add(deezerCacheTTL),
	}
	c.cacheMu.Unlock()

	return result, nil
}

// GetTrack fetches a single track by Deezer ID
func (c *DeezerClient) GetTrack(ctx context.Context, trackID string) (*TrackResponse, error) {
	trackURL := fmt.Sprintf(deezerTrackURL, trackID)
	
	var track deezerTrack
	if err := c.getJSON(ctx, trackURL, &track); err != nil {
		return nil, err
	}

	return &TrackResponse{
		Track: c.convertTrack(track),
	}, nil
}

// GetAlbum fetches album with tracks
func (c *DeezerClient) GetAlbum(ctx context.Context, albumID string) (*AlbumResponsePayload, error) {
	c.cacheMu.RLock()
	if entry, ok := c.albumCache[albumID]; ok && !entry.isExpired() {
		c.cacheMu.RUnlock()
		return entry.data.(*AlbumResponsePayload), nil
	}
	c.cacheMu.RUnlock()

	albumURL := fmt.Sprintf(deezerAlbumURL, albumID)
	
	var album deezerAlbumFull
	if err := c.getJSON(ctx, albumURL, &album); err != nil {
		return nil, err
	}

	albumImage := c.getBestAlbumImage(album)
	artistName := album.Artist.Name
	if len(album.Contributors) > 0 {
		names := make([]string, len(album.Contributors))
		for i, a := range album.Contributors {
			names[i] = a.Name
		}
		artistName = strings.Join(names, ", ")
	}

	info := AlbumInfoMetadata{
		TotalTracks: album.NbTracks,
		Name:        album.Title,
		ReleaseDate: album.ReleaseDate,
		Artists:     artistName,
		Images:      albumImage,
	}

	tracks := make([]AlbumTrackMetadata, 0, len(album.Tracks.Data))
	for _, track := range album.Tracks.Data {
		// Need to fetch full track info for ISRC
		fullTrack, _ := c.fetchFullTrack(ctx, fmt.Sprintf("%d", track.ID))
		isrc := ""
		if fullTrack != nil {
			isrc = fullTrack.ISRC
		}

		tracks = append(tracks, AlbumTrackMetadata{
			SpotifyID:   fmt.Sprintf("deezer:%d", track.ID),
			Artists:     track.Artist.Name,
			Name:        track.Title,
			AlbumName:   album.Title,
			AlbumArtist: artistName,
			DurationMS:  track.Duration * 1000,
			Images:      albumImage,
			ReleaseDate: album.ReleaseDate,
			TrackNumber: track.TrackPosition,
			TotalTracks: album.NbTracks,
			DiscNumber:  track.DiskNumber,
			ExternalURL: track.Link,
			ISRC:        isrc,
			AlbumID:     fmt.Sprintf("deezer:%d", album.ID),
		})
	}

	result := &AlbumResponsePayload{
		AlbumInfo: info,
		TrackList: tracks,
	}

	c.cacheMu.Lock()
	c.albumCache[albumID] = &cacheEntry{
		data:      result,
		expiresAt: time.Now().Add(deezerCacheTTL),
	}
	c.cacheMu.Unlock()

	return result, nil
}

// GetArtist fetches artist with albums
func (c *DeezerClient) GetArtist(ctx context.Context, artistID string) (*ArtistResponsePayload, error) {
	c.cacheMu.RLock()
	if entry, ok := c.artistCache[artistID]; ok && !entry.isExpired() {
		c.cacheMu.RUnlock()
		return entry.data.(*ArtistResponsePayload), nil
	}
	c.cacheMu.RUnlock()

	// Fetch artist info
	artistURL := fmt.Sprintf(deezerArtistURL, artistID)
	var artist deezerArtistFull
	if err := c.getJSON(ctx, artistURL, &artist); err != nil {
		return nil, err
	}

	artistInfo := ArtistInfoMetadata{
		ID:         fmt.Sprintf("deezer:%d", artist.ID),
		Name:       artist.Name,
		Images:     c.getBestArtistImageFull(artist),
		Followers:  artist.NbFan,
		Popularity: 0,
	}

	// Fetch artist albums
	albumsURL := fmt.Sprintf("%s/albums?limit=100", fmt.Sprintf(deezerArtistURL, artistID))
	var albumsResp struct {
		Data []struct {
			ID          int64  `json:"id"`
			Title       string `json:"title"`
			ReleaseDate string `json:"release_date"`
			NbTracks    int    `json:"nb_tracks"`
			Cover       string `json:"cover"`
			CoverMedium string `json:"cover_medium"`
			CoverBig    string `json:"cover_big"`
			CoverXL     string `json:"cover_xl"`
			RecordType  string `json:"record_type"` // album, single, ep, compile
		} `json:"data"`
	}

	albums := make([]ArtistAlbumMetadata, 0)
	if err := c.getJSON(ctx, albumsURL, &albumsResp); err == nil {
		for _, album := range albumsResp.Data {
			albumType := album.RecordType
			if albumType == "compile" {
				albumType = "compilation"
			}
			
			coverURL := album.CoverXL
			if coverURL == "" {
				coverURL = album.CoverBig
			}
			if coverURL == "" {
				coverURL = album.CoverMedium
			}
			if coverURL == "" {
				coverURL = album.Cover
			}

			albums = append(albums, ArtistAlbumMetadata{
				ID:          fmt.Sprintf("deezer:%d", album.ID),
				Name:        album.Title,
				ReleaseDate: album.ReleaseDate,
				TotalTracks: album.NbTracks,
				Images:      coverURL,
				AlbumType:   albumType,
				Artists:     artist.Name,
			})
		}
	}

	result := &ArtistResponsePayload{
		ArtistInfo: artistInfo,
		Albums:     albums,
	}

	c.cacheMu.Lock()
	c.artistCache[artistID] = &cacheEntry{
		data:      result,
		expiresAt: time.Now().Add(deezerCacheTTL),
	}
	c.cacheMu.Unlock()

	return result, nil
}

// GetPlaylist fetches playlist with tracks
func (c *DeezerClient) GetPlaylist(ctx context.Context, playlistID string) (*PlaylistResponsePayload, error) {
	playlistURL := fmt.Sprintf(deezerPlaylistURL, playlistID)
	
	var playlist deezerPlaylistFull
	if err := c.getJSON(ctx, playlistURL, &playlist); err != nil {
		return nil, err
	}

	playlistImage := playlist.PictureXL
	if playlistImage == "" {
		playlistImage = playlist.PictureBig
	}
	if playlistImage == "" {
		playlistImage = playlist.PictureMedium
	}

	var info PlaylistInfoMetadata
	info.Tracks.Total = playlist.NbTracks
	info.Owner.DisplayName = playlist.Creator.Name
	info.Owner.Name = playlist.Title
	info.Owner.Images = playlistImage

	tracks := make([]AlbumTrackMetadata, 0, len(playlist.Tracks.Data))
	for _, track := range playlist.Tracks.Data {
		albumImage := track.Album.CoverXL
		if albumImage == "" {
			albumImage = track.Album.CoverBig
		}
		if albumImage == "" {
			albumImage = track.Album.CoverMedium
		}

		// Fetch full track for ISRC
		fullTrack, _ := c.fetchFullTrack(ctx, fmt.Sprintf("%d", track.ID))
		isrc := ""
		releaseDate := ""
		if fullTrack != nil {
			isrc = fullTrack.ISRC
		}

		tracks = append(tracks, AlbumTrackMetadata{
			SpotifyID:   fmt.Sprintf("deezer:%d", track.ID),
			Artists:     track.Artist.Name,
			Name:        track.Title,
			AlbumName:   track.Album.Title,
			AlbumArtist: track.Artist.Name,
			DurationMS:  track.Duration * 1000,
			Images:      albumImage,
			ReleaseDate: releaseDate,
			TrackNumber: track.TrackPosition,
			DiscNumber:  track.DiskNumber,
			ExternalURL: track.Link,
			ISRC:        isrc,
			AlbumID:     fmt.Sprintf("deezer:%d", track.Album.ID),
		})
	}

	return &PlaylistResponsePayload{
		PlaylistInfo: info,
		TrackList:    tracks,
	}, nil
}

// SearchByISRC searches for a track by ISRC
func (c *DeezerClient) SearchByISRC(ctx context.Context, isrc string) (*TrackMetadata, error) {
	searchURL := fmt.Sprintf("%s/track?q=isrc:%s&limit=1", deezerSearchURL, isrc)
	
	var resp struct {
		Data []deezerTrack `json:"data"`
	}
	if err := c.getJSON(ctx, searchURL, &resp); err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no track found for ISRC: %s", isrc)
	}

	track := c.convertTrack(resp.Data[0])
	return &track, nil
}

func (c *DeezerClient) fetchFullTrack(ctx context.Context, trackID string) (*deezerTrack, error) {
	trackURL := fmt.Sprintf(deezerTrackURL, trackID)
	var track deezerTrack
	if err := c.getJSON(ctx, trackURL, &track); err != nil {
		return nil, err
	}
	return &track, nil
}

func (c *DeezerClient) convertTrack(track deezerTrack) TrackMetadata {
	artistName := track.Artist.Name
	if len(track.Contributors) > 0 {
		names := make([]string, len(track.Contributors))
		for i, a := range track.Contributors {
			names[i] = a.Name
		}
		artistName = strings.Join(names, ", ")
	}

	albumImage := track.Album.CoverXL
	if albumImage == "" {
		albumImage = track.Album.CoverBig
	}
	if albumImage == "" {
		albumImage = track.Album.CoverMedium
	}
	if albumImage == "" {
		albumImage = track.Album.Cover
	}

	return TrackMetadata{
		SpotifyID:   fmt.Sprintf("deezer:%d", track.ID),
		Artists:     artistName,
		Name:        track.Title,
		AlbumName:   track.Album.Title,
		AlbumArtist: track.Artist.Name,
		DurationMS:  track.Duration * 1000,
		Images:      albumImage,
		TrackNumber: track.TrackPosition,
		DiscNumber:  track.DiskNumber,
		ExternalURL: track.Link,
		ISRC:        track.ISRC,
	}
}

func (c *DeezerClient) getBestArtistImage(artist deezerArtist) string {
	if artist.PictureXL != "" {
		return artist.PictureXL
	}
	if artist.PictureBig != "" {
		return artist.PictureBig
	}
	if artist.PictureMedium != "" {
		return artist.PictureMedium
	}
	return artist.Picture
}

func (c *DeezerClient) getBestArtistImageFull(artist deezerArtistFull) string {
	if artist.PictureXL != "" {
		return artist.PictureXL
	}
	if artist.PictureBig != "" {
		return artist.PictureBig
	}
	if artist.PictureMedium != "" {
		return artist.PictureMedium
	}
	return artist.Picture
}

func (c *DeezerClient) getBestAlbumImage(album deezerAlbumFull) string {
	if album.CoverXL != "" {
		return album.CoverXL
	}
	if album.CoverBig != "" {
		return album.CoverBig
	}
	if album.CoverMedium != "" {
		return album.CoverMedium
	}
	return album.Cover
}

func (c *DeezerClient) getJSON(ctx context.Context, endpoint string, dst interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("deezer API returned status %d: %s", resp.StatusCode, string(body))
	}

	return json.Unmarshal(body, dst)
}

// parseDeezerURL is internal function, returns type and ID
func parseDeezerURL(input string) (string, string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", "", fmt.Errorf("empty URL")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", "", err
	}

	if parsed.Host != "www.deezer.com" && parsed.Host != "deezer.com" && parsed.Host != "deezer.page.link" {
		return "", "", fmt.Errorf("not a Deezer URL")
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	
	// Skip language prefix if present (e.g., /en/, /fr/)
	if len(parts) > 0 && len(parts[0]) == 2 {
		parts = parts[1:]
	}

	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid Deezer URL format")
	}

	resourceType := parts[0]
	resourceID := parts[1]

	switch resourceType {
	case "track", "album", "artist", "playlist":
		return resourceType, resourceID, nil
	default:
		return "", "", fmt.Errorf("unsupported Deezer resource type: %s", resourceType)
	}
}
