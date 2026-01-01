package gobackend

import (
	"sync"
)

// DownloadProgress represents current download progress
type DownloadProgress struct {
	CurrentFile   string  `json:"current_file"`
	Progress      float64 `json:"progress"`
	Speed         float64 `json:"speed_mbps"`
	BytesTotal    int64   `json:"bytes_total"`
	BytesReceived int64   `json:"bytes_received"`
	IsDownloading bool    `json:"is_downloading"`
}

var (
	currentProgress DownloadProgress
	progressMu      sync.RWMutex
	downloadDir     string
	downloadDirMu   sync.RWMutex
)

// getProgress returns current download progress
func getProgress() DownloadProgress {
	progressMu.RLock()
	defer progressMu.RUnlock()
	return currentProgress
}

// SetDownloadProgress sets the current download progress (MB downloaded)
func SetDownloadProgress(mbDownloaded float64) {
	progressMu.Lock()
	defer progressMu.Unlock()
	currentProgress.Progress = mbDownloaded
	currentProgress.IsDownloading = true
}

// SetDownloadSpeed sets the current download speed
func SetDownloadSpeed(speedMBps float64) {
	progressMu.Lock()
	defer progressMu.Unlock()
	currentProgress.Speed = speedMBps
}

// SetCurrentFile sets the current file being downloaded and resets progress
func SetCurrentFile(filename string) {
	progressMu.Lock()
	defer progressMu.Unlock()
	// Reset progress for new file
	currentProgress.BytesReceived = 0
	currentProgress.BytesTotal = 0
	currentProgress.Progress = 0
	currentProgress.CurrentFile = filename
	currentProgress.IsDownloading = true
}

// ResetProgress resets the download progress
func ResetProgress() {
	progressMu.Lock()
	defer progressMu.Unlock()
	currentProgress = DownloadProgress{}
}

// setDownloadDir sets the default download directory
func setDownloadDir(path string) error {
	downloadDirMu.Lock()
	defer downloadDirMu.Unlock()
	downloadDir = path
	return nil
}

// getDownloadDir returns the default download directory
func getDownloadDir() string {
	downloadDirMu.RLock()
	defer downloadDirMu.RUnlock()
	return downloadDir
}

// SetDownloading sets the download status
func SetDownloading(status bool) {
	progressMu.Lock()
	defer progressMu.Unlock()
	currentProgress.IsDownloading = status
}

// SetBytesTotal sets total bytes to download
func SetBytesTotal(total int64) {
	progressMu.Lock()
	defer progressMu.Unlock()
	currentProgress.BytesTotal = total
}

// SetBytesReceived sets bytes received so far
func SetBytesReceived(received int64) {
	progressMu.Lock()
	defer progressMu.Unlock()
	currentProgress.BytesReceived = received
	if currentProgress.BytesTotal > 0 {
		currentProgress.Progress = float64(received) / float64(currentProgress.BytesTotal) * 100
	}
}

// ProgressWriter wraps io.Writer to track download progress
type ProgressWriter struct {
	writer  interface{ Write([]byte) (int, error) }
	total   int64
	current int64
}

// NewProgressWriter creates a new progress writer wrapping an io.Writer
func NewProgressWriter(w interface{ Write([]byte) (int, error) }) *ProgressWriter {
	// Reset bytes received when starting new download
	SetBytesReceived(0)
	return &ProgressWriter{
		writer:  w,
		current: 0,
		total:   0,
	}
}

// Write implements io.Writer
func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	if err != nil {
		return n, err
	}
	pw.current += int64(n)
	pw.total += int64(n)
	SetBytesReceived(pw.current)
	return n, nil
}

// GetTotal returns total bytes written
func (pw *ProgressWriter) GetTotal() int64 {
	return pw.total
}
