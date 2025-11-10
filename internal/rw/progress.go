package rw

import (
	"io"

	"github.com/schollz/progressbar/v3"
)

// progressWriter wraps an io.Writer and updates a progress bar
type ProgressWriter struct {
	writer io.Writer
	bar    *progressbar.ProgressBar
}

func NewProgressWriter(writer io.Writer, bar *progressbar.ProgressBar) *ProgressWriter {
	return &ProgressWriter{
		writer: writer,
		bar:    bar,
	}
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	if pw.bar != nil {
		pw.bar.Add(n)
	}
	return n, err
}

// progressReader wraps an io.Reader and updates a progress bar
type ProgressReader struct {
	reader io.Reader
	bar    *progressbar.ProgressBar
}

func NewProgressReader(reader io.Reader, bar *progressbar.ProgressBar) *ProgressReader {
	return &ProgressReader{
		reader: reader,
		bar:    bar,
	}
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if pr.bar != nil {
		pr.bar.Add(n)
	}
	return n, err
}
