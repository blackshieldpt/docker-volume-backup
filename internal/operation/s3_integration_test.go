//go:build integration
// +build integration

package operation

import (
	"context"
	"docker-volume-backup/internal/docker"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	local "docker-volume-backup/internal/s3"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/testcontainers/testcontainers-go/modules/minio"
)

// createTestBucket creates an S3 bucket for testing
func createTestBucket(ctx context.Context, bucketName string) error {
	client, err := local.NewClient(ctx)
	if err != nil {
		return err
	}

	// Create bucket without specifying region (works with MinIO)
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	// Ignore BucketAlreadyOwnedByYou and BucketAlreadyExists errors
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "BucketAlreadyOwnedByYou") ||
			strings.Contains(errStr, "BucketAlreadyExists") {
			return nil
		}
	}
	return err
}

// objectExists checks if an S3 object exists
func objectExists(ctx context.Context, bucket, key string) (bool, error) {
	client, err := local.NewClient(ctx)
	if err != nil {
		return false, err
	}

	_, err = client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return false, nil
	}
	return true, nil
}

// TestS3BackupAndRestoreWithMinIO tests S3 functionality using MinIO testcontainer
func TestS3BackupAndRestoreWithMinIO(t *testing.T) {
	if !docker.IsDockerAvailable() {
		t.Skip("Docker is not available, skipping MinIO integration test")
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
			ctx := context.Background()

			// Start MinIO container
			minioContainer, err := minio.Run(ctx,
				"minio/minio:latest",
				minio.WithUsername("minioadmin"),
				minio.WithPassword("minioadmin"),
			)
			if err != nil {
				t.Fatalf("Failed to start MinIO container: %v", err)
			}
			defer func() {
				if err := minioContainer.Terminate(ctx); err != nil {
					t.Logf("Failed to terminate MinIO container: %v", err)
				}
			}()

			// Get MinIO connection details
			endpoint, err := minioContainer.ConnectionString(ctx)
			if err != nil {
				t.Fatalf("Failed to get MinIO endpoint: %v", err)
			}

			// AWS CLI needs http:// prefix for the endpoint
			if endpoint[:4] != "http" {
				endpoint = "http://" + endpoint
			}

			// Configure AWS CLI to use MinIO
			originalEndpoint := os.Getenv("AWS_ENDPOINT_URL_S3")
			originalAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
			originalSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

			os.Setenv("AWS_ENDPOINT_URL_S3", endpoint)
			os.Setenv("AWS_ACCESS_KEY_ID", "minioadmin")
			os.Setenv("AWS_SECRET_ACCESS_KEY", "minioadmin")

			// Create a test bucket using AWS SDK
			bucketName := "test-backup-bucket-" + tt.name
			if err := createTestBucket(ctx, bucketName); err != nil {
				t.Fatalf("Failed to create bucket: %v", err)
			}

			// Restore original environment after test
			defer func() {
				if originalEndpoint != "" {
					os.Setenv("AWS_ENDPOINT_URL_S3", originalEndpoint)
				} else {
					os.Unsetenv("AWS_ENDPOINT_URL_S3")
				}
				if originalAccessKey != "" {
					os.Setenv("AWS_ACCESS_KEY_ID", originalAccessKey)
				} else {
					os.Unsetenv("AWS_ACCESS_KEY_ID")
				}
				if originalSecretKey != "" {
					os.Setenv("AWS_SECRET_ACCESS_KEY", originalSecretKey)
				} else {
					os.Unsetenv("AWS_SECRET_ACCESS_KEY")
				}
			}()

			// Run the actual S3 backup/restore test
			volumeName := "test-volume-minio-s3-xyz123-" + tt.name
			s3Path := fmt.Sprintf("s3://%s/test-backups/minio-integration-test%s", bucketName, tt.fileExt)

			// Clean up any existing test volume
			exec.Command("docker", "volume", "rm", volumeName).Run()

			// Create test volume and write data to it
			if err := docker.CreateVolume(volumeName); err != nil {
				t.Fatalf("createVolume() error: %v", err)
			}
			defer exec.Command("docker", "volume", "rm", volumeName).Run()

			// Write test data to volume
			testData := "Hello from MinIO S3 Integration Test!"
			cmd := exec.Command("docker", "run", "--rm",
				"-v", volumeName+":/data",
				"alpine",
				"sh", "-c", "echo '"+testData+"' > /data/minio-test.txt")
			if err := cmd.Run(); err != nil {
				t.Fatalf("Failed to write test data: %v", err)
			}

			// Backup the volume to MinIO S3
			t.Logf("Backing up to MinIO S3: %s (endpoint: %s)", s3Path, endpoint)
			backupOp, err := NewBackup(volumeName, tt.compress, false)
			if err != nil {
				t.Fatalf("Failed to create backup: %v", err)
			}
			if err := backupOp.BackupToS3(s3Path); err != nil {
				t.Fatalf("BackupToS3() error: %v", err)
			}

			// Verify the file exists in MinIO using AWS SDK
			s3Key := "test-backups/minio-integration-test" + tt.fileExt
			exists, err := objectExists(ctx, bucketName, s3Key)
			if err != nil {
				t.Fatalf("Failed to check if S3 file exists: %v", err)
			}
			if !exists {
				t.Fatal("S3 file was not uploaded successfully")
			}
			t.Log("S3 file verified successfully")

			// Remove the volume
			cmd = exec.Command("docker", "volume", "rm", volumeName)
			if err := cmd.Run(); err != nil {
				t.Fatalf("Failed to remove volume: %v", err)
			}

			// Restore the volume from MinIO S3
			t.Logf("Restoring from MinIO S3: %s", s3Path)
			restoreOp, err := NewRestore(volumeName, false)
			if err != nil {
				t.Fatalf("Failed to create restore: %v", err)
			}
			if err := restoreOp.RestoreFromS3(s3Path, true); err != nil {
				t.Fatalf("RestoreFromS3() error: %v", err)
			}

			// Verify data was restored correctly
			cmd = exec.Command("docker", "run", "--rm",
				"-v", volumeName+":/data",
				"alpine",
				"cat", "/data/minio-test.txt")
			output, err := cmd.Output()
			if err != nil {
				t.Fatalf("Failed to read restored data: %v", err)
			}

			if string(output) != testData+"\n" {
				t.Errorf("Restored data mismatch: got %q, want %q", string(output), testData)
			}

			t.Log("MinIO S3 backup/restore test completed successfully")
		})
	}
}

