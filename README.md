# GovaGuard Landing Page

A distinctive, production-grade landing page for GovaGuard CTO & CISO consulting services, built with Go and htmx.

## Design Concept

**Aesthetic**: Refined Brutalism + Terminal Security

The design combines the serious, high-stakes nature of executive security consulting with a modern technical aesthetic. Key features:

- **Typography**: JetBrains Mono for technical authority, IBM Plex Sans for clarity
- **Color Palette**: Deep navy base with electric cyan accents, creating a "secure terminal" vibe
- **Motion**: Subtle scanline overlays, typewriter animations, and scroll-triggered reveals
- **Layout**: Asymmetric grid with deliberate whitespace and angular compositions

## Features

- **Responsive Design**: Mobile-first approach with breakpoints for all devices
- **htmx Integration**: Dynamic form submission without full page reloads
- **Smooth Animations**: Intersection Observer API for scroll-triggered animations
- **Terminal Effects**: Animated terminal display showcasing security scanning
- **Contact Form**: Secure contact form with htmx-powered submission
- **SEO Optimized**: Semantic HTML with proper meta tags

## Tech Stack

- **Backend**: Go 1.22+ with embed for static files
- **Frontend**: Vanilla HTML/CSS/JS
- **Interactivity**: htmx 1.9.10
- **Fonts**: Google Fonts (JetBrains Mono, IBM Plex Sans)

## Installation & Running

### Local Development

```bash
# Navigate to project directory
cd govaguard

# Run the server
go run main.go
```

Visit `http://localhost:8080` to view the landing page.

### Docker Deployment

```bash
# Build the Docker image
docker build -t govaguard .

# Run the container
docker run -p 8080:8080 govaguard
```

Or use the pre-built image from GitHub Container Registry:

```bash
docker pull ghcr.io/[YOUR-USERNAME]/govaguard:latest
docker run -p 8080:8080 ghcr.io/[YOUR-USERNAME]/govaguard:latest
```

See [DOCKER.md](DOCKER.md) for complete Docker deployment guide.

## Project Structure

```
govaguard/
├── main.go                 # Go server with htmx endpoints
├── templates/
│   └── index.html         # Main landing page template
├── static/
│   ├── css/
│   │   └── style.css      # Complete styling with animations
│   └── js/
│       └── main.js        # Interactive enhancements
├── go.mod                 # Go module definition
└── README.md             # This file
```

## Key Sections

1. **Hero**: Executive-level messaging with animated grid background
2. **Services**: Four core capabilities (CTO Advisory, CISO Consulting, Threat Intelligence, Infrastructure Hardening)
3. **Expertise**: Domain mastery with animated terminal display
4. **Contact**: Secure contact form with htmx-powered submission
5. **Footer**: Company information and navigation

## Customization

### Colors
Edit CSS variables in `static/css/style.css`:
```css
:root {
    --color-bg: #0a0e27;
    --color-primary: #00d9ff;
    --color-accent: #00ff88;
    /* ... */
}
```

### Content
Update section content in `templates/index.html`

### Form Endpoint
Modify the `/contact` handler in `main.go` to integrate with your email service or CRM

## Production Deployment

### With Docker Compose + Envoy (Recommended)

```bash
cd deployment
docker-compose up -d
```

Access at http://localhost - Envoy proxy handles security headers, compression, and health checks.

See [deployment/README.md](deployment/README.md) for details.

### Manual Deployment

1. Build the binary: `go build -o govaguard main.go`
2. Run on your server: `./govaguard`
3. Configure reverse proxy (nginx/Caddy) for HTTPS
4. Set up proper email handling for contact form

## Browser Support

- Chrome/Edge (latest)
- Firefox (latest)
- Safari (latest)
- Mobile browsers (iOS Safari, Chrome Mobile)

## License

Proprietary - GovaGuard © 2026
