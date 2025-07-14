# Release Process

This document describes the automated release process for the dreampipe project.

## Overview

The project uses GitHub Actions and GoReleaser to automate the creation of release artifacts for multiple operating systems and architectures. Releases are triggered by pushing tags to the main branch.

## Prerequisites

To set up the development environment, you can use the automated setup:

```bash
# Install Go only (supports Ubuntu/Debian, macOS, FreeBSD)
make setup

# Verify all dependencies are installed
make check-deps
```

The `make setup` command installs **Go** automatically but **does not install GoReleaser**. GoReleaser is only required for release builds, not for regular development.

### Manual Installation

If the automated setup doesn't work for your system, install the following manually:

- **Go** 1.21 or later - Install from [go.dev](https://go.dev/)
- **GoReleaser** - Install from [goreleaser.com/install/](https://goreleaser.com/install/) (only needed for releases)
- **Git**

### Installing GoReleaser

GoReleaser is optional for development but required for creating releases. Install it when you need to create releases:

```bash
# Install GoReleaser (various options available)
# See: https://goreleaser.com/install/
go install github.com/goreleaser/goreleaser@latest
```

## Supported Platforms

The automated release process builds binaries for the following platforms:

- **FreeBSD**: amd64, arm64
- **Linux**: amd64, arm64  
- **macOS**: amd64, arm64

## Creating a Release

### 1. Prepare the Release

Before creating a release, ensure:
- All changes are committed and pushed to the main branch
- Tests pass
- Documentation is up to date
- The version is ready for release

### 2. Test the Release Locally (Optional)

You can test the release process locally using:

```bash
# Test the release process
make test-release
```

This will create a snapshot release in the `dist/` directory without publishing anything.

### 3. Create and Push a Tag

Create a new git tag following semantic versioning (e.g., v1.0.0):

#### Using the Makefile (Recommended)

```bash
# Create and push a new release tag
make release VERSION=v1.0.0
```

This command will:
- Check that you're on the main branch
- Verify the working directory is clean
- Pull the latest changes
- Create the tag with a release message
- Push the tag to GitHub

#### Manual Process

```bash
# Create a new tag
git tag -a v1.0.0 -m "Release version 1.0.0"

# Push the tag to GitHub
git push origin v1.0.0
```

### 4. Monitor the Release

After pushing the tag:

1. Navigate to the "Actions" tab in your GitHub repository
2. You'll see the "Go Release" workflow running
3. Once it completes successfully, a new release will be created on the "Releases" page
4. The release will include:
   - Compiled binaries for all supported platforms
   - Checksums file for verification
   - Automatic changelog generation
   - Release notes

## Release Assets

Each release includes:

- **Binaries**: `dreampipe-{version}-{os}-{arch}.tar.gz`
- **Checksums**: `checksums.txt` with SHA256 hashes
- **Documentation**: README, LICENSE, SECURITY.md, config.toml.sample
- **Examples**: All example scripts from the `examples/` directory

## Version Management

The version is automatically:
- Extracted from the git tag during the release process
- Embedded in the binary using build-time ldflags
- Displayed when running `dreampipe --version`

## Local Development

For local development, you can use these Makefile targets:

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run the application with arguments
make run ARGS="your instruction here"

# Check current version
make version

# Clean build artifacts
make clean

# Install system-wide to /usr/local/bin
make install

# Install to ~/bin (user-local)
make installuser

# Install example scripts to ~/bin
make install-examples

# Remove from /usr/local/bin
make uninstall

# Remove from ~/bin
make uninstalluser

# Remove example scripts from ~/bin
make uninstall-examples FORCE=true  # Force removal of modified examples
```

## Troubleshooting

If the release process fails:

1. **Check GoReleaser installation**: Ensure GoReleaser is installed: `make check-deps`
2. **Review GitHub Actions logs**: Check for detailed error messages in the Actions tab
3. **Verify configuration**: Ensure the `.goreleaser.yml` configuration is valid
4. **Test locally**: Run `make test-release` to test the release process locally
5. **Check required files**: Ensure all required files are present in the repository

### Common Issues

- **GoReleaser not found**: Install GoReleaser from [goreleaser.com/install/](https://goreleaser.com/install/)
- **Go not in PATH**: After installing Go, restart your terminal or run `source ~/.profile`
- **Permission issues**: The automated setup may require `sudo` for system-wide installation
- **Homebrew not installed (macOS)**: Install Homebrew from [brew.sh](https://brew.sh/) then run `make setup` again
- **Broken Go installation**: If `/usr/local/go` exists but `go` command fails, manually remove and reinstall

## Configuration Files

- `.github/workflows/release.yml`: GitHub Actions workflow
- `.goreleaser.yml`: GoReleaser configuration
- `Makefile`: Build and release targets

## Setup Behavior and Safety

The `make setup` command is designed to be safe and non-destructive:

- **Never deletes existing installations** - If Go is already installed, it reports the current version
- **Detects broken installations** - If `/usr/local/go` exists but `go` command is unavailable, it provides clear guidance
- **Provides upgrade guidance** - Points users to official documentation for upgrades
- **Platform-specific installation** - Uses appropriate package managers (apt, brew, pkg) where available
- **Fallback to manual installation** - For unsupported platforms, directs users to [go.dev](https://go.dev/)

### Supported Platforms for Automated Setup

- **Ubuntu/Debian Linux** - Uses `apt-get` to install dependencies, downloads Go from official source
- **macOS** - Uses Homebrew to install Go and dependencies (requires Homebrew to be pre-installed from [brew.sh](https://brew.sh/))  
- **FreeBSD** - Uses `pkg` to install Go and dependencies
- **Other platforms** - Provides manual installation instructions
