package rw

import (
	"compress/gzip"
	"fmt"
	"io"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// CreateReader creates a reader with automatic decompression based on file extension
func CreateReader(r io.Reader, filename string) (io.ReadCloser, error) {
	if strings.HasSuffix(filename, ".gz") || strings.HasSuffix(filename, ".tar.gz") {
		gzr, err := gzip.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		return gzr, nil
	} else if strings.HasSuffix(filename, ".zst") || strings.HasSuffix(filename, ".tar.zst") {
		zr, err := zstd.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("failed to create zstd reader: %w", err)
		}
		return io.NopCloser(zr.IOReadCloser()), nil
	}
	// No compression
	return io.NopCloser(r), nil
}
