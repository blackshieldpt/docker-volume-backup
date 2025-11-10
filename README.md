# Docker Volume Backup

A command-line tool to backup and restore Docker volumes to local files or AWS S3.

## Features

- **Local Backup/Restore**: Backup Docker volumes to local filesystem
- **S3 Backup/Restore**: Backup Docker volumes to AWS S3 buckets
- **Multiple Compression Formats**: Support for gzip, zstd, or no compression
- **Progress Tracking**: Optional progress indicators during operations
- **Automatic Volume Creation**: Automatically creates volumes during restore if they don't exist
- **Input Validation**: Security-hardened with input validation to prevent injection attacks
- **Cross-Platform**: Builds for Linux, macOS, and Windows

## Installation

### Quick Install (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/blackshieldpt/docker-volume-backup/master/install.sh | bash
```

The install script:
- Detects your OS and architecture automatically
- Downloads the appropriate binary from GitHub releases
- Verifies integrity using SHA256 checksums
- Installs to `/usr/local/bin/`

### Pre-built Binaries

Download pre-built binaries from the [releases page](https://github.com/blackshieldpt/docker-volume-backup/releases).

Available platforms:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

### Debian/Ubuntu Installation

```bash
# Download the binary
wget https://github.com/blackshieldpt/docker-volume-backup/releases/latest/download/docker-volume-backup-linux-amd64

# Verify checksum (optional but recommended)
wget https://github.com/blackshieldpt/docker-volume-backup/releases/latest/download/docker-volume-backup-linux-amd64.sha256
sha256sum -c docker-volume-backup-linux-amd64.sha256

# Install to system
sudo install -m 755 docker-volume-backup-linux-amd64 /usr/local/bin/docker-volume-backup

# Verify installation
docker-volume-backup --help
```

For ARM64 systems (e.g., Raspberry Pi):
```bash
wget https://github.com/blackshieldpt/docker-volume-backup/releases/latest/download/docker-volume-backup-linux-arm64
sudo install -m 755 docker-volume-backup-linux-arm64 /usr/local/bin/docker-volume-backup
```

### Red Hat/CentOS/Fedora Installation

```bash
# Download the binary
curl -LO https://github.com/blackshieldpt/docker-volume-backup/releases/latest/download/docker-volume-backup-linux-amd64

# Verify checksum (optional but recommended)
curl -LO https://github.com/blackshieldpt/docker-volume-backup/releases/latest/download/docker-volume-backup-linux-amd64.sha256
sha256sum -c docker-volume-backup-linux-amd64.sha256

# Install to system
sudo install -m 755 docker-volume-backup-linux-amd64 /usr/local/bin/docker-volume-backup

# Verify installation
docker-volume-backup --help
```

For ARM64 systems:
```bash
curl -LO https://github.com/blackshieldpt/docker-volume-backup/releases/latest/download/docker-volume-backup-linux-arm64
sudo install -m 755 docker-volume-backup-linux-arm64 /usr/local/bin/docker-volume-backup
```

### From Source

```bash
git clone https://github.com/blackshieldpt/docker-volume-backup.git
cd docker-volume-backup
make build
sudo make install
```

**Requirements:**
- Go 1.24 or later
- Git

## Usage

### Basic Syntax

```bash
docker-volume-backup backup [--progress] [--compress gz|zstd|none] <volume> <dest>
docker-volume-backup restore [--progress] [--overwrite] <src> <volume>
```

**Flags:**
- `--progress` - Show progress bar during backup/restore
- `--compress <type>` - Compression type: `none`|`gz`|`zstd` (default: `gz`) [backup only]
- `--overwrite` - Clear existing volume before restore [restore only]

### Local Backup Examples

```bash
# Backup a volume to local file with gzip compression (default)
docker-volume-backup backup my-volume /backups/my-volume.tar.gz

# Backup with progress indicator
docker-volume-backup backup --progress my-volume /backups/my-volume.tar.gz

# Backup with zstd compression (better compression ratio)
docker-volume-backup backup --compress zstd my-volume /backups/my-volume.tar.zst

# Backup without compression
docker-volume-backup backup --compress none my-volume /backups/my-volume.tar
```

### Local Restore Examples

```bash
# Restore to a new volume (creates volume if it doesn't exist)
docker-volume-backup restore /backups/my-volume.tar.gz my-new-volume

