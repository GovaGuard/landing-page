# GitHub Actions - Docker Build & Publish

## Overview

This workflow automatically builds and publishes a Docker image to GitHub Container Registry (ghcr.io) when version tags are pushed.

## Trigger

The workflow triggers on tag pushes matching the pattern `v*.*.*` (e.g., `v1.0.0`, `v2.1.3`).

## How to Release

1. **Create a new tag:**
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **The workflow will automatically:**
   - Build the Docker image using distroless base
   - Tag it with multiple formats (version, major.minor, major, sha, latest)
   - Push to GitHub Container Registry
   - Build for multiple platforms (linux/amd64, linux/arm64)

## Image Tags Generated

For tag `v1.2.3`, the following tags will be created:
- `ghcr.io/[owner]/govaguard:1.2.3`
- `ghcr.io/[owner]/govaguard:1.2`
- `ghcr.io/[owner]/govaguard:1`
- `ghcr.io/[owner]/govaguard:latest` (if on default branch)
- `ghcr.io/[owner]/govaguard:main-[sha]` (commit sha)

## Accessing the Image

### Public Repository
If your repository is public, the image will be publicly accessible:

```bash
docker pull ghcr.io/[owner]/govaguard:latest
```

### Private Repository
For private repositories, authenticate first:

```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
docker pull ghcr.io/[owner]/govaguard:latest
```

## Running the Container

```bash
# Run the container
docker run -d -p 8080:8080 ghcr.io/[owner]/govaguard:latest

# Access the application
open http://localhost:8080
```

## Permissions

The workflow requires the following permissions (already configured):
- `contents: read` - to checkout the repository
- `packages: write` - to push to GitHub Container Registry
- `id-token: write` - for OIDC authentication

## Security Features

- Uses **distroless** base image (minimal attack surface)
- Runs as **non-root user** (nonroot:nonroot)
- **Multi-stage build** (smaller final image)
- **Multi-platform** builds (amd64 and arm64)
- **Build cache** optimization using GitHub Actions cache

## Troubleshooting

### View Workflow Runs
- Go to your repository → Actions tab → "Build and Push Docker Image"

### View Published Images
- Go to your repository → Packages (or `ghcr.io/[owner]/govaguard`)

### Common Issues

1. **Package visibility**: If the image is private, ensure you have proper authentication
2. **Tag format**: Ensure tags follow semver format (v1.0.0, not 1.0.0)
3. **Permissions**: Verify workflow permissions in repository settings

## Manual Build (Local)

```bash
# Build locally
docker build -t govaguard:local .

# Run locally built image
docker run -p 8080:8080 govaguard:local
```

## Image Size

Using distroless significantly reduces image size:
- Traditional alpine-based: ~15-20 MB
- Distroless: ~10-12 MB
- Contains only the binary and runtime dependencies