// TestS3BackupAndRestoreWithMinIOMultipleFiles tests backing up volumes with multiple files
func TestS3BackupAndRestoreWithMinIOMultipleFiles(t *testing.T) {
	if !docker.IsDockerAvailable() {
		t.Skip("Docker is not available, skipping MinIO integration test")
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
			ctx := context.Background()

			// Start MinIO container
			minioContainer, err := minio.Run(ctx,
				"minio/minio:latest",
				minio.WithUsername("minioadmin"),
				minio.WithPassword("minioadmin"),
			)
			if err != nil {
				t.Fatalf("Failed to start MinIO container: %v", err)
			}
			defer func() {
				if err := minioContainer.Terminate(ctx); err != nil {
					t.Logf("Failed to terminate MinIO container: %v", err)
				}
			}()

			// Get MinIO connection details
			endpoint, err := minioContainer.ConnectionString(ctx)
			if err != nil {
				t.Fatalf("Failed to get MinIO endpoint: %v", err)
			}

			// AWS CLI needs http:// prefix for the endpoint
			if endpoint[:4] != "http" {
				endpoint = "http://" + endpoint
			}

			// Configure AWS SDK to use MinIO
			os.Setenv("AWS_ENDPOINT_URL_S3", endpoint)
			os.Setenv("AWS_ACCESS_KEY_ID", "minioadmin")
			os.Setenv("AWS_SECRET_ACCESS_KEY", "minioadmin")
			defer func() {
				os.Unsetenv("AWS_ENDPOINT_URL_S3")
				os.Unsetenv("AWS_ACCESS_KEY_ID")
				os.Unsetenv("AWS_SECRET_ACCESS_KEY")
			}()

			// Create a test bucket using AWS SDK
			bucketName := "test-multifile-bucket-" + tt.name
			if err := createTestBucket(ctx, bucketName); err != nil {
				t.Fatalf("Failed to create bucket: %v", err)
			}

			volumeName := "test-volume-multifile-xyz123-" + tt.name
			s3Path := fmt.Sprintf("s3://%s/multifile-backup%s", bucketName, tt.fileExt)

			// Clean up
			exec.Command("docker", "volume", "rm", volumeName).Run()

			// Create volume with multiple files
			if err := docker.CreateVolume(volumeName); err != nil {
				t.Fatalf("createVolume() error: %v", err)
			}
			defer exec.Command("docker", "volume", "rm", volumeName).Run()

			// Write multiple files with different content
			createFilesScript := `
		mkdir -p /data/subdir
		echo 'File 1 content' > /data/file1.txt
		echo 'File 2 content' > /data/file2.txt
		echo 'Subdir file content' > /data/subdir/file3.txt
		echo '{"key": "value"}' > /data/config.json
	`
			cmd := exec.Command("docker", "run", "--rm",
				"-v", volumeName+":/data",
				"alpine",
				"sh", "-c", createFilesScript)
			if err := cmd.Run(); err != nil {
				t.Fatalf("Failed to create test files: %v", err)
			}

			// Backup
			backupOp, err := NewBackup(volumeName, tt.compress, false)
			if err != nil {
				t.Fatalf("Failed to create backup: %v", err)
			}
			if err := backupOp.BackupToS3(s3Path); err != nil {
				t.Fatalf("backupToS3() error: %v", err)
			}

			// Remove volume
			exec.Command("docker", "volume", "rm", volumeName).Run()

			// Restore
			restoreOp, err := NewRestore(volumeName, false)
			if err != nil {
				t.Fatalf("Failed to create restore: %v", err)
			}
			if err := restoreOp.RestoreFromS3(s3Path, true); err != nil {
				t.Fatalf("restoreFromS3() error: %v", err)
			}

			// Verify all files were restored
			verifyScript := `
		test -f /data/file1.txt && \
		test -f /data/file2.txt && \
		test -f /data/subdir/file3.txt && \
		test -f /data/config.json && \
		echo "All files exist"
	`
			cmd = exec.Command("docker", "run", "--rm",
				"-v", volumeName+":/data",
				"alpine",
				"sh", "-c", verifyScript)
			output, err := cmd.Output()
			if err != nil {
				t.Fatalf("File verification failed: %v", err)
			}

			if string(output) != "All files exist\n" {
				t.Error("Not all files were restored correctly")
			}

			t.Log("MinIO S3 multiple files backup/restore test completed successfully")
		})
	}
}

