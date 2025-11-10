//go:build integration
// +build integration

package operation

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"docker-volume-backup/internal/docker"
)

func TestBackupAndRestoreWorkflow(t *testing.T) {
	if !docker.IsDockerAvailable() {
		t.Skip("Docker is not available, skipping integration test")
	}

	compressionTests := []struct {
		name       string
		compress   string
		fileExt    string
	}{
		{"gzip", "gz", ".tar.gz"},
		{"zstd", "zstd", ".tar.zst"},
		{"none", "none", ".tar"},
	}

	for _, tt := range compressionTests {
		t.Run(tt.name, func(t *testing.T) {
			volumeName := "test-volume-backup-xyz123-" + tt.name
			tmpFile, err := os.CreateTemp("", "test-backup-*"+tt.fileExt)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			backupFile := tmpFile.Name()
			tmpFile.Close()
			defer os.Remove(backupFile)

			// Clean up any existing test volume
			exec.Command("docker", "volume", "rm", volumeName).Run()

			// Create test volume and write data to it
			if err := docker.CreateVolume(volumeName); err != nil {
				t.Fatalf("CreateVolume() error: %v", err)
			}
			defer exec.Command("docker", "volume", "rm", volumeName).Run()

			// Write test data to volume
			testData := "Hello, Docker Volume Backup!"
			cmd := exec.Command("docker", "run", "--rm",
				"-v", volumeName+":/data",
				"alpine",
				"sh", "-c", "echo '"+testData+"' > /data/test.txt")
			if err := cmd.Run(); err != nil {
				t.Fatalf("Failed to write test data: %v", err)
			}

			// Backup the volume with specific compression
			bkpOp, err := NewBackup(volumeName, tt.compress, false)
			if err != nil {
				t.Fatalf("NewBackup() error: %v", err)
			}
			if err := bkpOp.runBackup(backupFile); err != nil {
				t.Fatalf("runBackup() error: %v", err)
			}

			// Verify backup file exists
			if _, err := os.Stat(backupFile); os.IsNotExist(err) {
				t.Fatal("Backup file was not created")
			}

			// Remove the volume
			cmd = exec.Command("docker", "volume", "rm", volumeName)
			if err := cmd.Run(); err != nil {
				t.Fatalf("Failed to remove volume: %v", err)
			}

			// Restore the volume
			if err := docker.EnsureVolumeExists(volumeName); err != nil {
				t.Fatalf("EnsureVolumeExists() error: %v", err)
			}

			restoreOp, err := NewRestore(volumeName, false)
			if err != nil {
				t.Fatalf("NewRestore() error: %v", err)
			}
			if err := restoreOp.runRestore(backupFile); err != nil {
				t.Fatalf("runRestore() error: %v", err)
			}

			// Verify data was restored
			cmd = exec.Command("docker", "run", "--rm",
				"-v", volumeName+":/data",
				"alpine",
				"cat", "/data/test.txt")
			output, err := cmd.Output()
			if err != nil {
				t.Fatalf("Failed to read restored data: %v", err)
			}

			if string(output) != testData+"\n" {
				t.Errorf("Restored data mismatch: got %q, want %q", string(output), testData)
			}
		})
	}
}

