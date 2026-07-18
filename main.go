package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

const (
	contactWorkflowURL = "https://govaguard.app.n8n.cloud/webhook/contact"
)

//go:embed templates/* static/* whitepapers
var content embed.FS

type PageData struct {
	Title        string
	SiteDomain   string
	ContactEmail string
}

type AppConfig struct {
	PrimaryDomain string
	DomainAliases map[string]struct{}
	ContactEmail  string
}

type ContactForm struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"e_mail"`
	Company   string `json:"company"`
	Message   string `json:"message"`
}

func main() {
	config := loadConfig()

	// Parse templates
	tmpl := template.Must(template.ParseFS(content, "templates/*.html"))

	// Serve static files
	http.Handle("/static/", http.FileServer(http.FS(content)))

	// Serve whitepaper files
	http.Handle("/whitepapers/", http.FileServer(http.FS(content)))

	// Landing page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := config.pageData(r, "GovaGuard — CTO & CISO Consulting")
		tmpl.ExecuteTemplate(w, "index.html", data)
	})

	// Imprint page
	http.HandleFunc("/imprint", func(w http.ResponseWriter, r *http.Request) {
		data := config.pageData(r, "Imprint - GovaGuard")
		tmpl.ExecuteTemplate(w, "imprint.html", data)
	})

	// Privacy Policy page
	http.HandleFunc("/privacy", func(w http.ResponseWriter, r *http.Request) {
		data := config.pageData(r, "Privacy Policy - GovaGuard")
		tmpl.ExecuteTemplate(w, "privacy.html", data)
	})

	// Whitepapers page
	http.HandleFunc("/whitepapers", func(w http.ResponseWriter, r *http.Request) {
		data := config.pageData(r, "Whitepapers & Research - GovaGuard")
		tmpl.ExecuteTemplate(w, "whitepapers.html", data)
	})

	// HTMX endpoint for contact form
	http.HandleFunc("/contact", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte(`
					<div class="error-message">
						<div class="error-icon">✗</div>
						<p>Failed to process form. Please try again.</p>
					</div>
				`))
				return
			}

			// Extract form data
			form := ContactForm{
				FirstName: strings.TrimSpace(r.FormValue("first_name")),
				LastName:  strings.TrimSpace(r.FormValue("last_name")),
				Email:     strings.TrimSpace(r.FormValue("email")),
				Company:   strings.TrimSpace(r.FormValue("company")),
				Message:   strings.TrimSpace(r.FormValue("message")),
			}

			// Validate required fields
			if form.FirstName == "" || form.LastName == "" || form.Email == "" || form.Message == "" {
				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte(`
					<div class="error-message">
						<div class="error-icon">✗</div>
						<p>Please fill in all required fields.</p>
					</div>
				`))
				return
			}

			var buf bytes.Buffer
			err := json.NewEncoder(&buf).Encode(form)
			if err != nil {
				log.Print(err)
			}

			client := &http.Client{}
			request, err := http.NewRequest(http.MethodPost, contactWorkflowURL, &buf)
			if err != nil {
				log.Print(err)
			}

			request.Header.Set("Content-Type", "application/json")
			request.SetBasicAuth("user", "user")
			response, err := client.Do(request)
			if err != nil {
				log.Print(err)
			}

			defer response.Body.Close()

			body, err := io.ReadAll(response.Body)
			if err != nil {
				log.Print(err)
			}

			if response.StatusCode != http.StatusOK {
				log.Print(string(body), response.StatusCode)
			}

			// Success response
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`
				<div class="success-message">
					<div class="success-icon">✓</div>
					<p>Message received. We'll respond within 24 hours.</p>
				</div>
			`))
		}
	})

	log.Println("GovaGuard landing page running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func loadConfig() AppConfig {
	primaryDomain := envOrDefault("SITE_PRIMARY_DOMAIN", "govaguard.com")
	aliases := map[string]struct{}{
		normalizeHost(primaryDomain): {},
	}

	if rawAliases := strings.TrimSpace(os.Getenv("SITE_DOMAIN_ALIASES")); rawAliases != "" {
		for _, alias := range strings.Split(rawAliases, ",") {
			trimmedAlias := strings.TrimSpace(alias)
			normalizedAlias := normalizeHost(trimmedAlias)
			if normalizedAlias == "" {
				if trimmedAlias != "" {
					log.Printf("ignoring invalid SITE_DOMAIN_ALIASES entry: %q", trimmedAlias)
				}
				continue
			}

			aliases[normalizedAlias] = struct{}{}
		}
	}

	return AppConfig{
		PrimaryDomain: normalizeHost(primaryDomain),
		DomainAliases: aliases,
		ContactEmail:  envOrDefault("CONTACT_EMAIL", "hello@govaguard.com"),
	}
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	host = strings.TrimSuffix(host, ".")

	normalizedHost, _, err := net.SplitHostPort(host)
	if err == nil {
		host = normalizedHost
	}

	return host
}

func (c AppConfig) pageData(r *http.Request, title string) PageData {
	return PageData{
		Title:        title,
		SiteDomain:   c.siteDomainForRequest(r),
		ContactEmail: c.ContactEmail,
	}
}

func (c AppConfig) siteDomainForRequest(r *http.Request) string {
	host := normalizeHost(r.Host)
	if isLocalHost(host) {
		return c.PrimaryDomain
	}

	if _, ok := c.DomainAliases[host]; ok {
		return host
	}

	return c.PrimaryDomain
}

func isLocalHost(host string) bool {
	if host == "" || host == "localhost" {
		return true
	}

	hostWithoutZone := strings.SplitN(host, "%", 2)[0]
	ipPart := strings.Trim(hostWithoutZone, "[]")
	ip := net.ParseIP(ipPart)
	return ip != nil && ip.IsLoopback()
}
