package s3

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// parseS3Path parses an S3 path like "s3://bucket/key" into bucket and key
func parseS3Path(s3Path string) (bucket, key string, err error) {
	if !strings.HasPrefix(s3Path, "s3://") {
		return "", "", fmt.Errorf("invalid S3 path format, must start with s3://")
	}

	path := strings.TrimPrefix(s3Path, "s3://")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid S3 path format, expected s3://bucket/key")
	}

	return parts[0], parts[1], nil
}

// NewClient creates an AWS S3 client with default configuration
func NewClient(ctx context.Context) (*s3.Client, error) {
	// Default to us-east-1 if no region is set (required for MinIO and S3-compatible services)
	if os.Getenv("AWS_REGION") == "" && os.Getenv("AWS_DEFAULT_REGION") == "" {
		os.Setenv("AWS_REGION", "us-east-1")
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Use path-style addressing for S3-compatible services like MinIO
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	}), nil
}

func UploadFile(localFile, s3Path string) error {
	ctx := context.Background()

	// Parse S3 path
	bucket, key, err := parseS3Path(s3Path)
	if err != nil {
		return err
	}

	// Open local file
	file, err := os.Open(localFile)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", localFile, err)
	}
	defer file.Close()

	// Create S3 client
	client, err := NewClient(ctx)
	if err != nil {
		return err
	}

	// Create uploader
	uploader := manager.NewUploader(client)

	// Upload file
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

// DownloadFile downloads a file from the specified S3 path to the provided local file path.
// s3Path: The S3 path (e.g., "s3://bucket/key") of the file to download.
// localFile: The local file path where the downloaded file will be stored.
// Returns an error if the download fails, or if the S3 path is invalid or inaccessible.
func DownloadFile(s3Path, localFile string) error {
	ctx := context.Background()

	// Parse S3 path
	bucket, key, err := parseS3Path(s3Path)
	if err != nil {
		return err
	}

	// Create local file
	file, err := os.Create(localFile)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", localFile, err)
	}
	defer file.Close()

	// Create S3 client
	client, err := NewClient(ctx)
	if err != nil {
		return err
	}

	// Create downloader
	downloader := manager.NewDownloader(client)

	// Download file
	_, err = downloader.Download(ctx, file, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to download from S3: %w", err)
	}

	return nil
}

func ValidatePath(path string) error {
	if !strings.HasPrefix(path, "s3://") {
		return fmt.Errorf("S3 path must start with s3://")
	}
	// Basic S3 path validation: s3://bucket/key
	parts := strings.SplitN(strings.TrimPrefix(path, "s3://"), "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid S3 path format: expected s3://bucket/key")
	}
	return nil
}
