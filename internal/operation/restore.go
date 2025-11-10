package operation

import (
	"archive/tar"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"docker-volume-backup/internal/docker"
	"docker-volume-backup/internal/rw"
	"docker-volume-backup/internal/s3"

	"github.com/schollz/progressbar/v3"
)

type Restore struct {
	volume       string
	showProgress bool
}

func NewRestore(volume string, showProgress bool) (*Restore, error) {
	// Validate inputs
	if err := docker.ValidateVolumeName(volume); err != nil {
		return nil, err
	}
	return &Restore{volume, showProgress}, nil
}

// RestoreFromFile restores a volume from the specified file path. It optionally overwrites the target if it already exists.
func (r *Restore) RestoreFromFile(src string, overwrite bool) error {
	if err := ValidateFilePath(src); err != nil {
		return err
	}

	// Check if volume exists
	exists, err := docker.VolumeExists(r.volume)
	if err != nil {
		return err
	}

	if exists {
		// Volume exists - check if overwrite flag is set
		if !overwrite {
			return fmt.Errorf("volume '%s' already exists. Use --overwrite flag to clear and restore, or delete the volume first", r.volume)
		}
		// Clear the existing volume before restore
		if err := docker.ClearVolume(r.volume); err != nil {
			return err
		}
	} else {
		// Create new volume
		if err := docker.CreateVolume(r.volume); err != nil {
			return err
		}
	}

	log.Printf("Restoring %s to volume '%s'", src, r.volume)
	return r.runRestore(src)
}

// RestoreFromS3 restores a Docker volume from an S3 path. Requires the S3 path, and an overwrite flag for existing volumes.
func (r *Restore) RestoreFromS3(path string, overwrite bool) error {
	if err := s3.ValidatePath(path); err != nil {
		return err
	}

	// Check if volume exists
	exists, err := docker.VolumeExists(r.volume)
	if err != nil {
		return err
	}

	if exists {
		// Volume exists - check if overwrite flag is set
		if !overwrite {
			return fmt.Errorf("volume '%s' already exists. Use --overwrite flag to clear and restore, or delete the volume first", r.volume)
		}
		// Clear the existing volume before restore
		if err := docker.ClearVolume(r.volume); err != nil {
			return err
		}
	} else {
		// Create new volume
		if err := docker.CreateVolume(r.volume); err != nil {
			return err
		}
	}

	// Create temporary file for download with proper permissions
	// Preserve the original file extension for proper compression detection
	ext := ".tar.gz" // default
	if strings.HasSuffix(path, ".tar.zst") || strings.HasSuffix(path, ".zst") {
		ext = ".tar.zst"
	} else if strings.HasSuffix(path, ".tar") {
		ext = ".tar"
	}
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("docker-volume-restore-%s-*%s", r.volume, ext))
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpFilePath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpFilePath)

	// Download from S3
	log.Printf("Downloading from S3: %s", path)
	if err := s3.DownloadFile(path, tmpFilePath); err != nil {
		return err
	}

	// Restore from local file
	log.Printf("Restoring volume '%s' from downloaded backup", r.volume)
	if err := r.runRestore(tmpFilePath); err != nil {
		return fmt.Errorf("failed to restore from downloaded backup: %v", err)
	}

	log.Printf("Successfully restored volume '%s' from %s", r.volume, path)
	return nil
}

// runRestore performs the core logic to restore the contents of a compressed tar archive to a Docker volume.
func (r *Restore) runRestore(src string) error {
	// Get file size for progress bar
	var bar *progressbar.ProgressBar
	if r.showProgress {
		fileSize, err := GetFileSize(src)
		if err != nil {
			log.Printf("Warning: could not determine file size: %v", err)
		}

		if fileSize > 0 {
			bar = progressbar.DefaultBytes(
				fileSize,
				"Restoring",
			)
		} else {
			bar = progressbar.DefaultBytes(
				-1, // indeterminate progress
				"Restoring",
			)
		}
		defer bar.Finish()
	}

	// Open backup file
	inFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer inFile.Close()

	// Wrap input file with progress tracking if enabled
	var inReader io.Reader = inFile
	if bar != nil {
		inReader = rw.NewProgressReader(inFile, bar)
	}

	// Create reader with decompression
	reader, err := rw.CreateReader(inReader, src)
	if err != nil {
		return fmt.Errorf("failed to create decompressed reader: %w", err)
	}
	defer reader.Close()

	// Create tar reader
	tarReader := tar.NewReader(reader)

	// Create a temporary container to access the volume
	containerID, err := docker.CreateContainerWithVolume(r.volume)
	if err != nil {
		return fmt.Errorf("failed to create temp container: %w", err)
	}
	defer docker.RemoveContainer(containerID)

	// Use docker cp to write the tar stream to the volume
	cmd := exec.Command("docker", "cp", "-", containerID+":/data/")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start docker cp: %w", err)
	}

	// Write tar stream to docker cp
	tarWriter := tar.NewWriter(stdin)
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

	tarWriter.Close()
	stdin.Close()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("docker cp failed: %w", err)
	}

	return nil
}