// TestS3ValidationWithMinIO tests that S3 path validation works with MinIO
func TestS3ValidationWithMinIO(t *testing.T) {
	if !docker.IsDockerAvailable() {
		t.Skip("Docker is not available, skipping MinIO validation test")
	}

	ctx := context.Background()

	minioContainer, err := minio.Run(ctx,
		"minio/minio:latest",
		minio.WithUsername("minioadmin"),
		minio.WithPassword("minioadmin"),
	)
	if err != nil {
		t.Fatalf("Failed to start MinIO container: %v", err)
	}
	defer minioContainer.Terminate(ctx)

	endpoint, err := minioContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("Failed to get MinIO endpoint: %v", err)
	}

	// AWS CLI needs http:// prefix for the endpoint
	if endpoint[:4] != "http" {
		endpoint = "http://" + endpoint
	}

	// Configure AWS SDK to use MinIO
	os.Setenv("AWS_ENDPOINT_URL_S3", endpoint)
	os.Setenv("AWS_ACCESS_KEY_ID", "minioadmin")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "minioadmin")
	defer func() {
		os.Unsetenv("AWS_ENDPOINT_URL_S3")
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	}()

	// Test invalid S3 paths
	invalidPaths := []string{
		"s3://",
		"s3://bucket-only",
		"not-s3://bucket/key",
		"",
	}

	volumeName := "test-validation-volume"
	docker.CreateVolume(volumeName)
	defer exec.Command("docker", "volume", "rm", volumeName).Run()

	for _, invalidPath := range invalidPaths {
		t.Run("invalid_path_"+invalidPath, func(t *testing.T) {
			backupOp, err := NewBackup(volumeName, "", false)
			if err != nil {
				t.Fatalf("Failed to create backup: %v", err)
			}
			err = backupOp.BackupToS3(invalidPath)
			if err == nil {
				t.Errorf("backupToS3() with invalid path %q should have failed but didn't", invalidPath)
			}
		})
	}

	t.Log("S3 validation with MinIO test completed successfully")
}