# Restore with progress indicator
docker-volume-backup restore --progress /backups/my-volume.tar.gz my-new-volume

# Restore to an existing volume (ERROR - conservative by default)
docker-volume-backup restore /backups/my-volume.tar.gz existing-volume
# Error: volume 'existing-volume' already exists. Use --overwrite flag...

# Restore with --overwrite (clears existing data first)
docker-volume-backup restore --overwrite /backups/my-volume.tar.gz existing-volume
```

### S3 Backup Examples

```bash
# Backup to S3
docker-volume-backup backup my-volume s3://my-bucket/backups/my-volume.tar.gz

# Backup to S3 with progress
docker-volume-backup backup --progress my-volume s3://my-bucket/backups/my-volume.tar.gz
```

### S3 Restore Examples

```bash
# Restore from S3 to new volume (creates volume if it doesn't exist)
docker-volume-backup restore s3://my-bucket/backups/my-volume.tar.gz my-new-volume

# Restore from S3 with progress
docker-volume-backup restore --progress s3://my-bucket/backups/my-volume.tar.gz my-new-volume

# Restore from S3 with --overwrite (clears existing volume first)
docker-volume-backup restore --overwrite s3://my-bucket/backups/my-volume.tar.gz existing-volume
```

## Configuration

### S3 Configuration

For S3 operations, configure AWS credentials using environment variables or AWS credentials file:

**Option 1: Environment Variables**
```bash
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
export AWS_REGION=us-east-1  # Optional, defaults to us-east-1
```

**Option 2: AWS Credentials File**
```bash
# Create ~/.aws/credentials
mkdir -p ~/.aws
cat > ~/.aws/credentials << EOF
[default]
aws_access_key_id = your-access-key
aws_secret_access_key = your-secret-key
EOF

# Create ~/.aws/config
cat > ~/.aws/config << EOF
[default]
region = us-east-1
EOF
```

**Option 3: S3-Compatible Services (MinIO, DigitalOcean Spaces, etc.)**
```bash
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
export AWS_ENDPOINT_URL_S3=https://your-s3-compatible-endpoint.com
export AWS_REGION=us-east-1
```

## Requirements

### Runtime Requirements

- **Docker daemon** (required) - must be running
- **Docker CLI** (required) - for volume operations
- **AWS credentials** (optional) - only required for S3 operations
  - Uses AWS SDK for Go v2 internally
  - No AWS CLI installation needed

### Build Requirements

- **Go 1.24 or later** - for building from source
- **Git** - for cloning repository
- **Docker** - for running integration tests

## How It Works

The tool uses native Go libraries and Docker for efficient backup/restore operations:

1. **Backup**:
   - Creates a temporary Alpine container with the Docker volume mounted at `/data`
   - Uses `docker cp` to stream the volume contents
   - Compresses data using Go native libraries (gzip/zstd) while streaming
   - Writes compressed tar archive to destination (local file or S3)

2. **Restore**:
   - Reads and decompresses the backup archive using Go native libraries
   - Creates a temporary Alpine container with the target volume mounted
   - Uses `docker cp` to stream decompressed data into the volume
   - Cleans up temporary container

3. **S3 Operations**:
   - Uses AWS SDK for Go v2 for direct S3 upload/download
   - No AWS CLI dependency required
   - Supports S3-compatible services (MinIO, etc.)
   - Creates temporary local files for S3 operations, then cleans up

**Performance Benefits:**
- No shell command overhead for compression
- Streaming architecture minimizes memory usage
- Static binary with no external dependencies (except Docker)
- Efficient for large volumes

### Restore Behavior

The tool follows a **conservative approach** when restoring to ensure data safety:

**To a new volume (doesn't exist):**
- Volume is created automatically
- Backup data is restored
- Ready to use

**To an existing volume (without `--overwrite`):**
- Operation fails with error
- Existing data is preserved
- User must explicitly use `--overwrite` flag or delete volume manually

**To an existing volume (with `--overwrite`):**
- Existing volume contents are **completely cleared** first
- Backup data is restored to the now-empty volume
- Clean state - no data pollution from previous contents

**Why conservative by default?**
- Prevents accidental data loss
- Avoids data pollution (mixing old and new data)
- Makes restore intent explicit
- Safer for production environments

**Example workflow:**
```bash
# Attempt restore to existing volume
$ docker-volume-backup restore backup.tar.gz my-volume
ERROR: volume 'my-volume' already exists. Use --overwrite flag...

