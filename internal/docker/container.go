package docker

import (
	"fmt"
	"os/exec"
	"strings"
)

// CreateContainerWithVolume creates a temporary container with the volume mounted
func CreateContainerWithVolume(volume string) (string, error) {
	cmd := exec.Command("docker", "create", "-v", volume+":/data", "alpine", "true")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// RemoveContainer removes a container
func RemoveContainer(containerID string) error {
	cmd := exec.Command("docker", "rm", containerID)
	return cmd.Run()
}
