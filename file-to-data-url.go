package main

import (
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"path/filepath"
)

func FileToDataURL(relPath string) (string, error) {
	// Read file contents
	data, err := os.ReadFile(relPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Detect MIME type based on file extension
	ext := filepath.Ext(relPath)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		// Fallback if MIME type can't be detected
		mimeType = "application/octet-stream"
	}

	// Encode to Base64
	encoded := base64.StdEncoding.EncodeToString(data)

	// Build data URL
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)

	return dataURL, nil
}
