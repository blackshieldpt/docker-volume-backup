package docker

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"
)

// GetVolumeSize estimates the size of a Docker volume in bytes
func GetVolumeSize(volume string) (int64, error) {
	// Use docker run to calculate size
	cmd := exec.Command("docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/data:ro", volume),
		"alpine",
		"sh", "-c", "du -sb /data | cut -f1")
	output, err := cmd.Output()
	if err != nil {
		// If we can't get size, return 0 (progress will be indeterminate)
		return 0, nil
	}

	var size int64
	_, err = fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &size)
	if err != nil {
		return 0, nil
	}
	return size, nil
}

// ValidateVolumeName checks if the provided volume name is valid based on predefined rules and returns an error if invalid.
func ValidateVolumeName(volume string) error {
	if volume == "" {
		return fmt.Errorf("volume name cannot be empty")
	}
	// Docker volume names can contain: a-z, A-Z, 0-9, -, _, and .
	// Must not contain path separators or special shell characters
	matched, err := regexp.MatchString(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`, volume)
	if err != nil {
		return err
	}
	if !matched {
		return fmt.Errorf("invalid volume name '%s': must start with alphanumeric and contain only a-z, A-Z, 0-9, -, _, .", volume)
	}
	return nil
}

// VolumeExists checks if a Docker volume with the given name exists.
// It returns true if the volume exists, false otherwise, along with any error encountered during execution.
func VolumeExists(volume string) (bool, error) {
	cmd := exec.Command("docker", "volume", "inspect", volume)
	err := cmd.Run()
	if err != nil {
		// Volume doesn't exist
		return false, nil
	}
	return true, nil
}

// CreateVolume creates a new Docker volume with the specified name. It returns an error if the volume creation fails.
func CreateVolume(volume string) error {
	log.Printf("Creating volume '%s'", volume)
	cmd := exec.Command("docker", "volume", "create", volume)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create volume: %v, output: %s", err, string(output))
	}
	return nil
}

// EnsureVolumeExists ensures that a Docker volume with the given name exists.
// If the volume does not exist, it attempts to create it.
// Returns an error if checking existence or creating the volume fails.
func EnsureVolumeExists(volume string) error {
	exists, err := VolumeExists(volume)
	if err != nil {
		return err
	}
	if !exists {
		return CreateVolume(volume)
	}
	return nil
}

// ClearVolume removes all contents of the specified Docker volume using an Alpine container.
// Returns an error if the operation fails.
func ClearVolume(volume string) error {
	log.Printf("Clearing volume '%s'", volume)
	// Use Alpine container to remove all contents from the volume
	cmd := exec.Command("docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/data", volume),
		"alpine",
		"sh", "-c", "rm -rf /data/* /data/..?* /data/.[!.]* 2>/dev/null || true")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clear volume: %v, output: %s", err, string(output))
	}
	return nil
}

// Helper function to check if Docker is available
func IsDockerAvailable() bool {
	cmd := exec.Command("docker", "version")
	err := cmd.Run()
	return err == nil
}
