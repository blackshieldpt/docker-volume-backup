//go:build integration
// +build integration

package docker

import (
	"os/exec"
	"testing"
)

func TestVolumeExists(t *testing.T) {
	// This test requires Docker to be running
	if !IsDockerAvailable() {
		t.Skip("Docker is not available, skipping integration test")
	}

	// Test non-existent volume
	exists, err := VolumeExists("nonexistent-test-volume-xyz123")
	if err != nil {
		t.Errorf("VolumeExists() error: %v", err)
	}
	if exists {
		t.Error("VolumeExists() returned true for non-existent volume")
	}
}

func TestCreateAndDeleteVolume(t *testing.T) {
	if !IsDockerAvailable() {
		t.Skip("Docker is not available, skipping integration test")
	}

	volumeName := "test-volume-create-delete-xyz123"

	// Clean up any existing test volume
	exec.Command("docker", "volume", "rm", volumeName).Run()

	// Create volume
	err := CreateVolume(volumeName)
	if err != nil {
		t.Fatalf("CreateVolume() error: %v", err)
	}

	// Verify it exists
	exists, err := VolumeExists(volumeName)
	if err != nil {
		t.Errorf("VolumeExists() error: %v", err)
	}
	if !exists {
		t.Error("Volume was not created successfully")
	}

	// Clean up
	cmd := exec.Command("docker", "volume", "rm", volumeName)
	if err := cmd.Run(); err != nil {
		t.Errorf("Failed to clean up test volume: %v", err)
	}
}

func TestEnsureVolumeExists(t *testing.T) {
	if !IsDockerAvailable() {
		t.Skip("Docker is not available, skipping integration test")
	}

	volumeName := "test-volume-ensure-xyz123"

	// Clean up any existing test volume
	exec.Command("docker", "volume", "rm", volumeName).Run()

	// Ensure volume exists (should create it)
	err := EnsureVolumeExists(volumeName)
	if err != nil {
		t.Fatalf("EnsureVolumeExists() error: %v", err)
	}

	// Verify it exists
	exists, err := VolumeExists(volumeName)
	if err != nil {
		t.Errorf("VolumeExists() error: %v", err)
	}
	if !exists {
		t.Error("EnsureVolumeExists() did not create the volume")
	}

	// Call again (should not error)
	err = EnsureVolumeExists(volumeName)
	if err != nil {
		t.Errorf("EnsureVolumeExists() on existing volume error: %v", err)
	}

	// Clean up
	cmd := exec.Command("docker", "volume", "rm", volumeName)
	if err := cmd.Run(); err != nil {
		t.Errorf("Failed to clean up test volume: %v", err)
	}
}

func TestValidateVolumeName(t *testing.T) {
	tests := []struct {
		name      string
		volume    string
		shouldErr bool
	}{
		{"valid simple", "myvolume", false},
		{"valid with dash", "my-volume", false},
		{"valid with underscore", "my_volume", false},
		{"valid with dot", "my.volume", false},
		{"valid complex", "my-data_volume.1", false},
		{"empty", "", true},
		{"with slash", "my/volume", true},
		{"with backslash", "my\\volume", true},
		{"with space", "my volume", true},
		{"with special chars", "my$volume", true},
		{"with pipe", "my|volume", true},
		{"with semicolon", "my;volume", true},
		{"with ampersand", "my&volume", true},
		{"starts with dash", "-myvolume", true},
		{"starts with dot", ".myvolume", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVolumeName(tt.volume)
			if tt.shouldErr && err == nil {
				t.Errorf("ValidateVolumeName(%q) expected error but got none", tt.volume)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("ValidateVolumeName(%q) unexpected error: %v", tt.volume, err)
			}
		})
	}
}