func TestRestoreToExistingVolumeWithoutOverwrite(t *testing.T) {
	if !docker.IsDockerAvailable() {
		t.Skip("Docker is not available, skipping integration test")
	}

	compressionTests := []struct {
		name     string
		compress string
		fileExt  string
	}{
		{"gzip", "gz", ".tar.gz"},
		{"zstd", "zstd", ".tar.zst"},
	}

	for _, tt := range compressionTests {
		t.Run(tt.name, func(t *testing.T) {
			volumeName := "test-volume-existing-xyz123-" + tt.name
			tmpFile, err := os.CreateTemp("", "test-existing-backup-*"+tt.fileExt)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			backupFile := tmpFile.Name()
			tmpFile.Close()
			defer os.Remove(backupFile)

			// Clean up
			exec.Command("docker", "volume", "rm", volumeName).Run()

			// Create volume with existing data
			if err := docker.CreateVolume(volumeName); err != nil {
				t.Fatalf("CreateVolume() error: %v", err)
			}
			defer exec.Command("docker", "volume", "rm", volumeName).Run()

			cmd := exec.Command("docker", "run", "--rm",
				"-v", volumeName+":/data",
				"alpine",
				"sh", "-c", "echo 'existing data' > /data/existing.txt")
			if err := cmd.Run(); err != nil {
				t.Fatalf("Failed to write existing data: %v", err)
			}

			// Create a different backup
			backupOp, err := NewBackup(volumeName, tt.compress, false)
			if err != nil {
				t.Fatalf("NewBackup() error: %v", err)
			}
			if err := backupOp.runBackup(backupFile); err != nil {
				t.Fatalf("runBackup() error: %v", err)
			}

			// Try to restore without --overwrite flag (should fail)
			restoreOp, err := NewRestore(volumeName, false)
			if err != nil {
				t.Fatalf("NewRestore() error: %v", err)
			}
			err = restoreOp.RestoreFromFile(backupFile, false)
			if err == nil {
				t.Error("restoreFromFile() should have failed for existing volume without --overwrite")
			}
			if !strings.Contains(err.Error(), "already exists") {
				t.Errorf("Error should mention volume already exists, got: %v", err)
			}
		})
	}
}

func TestRestoreToExistingVolumeWithOverwrite(t *testing.T) {
	if !docker.IsDockerAvailable() {
		t.Skip("Docker is not available, skipping integration test")
	}

	compressionTests := []struct {
		name     string
		compress string
		fileExt  string
	}{
		{"gzip", "gz", ".tar.gz"},
		{"zstd", "zstd", ".tar.zst"},
	}

	for _, tt := range compressionTests {
		t.Run(tt.name, func(t *testing.T) {
			volumeName := "test-volume-overwrite-xyz123-" + tt.name
			tmpFile, err := os.CreateTemp("", "test-overwrite-backup-*"+tt.fileExt)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			backupFile := tmpFile.Name()
			tmpFile.Close()
			defer os.Remove(backupFile)

			// Clean up
			exec.Command("docker", "volume", "rm", volumeName).Run()

			// Create volume with existing data
			if err := docker.CreateVolume(volumeName); err != nil {
				t.Fatalf("CreateVolume() error: %v", err)
			}
			defer exec.Command("docker", "volume", "rm", volumeName).Run()

			// Write existing data
			cmd := exec.Command("docker", "run", "--rm",
				"-v", volumeName+":/data",
				"alpine",
				"sh", "-c", "echo 'old data' > /data/old.txt")
			if err := cmd.Run(); err != nil {
				t.Fatalf("Failed to write existing data: %v", err)
			}

			// Create backup with different data from a temp volume
			tempVolume := "test-temp-backup-volume-xyz123-" + tt.name
			if err := docker.CreateVolume(tempVolume); err != nil {
				t.Fatalf("CreateVolume() error: %v", err)
			}
			defer exec.Command("docker", "volume", "rm", tempVolume).Run()

			cmd = exec.Command("docker", "run", "--rm",
				"-v", tempVolume+":/data",
				"alpine",
				"sh", "-c", "echo 'new backup data' > /data/new.txt")
			if err := cmd.Run(); err != nil {
				t.Fatalf("Failed to write backup data: %v", err)
			}

			backupOp, err := NewBackup(tempVolume, tt.compress, false)
			if err != nil {
				t.Fatalf("NewBackup() error: %v", err)
			}
			if err := backupOp.runBackup(backupFile); err != nil {
				t.Fatalf("runBackup() error: %v", err)
			}

			// Restore with --overwrite flag (should succeed and clear old data)
			restoreOp, err := NewRestore(volumeName, false)
			if err != nil {
				t.Fatalf("NewRestore() error: %v", err)
			}
			if err := restoreOp.RestoreFromFile(backupFile, true); err != nil {
				t.Fatalf("RestoreFromFile() with --overwrite error: %v", err)
			}

			// Verify old file is gone and new file exists
			cmd = exec.Command("docker", "run", "--rm",
				"-v", volumeName+":/data",
				"alpine",
				"sh", "-c", "! test -f /data/old.txt && test -f /data/new.txt && cat /data/new.txt")
			output, err := cmd.Output()
			if err != nil {
				t.Fatalf("Verification failed: %v", err)
			}

			expectedContent := "new backup data\n"
			if string(output) != expectedContent {
				t.Errorf("Content mismatch: got %q, want %q", string(output), expectedContent)
			}
		})
	}
}

