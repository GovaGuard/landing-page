# Docker Deployment Guide

## Overview

GovaGuard uses a **distroless** Docker image for maximum security and minimal attack surface. The image is automatically built and published to GitHub Container Registry when version tags are pushed.

## Quick Start

### Pull and Run

```bash
# Pull the latest image
docker pull ghcr.io/[YOUR-GITHUB-USERNAME]/govaguard:latest

# Run the container
docker run -d -p 8080:8080 --name govaguard ghcr.io/[YOUR-GITHUB-USERNAME]/govaguard:latest

# Access the application
open http://localhost:8080
```

## Building Locally

### Prerequisites
- Docker installed and running
- Go 1.22+ (for local development)

### Build Commands

```bash
# Build the Docker image
docker build -t govaguard:local .

# Run locally built image
docker run -p 8080:8080 govaguard:local

# Build for multiple platforms
docker buildx build --platform linux/amd64,linux/arm64 -t govaguard:multi .
```

## Automated GitHub Actions Build

### How to Trigger a Build

1. **Create and push a version tag:**
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **GitHub Actions will automatically:**
   - Build the Docker image
   - Tag it appropriately (semver + latest)
   - Push to GitHub Container Registry
   - Build for both amd64 and arm64

### Image Tags

For tag `v1.2.3`, the following tags are created:
- `ghcr.io/[owner]/govaguard:1.2.3` (full version)
- `ghcr.io/[owner]/govaguard:1.2` (major.minor)
- `ghcr.io/[owner]/govaguard:1` (major)
- `ghcr.io/[owner]/govaguard:latest` (latest stable)
- `ghcr.io/[owner]/govaguard:main-[sha]` (commit sha)

## Docker Image Details

### Base Image
- **Base:** `gcr.io/distroless/static-debian12:nonroot`
- **User:** nonroot:nonroot (UID/GID: 65532)
- **Size:** ~10-12 MB (minimal footprint)

### Security Features
- ✅ No shell or package manager (distroless)
- ✅ Non-root user
- ✅ Minimal dependencies
- ✅ Static binary (no dynamic linking)
- ✅ Multi-stage build

### Exposed Ports
- `8080` - HTTP server

### Environment Variables
Currently no environment variables are required. The application uses embedded templates and static files.

## Production Deployment

### Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  govaguard:
    image: ghcr.io/[owner]/govaguard:latest
    ports:
      - "8080:8080"
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "wget --no-verbose --tries=1 --spider http://localhost:8080 || exit 1"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

Run:
```bash
docker-compose up -d
```

### Kubernetes Deployment

Create `deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: govaguard
spec:
  replicas: 3
  selector:
    matchLabels:
      app: govaguard
  template:
    metadata:
      labels:
        app: govaguard
    spec:
      containers:
      - name: govaguard
        image: ghcr.io/[owner]/govaguard:latest
        ports:
        - containerPort: 8080
        resources:
          requests:
            memory: "32Mi"
            cpu: "50m"
          limits:
            memory: "64Mi"
            cpu: "100m"
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          runAsUser: 65532
          capabilities:
            drop:
              - ALL
---
apiVersion: v1
kind: Service
metadata:
  name: govaguard
spec:
  selector:
    app: govaguard
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

Apply:
```bash
kubectl apply -f deployment.yaml
```

## Authentication for Private Images

If your repository is private, authenticate before pulling:

```bash
# Create a GitHub Personal Access Token with read:packages scope
# Then login to ghcr.io

echo $GITHUB_TOKEN | docker login ghcr.io -u YOUR-USERNAME --password-stdin
```

For Kubernetes, create an image pull secret:

```bash
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=YOUR-USERNAME \
  --docker-password=$GITHUB_TOKEN \
  --docker-email=YOUR-EMAIL
```

Add to deployment:
```yaml
spec:
  imagePullSecrets:
  - name: ghcr-secret
```

## Health Checks

The application doesn't currently expose a health endpoint, but you can check the homepage:

```bash
# Simple health check
curl -f http://localhost:8080 || exit 1

# In Docker
docker run --rm --network container:govaguard curlimages/curl:latest \
  curl -f http://localhost:8080
```

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker logs govaguard

# Run in foreground to see output
docker run -p 8080:8080 ghcr.io/[owner]/govaguard:latest
```

### Port Already in Use

```bash
# Find process using port 8080
lsof -i :8080

# Kill the process or use a different port
docker run -p 9090:8080 ghcr.io/[owner]/govaguard:latest
```

### Image Pull Errors

```bash
# Check if you're authenticated
docker login ghcr.io

# Verify image exists
docker manifest inspect ghcr.io/[owner]/govaguard:latest
```

## Development Workflow

```bash
# 1. Make changes to code
# 2. Test locally
go run main.go

# 3. Build and test Docker image locally
docker build -t govaguard:dev .
docker run -p 8080:8080 govaguard:dev

# 4. Create and push tag
git tag v1.0.0
git push origin v1.0.0

# 5. GitHub Actions builds and publishes
# 6. Deploy to production
```

## Image Size Comparison

| Base Image | Size | Security |
|------------|------|----------|
| Ubuntu | ~70 MB | Low (many packages) |
| Alpine | ~15 MB | Medium (shell, apk) |
| **Distroless** | **~10 MB** | **High (minimal)** |

## Best Practices

1. **Always use specific version tags in production** (not `latest`)
2. **Enable security scanning** in GitHub repository settings
3. **Use non-root user** (already configured)
4. **Set resource limits** in Kubernetes/Docker Compose
5. **Implement health checks** for orchestration
6. **Use HTTPS** with reverse proxy (nginx, Caddy, Traefik)

## Next Steps

- [ ] Add health check endpoint to the application
- [ ] Implement graceful shutdown
- [ ] Add configuration via environment variables
- [ ] Set up monitoring and logging
- [ ] Configure TLS/HTTPS
