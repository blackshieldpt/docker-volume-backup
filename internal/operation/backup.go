package operation

import (
	"archive/tar"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"docker-volume-backup/internal/docker"
	"docker-volume-backup/internal/rw"
	"docker-volume-backup/internal/s3"

	"github.com/schollz/progressbar/v3"
)

type Backup struct {
	volume       string
	compression  string
	showProgress bool
}

func NewBackup(volume string, compression string, showProgress bool) (*Backup, error) {
	if err := docker.ValidateVolumeName(volume); err != nil {
		return nil, err
	}
	exists, err := docker.VolumeExists(volume)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("volume '%s' does not exist", volume)
	}

	return &Backup{
		volume:       volume,
		compression:  compression,
		showProgress: showProgress,
	}, nil
}

// BackupToFile saves the volume data to the specified destination file path with optional validation and logging.
func (b *Backup) BackupToFile(dest string) error {
	if err := ValidateFilePath(dest); err != nil {
		return err
	}
	log.Printf("Backing up volume '%s' to %s", b.volume, dest)
	return b.runBackup(dest)
}

// BackupToS3 performs a backup of the volume to a local file, then uploads the file to the specified S3 path.
func (b *Backup) BackupToS3(s3Path string) error {
	if err := s3.ValidatePath(s3Path); err != nil {
		return err
	}
	// Create temporary file for backup
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("docker-volume-backup-%s-*.tar.gz", b.volume))
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpFilePath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpFilePath)

	// First backup to local file
	log.Printf("Creating temporary backup of volume '%s'", b.volume)
	if err := b.runBackup(tmpFilePath); err != nil {
		return fmt.Errorf("failed to create temporary backup: %v", err)
	}

	// Then upload to S3
	log.Printf("Uploading to S3: %s", s3Path)
	if err := s3.UploadFile(tmpFilePath, s3Path); err != nil {
		return err
	}

	log.Printf("Successfully backed up volume '%s' to %s", b.volume, s3Path)
	return nil
}

// runBackup performs a backup of the specified Docker volume to the destination file with optional compression and progress.
func (b *Backup) runBackup(dest string) error {
	// Get volume size for progress bar
	var bar *progressbar.ProgressBar
	if b.showProgress {
		volumeSize, err := docker.GetVolumeSize(b.volume)
		if err != nil {
			log.Printf("Warning: could not determine volume size: %v", err)
		}
		if volumeSize > 0 {
			bar = progressbar.DefaultBytes(
				volumeSize,
				"Backing up",
			)
		} else {
			bar = progressbar.DefaultBytes(
				-1, // indeterminate progress
				"Backing up",
			)
		}
		defer bar.Finish()
	}

	// Create output file
	outFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer outFile.Close()

	// Wrap output file with progress tracking if enabled
	var outWriter io.Writer = outFile
	if bar != nil {
		outWriter = rw.NewProgressWriter(outFile, bar)
	}

	// Create writer with compression
	writer, err := rw.CreateWriter(outWriter, b.compression)
	if err != nil {
		return fmt.Errorf("failed to create compressed writer: %w", err)
	}
	defer writer.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	// Use docker cp to copy volume contents to tar stream
	// Create a temporary container to access the volume
	containerID, err := docker.CreateContainerWithVolume(b.volume)
	if err != nil {
		return fmt.Errorf("failed to create temp container: %w", err)
	}
	defer docker.RemoveContainer(containerID)

	// Use docker cp to stream the volume data
	cmd := exec.Command("docker", "cp", containerID+":/data/.", "-")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start docker cp: %w", err)
	}

	// Copy the tar stream from docker cp to our compressed tar
	tarReader := tar.NewReader(stdout)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		if header.Typeflag == tar.TypeReg {
			if _, err := io.Copy(tarWriter, tarReader); err != nil {
				return fmt.Errorf("failed to write file data: %w", err)
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("docker cp failed: %w", err)
	}

	return nil
}
