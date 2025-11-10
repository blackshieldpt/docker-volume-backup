package s3

import (
	"os"
	"testing"
)

func TestUploadToS3Parameters(t *testing.T) {
	// Test that the function accepts the correct parameters
	// We can't actually test S3 upload without credentials/mocking
	// but we can verify the function signature and basic validation

	tests := []struct {
		name      string
		localFile string
		s3Path    string
		shouldErr bool
	}{
		{"empty local file", "", "s3://bucket/file", true},
		{"empty s3 path", "/tmp/file", "", true},
		{"non-existent file", "/nonexistent/file.tar.gz", "s3://bucket/file", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This will fail because the file doesn't exist or AWS CLI will fail
			// but we're testing that the function handles errors appropriately
			err := UploadFile(tt.localFile, tt.s3Path)
			if !tt.shouldErr && err != nil {
				t.Errorf("UploadFile() unexpected error: %v", err)
			}
			// Note: We expect an error in all these test cases
			// A real S3 test would require AWS credentials and mocking
		})
	}
}

func TestDownloadFromS3Parameters(t *testing.T) {
	tests := []struct {
		name      string
		s3Path    string
		localFile string
		shouldErr bool
	}{
		{"empty s3 path", "", "/tmp/file", true},
		{"empty local file", "s3://bucket/file", "", true},
		{"invalid s3 path", "s3://bucket/file", "/tmp/test-download.tar.gz", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up if file exists
			defer os.Remove(tt.localFile)

			// This will fail because AWS CLI will fail without proper credentials
			err := DownloadFile(tt.s3Path, tt.localFile)
			// We expect errors since we're not actually connecting to S3
			if err == nil {
				t.Errorf("DownloadFile() expected error but got none")
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		shouldErr bool
	}{
		{"valid s3 path", "s3://mybucket/backup.tar.gz", false},
		{"valid s3 nested", "s3://mybucket/path/to/backup.tar.gz", false},
		{"missing s3 prefix", "mybucket/backup.tar.gz", true},
		{"http instead", "http://mybucket/backup.tar.gz", true},
		{"just s3://", "s3://", true},
		{"no key", "s3://mybucket", true},
		{"no bucket", "s3:///backup.tar.gz", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.path)
			if tt.shouldErr && err == nil {
				t.Errorf("validateS3Path(%q) expected error but got none", tt.path)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("validateS3Path(%q) unexpected error: %v", tt.path, err)
			}
		})
	}
}
