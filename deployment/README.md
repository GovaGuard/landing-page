# GovaGuard Deployment

Production-ready deployment with Envoy proxy. Just run `docker-compose up` and access on port 80.

## Quick Start

1. **Configure SMTP credentials** (edit `docker-compose.yml`):
   ```yaml
   environment:
     - SMTP_HOST=smtp.gmail.com
     - SMTP_PORT=587
     - SMTP_USER=your-email@example.com
     - SMTP_PASSWORD=your-password
     - TO_EMAIL=contact@govaguard.com
   ```

2. **Start the services**:
   ```bash
   cd deployment
   docker-compose up
   ```

That's it! Access at:
- **Application**: http://localhost (or use `curl -H "Host: govaguard.com" http://localhost/`)
- **Envoy Admin**: http://localhost:9901

See [TESTING.md](TESTING.md) for testing with the govaguard.com domain before DNS is configured.

## What You Get

- ✅ **Envoy Proxy** on port 80/443
- ✅ **Security Headers** (X-Frame-Options, HSTS, CSP, etc.)
- ✅ **Gzip Compression** for HTML, CSS, JS
- ✅ **Health Checks** automatic
- ✅ **Circuit Breaking** prevents cascade failures
- ✅ **JSON Access Logs**

## Configuration

### Port Configuration

Edit `.env` to change ports:
```bash
HTTP_PORT=80
HTTPS_PORT=443
ADMIN_PORT=9901
```

### Contact Form Email Configuration

Configure SMTP settings directly in `docker-compose.yml` to enable email delivery. Edit the `environment` section of the `govaguard` service:

```yaml
environment:
  - SMTP_HOST=smtp.gmail.com          # Your SMTP server
  - SMTP_PORT=587                      # 587 for TLS, 465 for SSL
  - SMTP_USER=your-email@example.com  # SMTP username
  - SMTP_PASSWORD=your-password        # SMTP password or app-specific password
  - TO_EMAIL=contact@govaguard.com    # Where to receive form submissions
```

**Note**: If SMTP is not configured (values left empty), form submissions will be logged to the container console instead.

**For Gmail**: Use [App-Specific Passwords](https://support.google.com/accounts/answer/185833) instead of your main password.

**For SendGrid/Mailgun**: Use your API key as the SMTP password.

## Production Use

Run in background:
```bash
docker-compose up -d
```

View logs:
```bash
docker-compose logs -f
```

Stop:
```bash
docker-compose down
```

## Monitoring

### Health Check
```bash
curl http://localhost/
```

### Envoy Stats
```bash
curl http://localhost:9901/stats
```

### View Logs
```bash
docker-compose logs -f envoy
docker-compose logs -f govaguard
```

## Security Headers

Envoy automatically adds:
- `X-Frame-Options: DENY`
- `X-Content-Type-Options: nosniff`
- `X-XSS-Protection: 1; mode=block`
- `Strict-Transport-Security: max-age=31536000`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy: geolocation=(), microphone=(), camera=()`

## Architecture

```
Internet → Envoy (Port 80) → GovaGuard App (Port 8080)
              ↓
        Admin (Port 9901)
```

## Troubleshooting

**Port 80 already in use?**
```bash
# Change port in .env
HTTP_PORT=8080
```

**View container status:**
```bash
docker-compose ps
```

**Check Envoy config:**
```bash
docker run --rm -v $(pwd)/envoy.yaml:/envoy.yaml \
  envoyproxy/envoy:v1.29-latest \
  --mode validate -c /envoy.yaml
```

## Update

```bash
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```