# Two options:
# Option 1: Delete volume manually
$ docker volume rm my-volume
$ docker-volume-backup restore backup.tar.gz my-volume
Success

# Option 2: Use --overwrite flag
$ docker-volume-backup restore --overwrite backup.tar.gz my-volume
Clearing volume 'my-volume'
Success
```

## Compression Options

- `gz` (default): gzip compression - good balance of speed and compression
- `zstd`: Zstandard compression - better compression ratio, slightly slower
- `none`: No compression - fastest, largest file size

## Development

### Build Commands

```bash
# Build for current platform
make build

# Run tests
make test

# Run integration tests (requires Docker)
go test -tags=integration ./...

# Clean build artifacts
make clean

# Build for all platforms
make release
```

### Project Structure

```
docker-volume-backup/
├── cmd/
│   ├── main.go                 # Main CLI application
│   ├── s3.go                   # S3 integration
│   ├── main_test.go            # Unit tests
│   ├── integration_test.go     # Docker volume integration tests
│   ├── s3_test.go              # S3 unit tests
│   └── s3_integration_test.go  # MinIO-based S3 integration tests
├── Dockerfile                  # Container image
├── Makefile                    # Build automation
├── go.mod                      # Go module dependencies
├── install.sh                  # Installation script
├── CHANGELOG.md                # Version history
├── .gitignore                  # Git exclusions
└── README.md                   # This file
```

### Running Tests

```bash
# Unit tests
make test

# Integration tests (requires Docker)
go test -tags=integration ./... -v

# All tests with coverage
go test -cover ./...
```

**Test Coverage:**
- Basic S3 backup and restore
- Multiple files and directories
- S3 path validation
- Error handling
- Temporary file cleanup
- Gzip and zstd compression

## CI/CD

The project uses GitHub Actions for automated testing and releases:

**Continuous Integration (CI):**
- Runs on every push/PR to master
- Executes `go vet` for code quality
- Runs unit tests and integration tests
- Builds and verifies static binary
- Tests cross-platform builds

**Release Automation:**
- Triggers on version tags (`v*`)
- Builds binaries for 5 platforms:
  - Linux (amd64, arm64)
  - macOS (amd64, arm64)
  - Windows (amd64)
- Generates SHA256 checksums for all binaries
- Optional GPG signing (if `GPG_PRIVATE_KEY` secret is configured)
- Creates GitHub releases with all artifacts automatically
- Includes auto-generated release notes

## Common Use Cases

### Scheduled Backups

```bash
# Cron job for daily backups
0 2 * * * /usr/local/bin/docker-volume-backup backup my-volume /backups/my-volume-$(date +\%Y\%m\%d).tar.gz
```

### Backup Before Upgrades

```bash
# Backup all volumes before system upgrade
for volume in $(docker volume ls -q); do
  docker-volume-backup backup $volume /backups/$volume-$(date +%Y%m%d).tar.gz
done
```

### Migrate Between Hosts

```bash
# On source host
docker-volume-backup backup my-volume s3://my-bucket/migration/my-volume.tar.gz

# On destination host
docker-volume-backup restore s3://my-bucket/migration/my-volume.tar.gz my-volume
```

## Troubleshooting

### "volume does not exist" error

The volume you're trying to backup doesn't exist. List volumes with:
```bash
docker volume ls
```

### "Docker daemon is not running"

Ensure Docker is running:
```bash
docker version
```

### S3 upload/download failures

Check AWS credentials are properly configured:
```bash
# Verify credentials file exists
cat ~/.aws/credentials

# Or check environment variables
echo $AWS_ACCESS_KEY_ID
echo $AWS_SECRET_ACCESS_KEY
echo $AWS_REGION

# Test with a simple backup to ensure credentials work
docker-volume-backup backup test-volume s3://your-bucket/test.tar.gz
```

### Permission denied

Ensure you have permission to run Docker commands and write to backup directory:
```bash
# Add user to docker group
sudo usermod -aG docker $USER
```

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

Copyright 2025 BlackShield LDA
