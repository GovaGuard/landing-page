# Testing GovaGuard with govaguard.com Domain

Since the DNS for govaguard.com is not yet configured, you can test the application locally using the `Host` header with curl.

## ✅ Services Running

- **GovaGuard App**: Running on internal port 8080
- **Envoy Proxy**: Listening on ports 80, 443, and 9901
- **Network**: Both services on `govaguard-network` bridge

## Testing with curl

### HTTP (Port 80)

```bash
# Full page
curl -H "Host: govaguard.com" http://localhost/

# Headers only (shows security headers)
curl -I -H "Host: govaguard.com" http://localhost/

# Imprint page
curl -H "Host: govaguard.com" http://localhost/imprint
```

### HTTPS (Port 443)

```bash
# Note: No SSL certificates configured yet, but port responds
curl -H "Host: govaguard.com" http://localhost:443/
```

### Envoy Admin Interface

```bash
# Health check
curl http://localhost:9901/ready

# Statistics
curl http://localhost:9901/stats

# Configuration
curl http://localhost:9901/config_dump
```

## Security Headers Verified ✅

All responses include these security headers:

```
x-frame-options: DENY
x-content-type-options: nosniff
x-xss-protection: 1; mode=block
strict-transport-security: max-age=31536000; includeSubDomains
referrer-policy: strict-origin-when-cross-origin
permissions-policy: geolocation=(), microphone=(), camera=()
server: envoy
```

## Testing from External Machines

Once you deploy to a server with a public IP, you can test from any machine:

```bash
# Replace SERVER_IP with your server's IP address
curl -H "Host: govaguard.com" http://SERVER_IP/

# Example
curl -H "Host: govaguard.com" http://203.0.113.10/
```

## When DNS is Configured

Once you point govaguard.com to your server's IP, you can test directly:

```bash
# No Host header needed
curl http://govaguard.com/
curl https://govaguard.com/  # After SSL is configured
```

## Container Status

Check container health:

```bash
docker-compose ps

# Expected output:
# govaguard-app    Up
# govaguard-envoy  Up (healthy)
```

View logs:

```bash
# All logs
docker-compose logs -f

# Specific service
docker-compose logs -f envoy
docker-compose logs -f govaguard
```

## Port Mappings

- **80** → Envoy HTTP listener → GovaGuard:8080
- **443** → Envoy HTTPS listener → GovaGuard:8080
- **9901** → Envoy admin interface

## Quick Health Checks

```bash
# Test application
curl -s -H "Host: govaguard.com" http://localhost/ | grep -o "GOVAGUARD" && echo "✅ App working"

# Test Envoy
curl -s http://localhost:9901/ready | grep "LIVE" && echo "✅ Envoy ready"

# Test security headers
curl -I -H "Host: govaguard.com" http://localhost/ | grep "x-frame-options" && echo "✅ Security headers active"
```

## Testing Compression

```bash
# Request with gzip support
curl -H "Host: govaguard.com" -H "Accept-Encoding: gzip" -I http://localhost/ | grep -i "content-encoding"

# Should show: content-encoding: gzip
```

## Next Steps

1. **Configure DNS**: Point govaguard.com A record to your server IP
2. **Add SSL Certificate**: Place certificates in `certs/` folder and uncomment TLS config in `envoy.yaml`
3. **Test from external network**: Verify accessibility from internet
4. **Monitor**: Use Envoy admin interface at `:9901` for metrics