func TestClearVolume(t *testing.T) {
	if !docker.IsDockerAvailable() {
		t.Skip("Docker is not available, skipping integration test")
	}

	volumeName := "test-volume-clear-xyz123"

	// Clean up
	exec.Command("docker", "volume", "rm", volumeName).Run()

	// Create volume with multiple files including hidden files
	if err := docker.CreateVolume(volumeName); err != nil {
		t.Fatalf("CreateVolume() error: %v", err)
	}
	defer exec.Command("docker", "volume", "rm", volumeName).Run()

	cmd := exec.Command("docker", "run", "--rm",
		"-v", volumeName+":/data",
		"alpine",
		"sh", "-c", "echo 'file1' > /data/file1.txt && echo 'file2' > /data/file2.txt && echo 'hidden' > /data/.hidden && mkdir /data/subdir && echo 'sub' > /data/subdir/file.txt")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create test files: %v", err)
	}

	// Clear the volume
	if err := docker.ClearVolume(volumeName); err != nil {
		t.Fatalf("ClearVolume() error: %v", err)
	}

	// Verify volume is empty
	cmd = exec.Command("docker", "run", "--rm",
		"-v", volumeName+":/data",
		"alpine",
		"sh", "-c", "ls -A /data")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to check volume: %v", err)
	}

	if len(output) > 0 {
		t.Errorf("Volume should be empty after clear, but contains: %s", string(output))
	}
}

func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		shouldErr bool
	}{
		{"valid absolute", "/tmp/backup.tar.gz", false},
		{"valid relative", "backup.tar.gz", false},
		{"valid nested", "/var/backups/data/backup.tar", false},
		{"empty", "", true},
		{"path traversal", "../etc/passwd", true},
		{"path traversal absolute", "/tmp/../etc/passwd", true},
		{"double dot in name", "backup..tar", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilePath(tt.path)
			if tt.shouldErr && err == nil {
				t.Errorf("ValidateFilePath(%q) expected error but got none", tt.path)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("ValidateFilePath(%q) unexpected error: %v", tt.path, err)
			}
		})
	}
}

func TestGetFileName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"simple file", "/path/to/file.tar.gz", "file.tar.gz"},
		{"relative path", "backup/data.tar", "data.tar"},
		{"just filename", "file.tar.gz", "file.tar.gz"},
		{"with dots", "/path/to/my.backup.tar.gz", "my.backup.tar.gz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetFileName(tt.path)
			if result != tt.expected {
				t.Errorf("GetFileName(%q) = %q; want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestGetHostDir(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantDir bool // check if result is a directory path
	}{
		{"absolute path", "/tmp/backup.tar.gz", true},
		{"relative path", "backup.tar.gz", true},
		{"nested path", "/var/backups/docker/data.tar", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetHostDir(tt.path)
			if result == "" {
				t.Errorf("GetHostDir(%q) returned empty string", tt.path)
			}
			// Result should not be the full path (should be directory only)
			if result == tt.path {
				t.Errorf("GetHostDir(%q) = %q; should be directory only", tt.path, result)
			}
		})
	}
}
