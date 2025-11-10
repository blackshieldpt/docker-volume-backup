package rw

import (
	"compress/gzip"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
)

// nopWriteCloser wraps an io.Writer to add a no-op Close method
type nopWriteCloser struct {
	io.Writer
}

func (n *nopWriteCloser) Close() error {
	return nil
}

// CreateWriter creates a writer with the specified compression
func CreateWriter(w io.Writer, compressionType string) (io.WriteCloser, error) {
	switch compressionType {
	case "none":
		return &nopWriteCloser{w}, nil
	case "gz":
		return gzip.NewWriter(w), nil
	case "zstd":
		return zstd.NewWriter(w)
	default:
		return nil, fmt.Errorf("unsupported compression type: %s", compressionType)
	}
}
