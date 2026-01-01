package gobackend

import (
	"os"
	"path/filepath"
	"strings"
)

// checkISRCExistsInternal checks if a file with the given ISRC exists (internal use)
func checkISRCExistsInternal(outputDir, isrc string) (string, bool) {
	if isrc == "" || outputDir == "" {
		return "", false
	}

	// Walk through directory looking for FLAC files
	var foundFile string
	filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Only check FLAC files
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".flac") {
			return nil
		}

		// Read metadata from file
		metadata, err := ReadMetadata(path)
		if err != nil {
			return nil
		}

		// Check if ISRC matches
		if metadata.ISRC == isrc {
			foundFile = path
			return filepath.SkipAll // Stop walking
		}

		return nil
	})

	if foundFile != "" {
		return foundFile, true
	}

	return "", false
}

// CheckISRCExists is the exported version for gomobile (returns string, error)
// Returns the filepath if exists, empty string if not
func CheckISRCExists(outputDir, isrc string) (string, error) {
	filepath, _ := checkISRCExistsInternal(outputDir, isrc)
	return filepath, nil
}

// CheckFileExists checks if a file with the given name exists
func CheckFileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Size() > 0
}
